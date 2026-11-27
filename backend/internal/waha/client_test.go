package waha

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthCheckAndSessionLifecycle(t *testing.T) {
	var createdWebhook string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "secret-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/server/version":
			_, _ = w.Write([]byte(`{"version":"2026.6.1","engine":"GOWS","tier":"PLUS"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/sessions":
			_, _ = w.Write([]byte(`[{"name":"cli-1","status":"WORKING"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/sessions":
			createdWebhook = "ok"
			_, _ = w.Write([]byte(`{"name":"cli-2","status":"STARTING"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/cli-2/auth/qr":
			b64 := base64.StdEncoding.EncodeToString([]byte("fakepng"))
			_, _ = w.Write([]byte(`{"mimetype":"image/png","data":"` + b64 + `"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/cli-2/auth/request-code":
			_, _ = w.Write([]byte(`{"code":"ABCD-1234"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/cli-2/lids/12345@lid":
			_, _ = w.Write([]byte(`{"lid":"12345@lid","pn":"5511999998888@c.us"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/sendText":
			_, _ = w.Write([]byte(`{"id":"true_5511999998888@c.us_ABC"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cli := NewClient(srv.URL, "secret-key")
	ctx := context.Background()

	h, err := cli.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if h.Version != "2026.6.1" || h.Engine != "GOWS" || h.InstanceCount != 1 {
		t.Fatalf("unexpected health: %+v", h)
	}

	sess, err := cli.CreateAndStartSession(ctx, "cli-2", "https://api.example/webhooks/waha/sec")
	if err != nil || createdWebhook != "ok" || sess.Name != "cli-2" {
		t.Fatalf("CreateAndStartSession failed: sess=%+v err=%v", sess, err)
	}

	qr, err := cli.GetQRCode(ctx, "cli-2")
	if err != nil || !strings.HasPrefix(qr, "data:image/png;base64,") {
		t.Fatalf("GetQRCode failed: qr=%q err=%v", qr, err)
	}

	code, err := cli.RequestPairingCode(ctx, "cli-2", "+55 (11) 99999-8888")
	if err != nil || code != "ABCD-1234" {
		t.Fatalf("RequestPairingCode failed: code=%q err=%v", code, err)
	}

	if pn := cli.ResolveLID(ctx, "cli-2", "12345@lid"); pn != "5511999998888@c.us" {
		t.Fatalf("ResolveLID failed: got %q", pn)
	}

	if err := cli.SendText(ctx, "cli-2", "+5511999998888", "Alerta teste"); err != nil {
		t.Fatalf("SendText failed: %v", err)
	}
}

func TestDownloadMediaRewriteAuthAndFallback(t *testing.T) {
	var externalSawKey bool
	s3Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "" {
			externalSawKey = true
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("s3-jpeg-bytes"))
	}))
	defer s3Srv.Close()

	wahaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "waha-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/files/s1/img1.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("local-waha-img"))
		case "/api/s1/chats/5511999@c.us/messages/msg2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"media":{"url":"http://localhost:3000/api/files/s1/img1.jpg","mimetype":"image/jpeg"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer wahaSrv.Close()

	cli := NewClient(wahaSrv.URL, "waha-key")
	ctx := context.Background()

	// 1. Rewrite localhost:3000 -> wahaSrv + envia X-Api-Key
	data, mime, err := cli.DownloadMedia(ctx, "s1", "http://localhost:3000/api/files/s1/img1.jpg", "", "", 1024)
	if err != nil || string(data) != "local-waha-img" || mime != "image/jpeg" {
		t.Fatalf("localhost rewrite failed: data=%q mime=%q err=%v", string(data), mime, err)
	}

	// 2. Host externo S3 não recebe X-Api-Key
	data2, _, err := cli.DownloadMedia(ctx, "s1", s3Srv.URL+"/bucket/pic.jpg", "", "", 1024)
	if err != nil || string(data2) != "s3-jpeg-bytes" || externalSawKey {
		t.Fatalf("s3 download failed: data=%q sawKey=%v err=%v", string(data2), externalSawKey, err)
	}

	// 3. Fallback sob demanda quando mediaURL vem vazio
	data3, mime3, err := cli.DownloadMedia(ctx, "s1", "", "5511999@c.us", "msg2", 1024)
	if err != nil || string(data3) != "local-waha-img" || mime3 != "image/jpeg" {
		t.Fatalf("on-demand fallback failed: data=%q mime=%q err=%v", string(data3), mime3, err)
	}

	// 4. Respeita limite maxBytes
	_, _, err = cli.DownloadMedia(ctx, "s1", "http://localhost:3000/api/files/s1/img1.jpg", "", "", 4)
	if !errors.Is(err, ErrMediaTooLarge) {
		t.Fatalf("expected ErrMediaTooLarge, got %v", err)
	}
}
