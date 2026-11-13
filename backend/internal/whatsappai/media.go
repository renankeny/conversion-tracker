package whatsappai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
)

const (
	maxAudioBytes = 20 * 1024 * 1024
	maxImageBytes = 8 * 1024 * 1024
	maxPDFBytes   = 12 * 1024 * 1024

	maxImagesPerAnalysis = 6
	maxPDFsPerAnalysis   = 4
)

func readMinIOBytes(ctx context.Context, st *storage.MediaStorage, key string, maxBytes int64) ([]byte, string, error) {
	if st == nil || !st.Enabled() || strings.TrimSpace(key) == "" {
		return nil, "", fmt.Errorf("media storage unavailable")
	}
	obj, info, err := st.OpenObject(ctx, key)
	if err != nil {
		return nil, "", err
	}
	defer obj.Close()
	if info.Size > maxBytes {
		return nil, "", fmt.Errorf("media exceeds max size (%d > %d)", info.Size, maxBytes)
	}
	data, err := io.ReadAll(io.LimitReader(obj, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxBytes {
		return nil, "", fmt.Errorf("media exceeds max size")
	}
	return data, info.ContentType, nil
}

func audioFilenameForMime(mimeStr, origName string) string {
	m := strings.ToLower(strings.TrimSpace(mimeStr))
	switch {
	case strings.Contains(m, "ogg") || strings.Contains(m, "opus"):
		return "audio.ogg"
	case strings.Contains(m, "mpeg") || strings.Contains(m, "mp3"):
		return "audio.mp3"
	case strings.Contains(m, "mp4") || strings.Contains(m, "m4a") || strings.Contains(m, "aac"):
		return "audio.m4a"
	case strings.Contains(m, "wav"):
		return "audio.wav"
	case strings.Contains(m, "webm"):
		return "audio.webm"
	}
	ext := strings.ToLower(filepath.Ext(origName))
	switch ext {
	case ".ogg", ".oga", ".mp3", ".m4a", ".mp4", ".wav", ".webm":
		return "audio" + ext
	}
	return "audio.ogg"
}

func transcribeAudioOpenAI(ctx context.Context, apiKey string, audioBytes []byte, filename string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(audioBytes); err != nil {
		return "", err
	}
	_ = writer.WriteField("model", "whisper-1")
	_ = writer.WriteField("language", "pt")
	_ = writer.Close()

	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, "https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("whisper status %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Text), nil
}

func isPDFMessage(m *domain.WhatsAppMessage) bool {
	if m == nil {
		return false
	}
	if m.MediaMimeType != nil && strings.Contains(strings.ToLower(*m.MediaMimeType), "pdf") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(m.MediaFilename)), ".pdf") ||
		strings.HasSuffix(strings.ToLower(strings.TrimSpace(m.MediaObjectKey)), ".pdf")
}

func buildMultimodalUserParts(
	ctx context.Context,
	apiKey string,
	st *storage.MediaStorage,
	waRepo *repo.WhatsAppRepo,
	msgs []*domain.WhatsAppMessage,
	lastAnalyzedMsgID int64,
) []map[string]any {
	var transcriptLines strings.Builder
	transcriptLines.WriteString("HISTÓRICO CRONOLÓGICO DA CONVERSA NO WHATSAPP:\n\n")

	attachments := make([]map[string]any, 0, 6)
	imgCount := 0
	pdfCount := 0

	for _, m := range msgs {
		role := "LEAD"
		if m.IsFromMe {
			role = "ATENDENTE"
		}
		isNew := m.ID > lastAnalyzedMsgID
		tagNew := ""
		if isNew {
			tagNew = " [NOVA]"
		}
		header := fmt.Sprintf("[Msg #%d | %s | %s]%s: ", m.ID, m.Timestamp.Format("02/01 15:04"), role, tagNew)

		switch m.MessageType {
		case "video", "sticker":
			if strings.TrimSpace(m.Content) != "" {
				fmt.Fprintf(&transcriptLines, "%s[%s com legenda]: %s\n", header, m.MessageType, strings.TrimSpace(m.Content))
			}
		case "audio":
			text := strings.TrimSpace(m.AITranscript)
			if text == "" && m.MediaStatus == domain.MediaStatusReady && m.MediaObjectKey != "" {
				mimeVal := ""
				if m.MediaMimeType != nil {
					mimeVal = *m.MediaMimeType
				}
				if data, objMime, err := readMinIOBytes(ctx, st, m.MediaObjectKey, maxAudioBytes); err == nil && len(data) > 0 {
					fname := audioFilenameForMime(firstStr(mimeVal, objMime), m.MediaFilename)
					if tr, terr := transcribeAudioOpenAI(ctx, apiKey, data, fname); terr == nil && tr != "" {
						text = tr
						m.AITranscript = tr
						if waRepo != nil {
							_ = waRepo.SaveMessageAITranscript(ctx, m.ID, tr)
						}
					}
				}
			}
			if text != "" {
				fmt.Fprintf(&transcriptLines, "%s[Áudio transcrito]: %s\n", header, text)
			} else {
				fmt.Fprintf(&transcriptLines, "%s[Áudio enviado]\n", header)
			}
		case "image":
			caption := strings.TrimSpace(m.Content)
			if isNew && imgCount < maxImagesPerAnalysis && m.MediaStatus == domain.MediaStatusReady && m.MediaObjectKey != "" {
				if data, objMime, err := readMinIOBytes(ctx, st, m.MediaObjectKey, maxImageBytes); err == nil && len(data) > 0 {
					mimeType := normalizeImageMime(m.MediaMimeType, objMime)
					dataURL := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
					attachments = append(attachments,
						map[string]any{"type": "text", "text": fmt.Sprintf("Imagem anexada referente à Msg #%d (%s):", m.ID, role)},
						map[string]any{"type": "image_url", "image_url": map[string]any{"url": dataURL, "detail": "auto"}},
					)
					imgCount++
				}
			}
			desc := "[Imagem enviada"
			if m.AITranscript != "" && !isNew {
				desc += " — resumo anterior: " + m.AITranscript
			}
			desc += "]"
			if caption != "" {
				desc += " Legenda: " + caption
			}
			fmt.Fprintf(&transcriptLines, "%s%s\n", header, desc)
		case "document":
			caption := strings.TrimSpace(m.Content)
			if isPDFMessage(m) && isNew && pdfCount < maxPDFsPerAnalysis && m.MediaStatus == domain.MediaStatusReady && m.MediaObjectKey != "" {
				if data, _, err := readMinIOBytes(ctx, st, m.MediaObjectKey, maxPDFBytes); err == nil && len(data) > 0 {
					fname := strings.TrimSpace(m.MediaFilename)
					if fname == "" || !strings.HasSuffix(strings.ToLower(fname), ".pdf") {
						fname = fmt.Sprintf("documento_msg_%d.pdf", m.ID)
					}
					dataURL := "data:application/pdf;base64," + base64.StdEncoding.EncodeToString(data)
					attachments = append(attachments,
						map[string]any{"type": "text", "text": fmt.Sprintf("Documento PDF anexado referente à Msg #%d (%s, %s):", m.ID, role, fname)},
						map[string]any{"type": "file", "file": map[string]any{"filename": fname, "file_data": dataURL}},
					)
					pdfCount++
				}
			}
			desc := fmt.Sprintf("[Documento %s", firstStr(m.MediaFilename, "anexo"))
			if m.AITranscript != "" && !isNew {
				desc += " — resumo anterior: " + m.AITranscript
			}
			desc += "]"
			if caption != "" {
				desc += " Legenda: " + caption
			}
			fmt.Fprintf(&transcriptLines, "%s%s\n", header, desc)
		default:
			if strings.TrimSpace(m.Content) != "" {
				fmt.Fprintf(&transcriptLines, "%s%s\n", header, strings.TrimSpace(m.Content))
			}
		}
	}

	parts := make([]map[string]any, 0, 1+len(attachments))
	parts = append(parts, map[string]any{"type": "text", "text": transcriptLines.String()})
	parts = append(parts, attachments...)
	return parts
}

