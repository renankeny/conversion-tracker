package handlers

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/storage"
)

type WhatsAppMediaHandler struct {
	repo      *repo.WhatsAppRepo
	storage   *storage.MediaStorage
	jwtSecret string
}

func NewWhatsAppMediaHandler(repo *repo.WhatsAppRepo, st *storage.MediaStorage, jwtSecret string) *WhatsAppMediaHandler {
	return &WhatsAppMediaHandler{
		repo:      repo,
		storage:   st,
		jwtSecret: jwtSecret,
	}
}

// ServeHTTP transmite o arquivo de mídia do MinIO com suporte nativo a HTTP Range (206 Partial Content),
// exigindo assinatura HMAC-SHA256 válida e escopada ao account_id.
func (h *WhatsAppMediaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil || !h.storage.Enabled() {
		http.Error(w, "storage indisponível", http.StatusServiceUnavailable)
		return
	}

	msgID, err := strconv.ParseInt(chi.URLParam(r, "messageID"), 10, 64)
	if err != nil || msgID <= 0 {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	accID, _ := strconv.ParseInt(q.Get("acc"), 10, 64)
	expUnix, _ := strconv.ParseInt(q.Get("exp"), 10, 64)
	sig := q.Get("sig")

	if !storage.VerifyMediaSignature(h.jwtSecret, accID, msgID, expUnix, sig, time.Now()) {
		http.Error(w, "invalid or expired media signature", http.StatusForbidden)
		return
	}

	msg, err := h.repo.GetMessageForMediaStream(r.Context(), accID, msgID)
	if err != nil || msg == nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}
	if msg.MediaStatus == domain.MediaStatusExpired || msg.MediaObjectKey == "" {
		http.Error(w, "media expired", http.StatusGone)
		return
	}

	obj, stat, err := h.storage.OpenObject(r.Context(), msg.MediaObjectKey)
	if err != nil {
		http.Error(w, "object not found", http.StatusNotFound)
		return
	}
	defer obj.Close()

	contentType := stat.ContentType
	if msg.MediaMimeType != nil && *msg.MediaMimeType != "" {
		contentType = *msg.MediaMimeType
	}
	filename := strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r == '\r' || r == '\n' {
			return '_'
		}
		return r
	}, msg.MediaFilename)
	if filename == "" {
		filename = fmt.Sprintf("whatsapp_%d", msg.ID)
	}

	safeMime, inlineSafe := sanitizeServedMediaMime(contentType)
	w.Header().Set("Content-Type", safeMime)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; media-src 'self'; img-src 'self'; style-src 'unsafe-inline'; sandbox")
	if !inlineSafe {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	}
	http.ServeContent(w, r, filename, stat.LastModified, obj)
}

func sanitizeServedMediaMime(raw string) (string, bool) {
	base, _, err := mime.ParseMediaType(strings.TrimSpace(raw))
	if err != nil || base == "" {
		return "application/octet-stream", false
	}
	base = strings.ToLower(base)
	switch base {
	case "image/jpeg", "image/png", "image/webp", "image/gif",
		"video/mp4", "video/webm", "video/3gpp", "application/pdf":
		return raw, true
	}
	if strings.HasPrefix(base, "audio/") {
		return raw, true
	}
	return "application/octet-stream", false
}

