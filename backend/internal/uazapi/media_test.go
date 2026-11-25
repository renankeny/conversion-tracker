package uazapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadMedia_StreamsFromFileURL(t *testing.T) {
	wantBytes := []byte("fake-mp3-audio-payload")
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/message/download":
			if r.Header.Get("token") != "tok-123" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var req struct {
				ID           string `json:"id"`
				ReturnBase64 bool   `json:"return_base64"`
				GenerateMP3  bool   `json:"generate_mp3"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.ID != "MSG_AUDIO_1" || req.ReturnBase64 || !req.GenerateMP3 {
				http.Error(w, "unexpected request body", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"fileURL":  srv.URL + "/files/audio1.mp3",
				"mimetype": "audio/mpeg",
			})
		case "/files/audio1.mp3":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(wantBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cli := New(srv.URL, "tok-123")
	data, mime, err := cli.DownloadMedia(context.Background(), "MSG_AUDIO_1", "5511:MSG_AUDIO_1", 1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(wantBytes) {
		t.Fatalf("unexpected media bytes: %q", string(data))
	}
	if mime != "audio/mpeg" {
		t.Fatalf("expected audio/mpeg preferred over octet-stream, got %q", mime)
	}
}

func TestDownloadMedia_FallsBackToRawIDAndBase64(t *testing.T) {
	wantBytes := []byte("%PDF-1.7-comprovante")
	b64 := base64.StdEncoding.EncodeToString(wantBytes)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID           string `json:"id"`
			ReturnBase64 bool   `json:"return_base64"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.ID == "SHORT_ID" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if req.ID == "5511:SHORT_ID" && !req.ReturnBase64 {
			_ = json.NewEncoder(w).Encode(map[string]any{"fileURL": ""})
			return
		}
		if req.ID == "5511:SHORT_ID" && req.ReturnBase64 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"mimetype":   "application/pdf",
				"base64Data": b64,
			})
			return
		}
		http.Error(w, "unexpected", http.StatusBadRequest)
	}))
	defer srv.Close()

	cli := New(srv.URL, "tok-xyz")
	data, mime, err := cli.DownloadMedia(context.Background(), "SHORT_ID", "5511:SHORT_ID", 1024*1024)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got err=%v", err)
	}
	if string(data) != string(wantBytes) || mime != "application/pdf" {
		t.Fatalf("unexpected fallback result: mime=%q data=%q", mime, string(data))
	}
}

func TestDownloadMedia_EnforcesMaxBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/message/download" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"fileURL":  "http://" + r.Host + "/files/big.mp4",
				"mimetype": "video/mp4",
			})
			return
		}
		_, _ = w.Write(make([]byte, 2048))
	}))
	defer srv.Close()

	cli := New(srv.URL, "tok-123")
	_, _, err := cli.DownloadMedia(context.Background(), "BIG1", "", 512)
	if !errors.Is(err, ErrMediaTooLarge) {
		t.Fatalf("expected ErrMediaTooLarge, got %v", err)
	}
}
