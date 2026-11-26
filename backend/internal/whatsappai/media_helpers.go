package whatsappai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
)

const (
	maxAudioBytes        = 20 * 1024 * 1024
	maxImageBytes        = 8 * 1024 * 1024
	maxPDFBytes          = 12 * 1024 * 1024
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
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := openAIHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", &openAIHTTPError{StatusCode: resp.StatusCode, Body: string(raw)}
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

func shouldAttachMedia(m *domain.WhatsAppMessage, isNew bool) bool {
	if m == nil || m.MediaStatus != domain.MediaStatusReady || m.MediaObjectKey == "" {
		return false
	}
	if isNew {
		return true
	}
	// Se uma imagem/PDF ficou pronta depois do ciclo anterior (ainda sem resumo salvo),
	// reanexa dentro da janela de 6h para não perder comprovantes atrasados.
	return strings.TrimSpace(m.AITranscript) == "" && !m.CreatedAt.IsZero() && time.Since(m.CreatedAt) <= 6*time.Hour
}
