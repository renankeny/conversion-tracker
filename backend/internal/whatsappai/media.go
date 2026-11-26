package whatsappai

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
)

type multimodalBuild struct {
	Parts          []map[string]any
	NoPDFParts     []map[string]any
	HasPDF         bool
	RetryCursorCap int64
}

func buildMultimodalUserParts(
	ctx context.Context,
	apiKey string,
	st *storage.MediaStorage,
	waRepo *repo.WhatsAppRepo,
	msgs []*domain.WhatsAppMessage,
	lastAnalyzedMsgID int64,
) multimodalBuild {
	var transcriptLines strings.Builder
	transcriptLines.WriteString("HISTÓRICO CRONOLÓGICO DA CONVERSA NO WHATSAPP:\n\n")

	attachments := make([]map[string]any, 0, 6)
	noPDFAttachments := make([]map[string]any, 0, 6)
	imgCount, pdfCount := 0, 0
	var retryCursorCap int64

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
			} else if m.MessageType == "video" {
				fmt.Fprintf(&transcriptLines, "%s[Vídeo enviado]\n", header)
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
					} else if terr != nil {
						slog.Warn("whatsapp ai whisper transcription failed", "msg_id", m.ID, "error", terr)
						if isNew && (m.CreatedAt.IsZero() || time.Since(m.CreatedAt) <= 2*time.Hour) && (retryCursorCap == 0 || m.ID-1 < retryCursorCap) {
							retryCursorCap = m.ID - 1
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
			if imgCount < maxImagesPerAnalysis && shouldAttachMedia(m, isNew) {
				if data, objMime, err := readMinIOBytes(ctx, st, m.MediaObjectKey, maxImageBytes); err == nil && len(data) > 0 {
					dataURL := "data:" + normalizeImageMime(m.MediaMimeType, objMime) + ";base64," + base64.StdEncoding.EncodeToString(data)
					lbl := map[string]any{"type": "text", "text": fmt.Sprintf("Imagem anexada referente à Msg #%d (%s):", m.ID, role)}
					img := map[string]any{"type": "image_url", "image_url": map[string]any{"url": dataURL, "detail": "auto"}}
					attachments = append(attachments, lbl, img)
					noPDFAttachments = append(noPDFAttachments, lbl, img)
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
			if isPDFMessage(m) && pdfCount < maxPDFsPerAnalysis && shouldAttachMedia(m, isNew) {
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

	textPart := map[string]any{"type": "text", "text": transcriptLines.String()}
	parts := append([]map[string]any{textPart}, attachments...)
	noPDFParts := append([]map[string]any{textPart}, noPDFAttachments...)
	return multimodalBuild{
		Parts:          parts,
		NoPDFParts:     noPDFParts,
		HasPDF:         pdfCount > 0,
		RetryCursorCap: retryCursorCap,
	}
}
