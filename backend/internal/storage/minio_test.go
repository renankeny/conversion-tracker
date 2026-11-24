package storage

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestBuildMinIOEndpointAndTransport_PathStyleForAllHosts(t *testing.T) {
	for _, raw := range []string{"trackeado-minio:9000", "https://s3.meudominio.com.br"} {
		ep, tr, lookup, err := buildMinIOEndpointAndTransport(raw, strings.HasPrefix(raw, "https://"))
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", raw, err)
		}
		if ep == "" {
			t.Fatalf("empty endpoint for %q", raw)
		}
		if tr != nil {
			t.Fatalf("expected nil custom transport when no underscore is present (%q)", raw)
		}
		if lookup != minio.BucketLookupPath {
			t.Fatalf("got lookup %v for %q, want BucketLookupPath", lookup, raw)
		}
	}
}

func TestPutMedia_LegacyMinIOCompatibilityNoTrailingCRC(t *testing.T) {
	var gotMD5, gotCRC32C, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("location") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`))
			return
		}
		if r.Method == http.MethodPut {
			gotMD5 = r.Header.Get("Content-Md5")
			gotCRC32C = r.Header.Get("X-Amz-Checksum-Crc32c")
			gotPath = r.URL.Path
			w.Header().Set("ETag", `"d41d8cd98f00b204e9800998ecf8427e"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ep := strings.TrimPrefix(srv.URL, "http://")
	cli, err := minio.New(ep, &minio.Options{
		Creds:           credentials.NewStaticV4("old_minio_key", "old_minio_secret", ""),
		Secure:          false,
		BucketLookup:    minio.BucketLookupPath,
		TrailingHeaders: false,
	})
	if err != nil {
		t.Fatalf("minio.New: %v", err)
	}
	ms := &MediaStorage{client: cli, bucket: "whatsapp-media"}
	payload := []byte("audio-opus-bytes")
	n, err := ms.PutMedia(context.Background(), "accounts/1/chats/2/msg_3.ogg", bytes.NewReader(payload), int64(len(payload)), "audio/ogg")
	if err != nil {
		t.Fatalf("PutMedia failed: %v", err)
	}
	if n != int64(len(payload)) {
		t.Fatalf("expected %d bytes written, got %d", len(payload), n)
	}
	if gotMD5 == "" {
		t.Fatalf("expected classic Content-MD5 header for legacy MinIO compatibility")
	}
	if gotCRC32C != "" {
		t.Fatalf("expected no X-Amz-Checksum-Crc32c header, got %q", gotCRC32C)
	}
	if gotPath != "/whatsapp-media/accounts/1/chats/2/msg_3.ogg" {
		t.Fatalf("expected path-style upload URL, got %q", gotPath)
	}
}

func TestBuildMinIOEndpointAndTransport_EasyPanelUnderscoreHost(t *testing.T) {
	var receivedHost string
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		receivedPath = r.URL.Path
		if strings.Contains(r.Host, "_") {
			http.Error(w, "Invalid Request (invalid hostname)", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Has("location") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatalf("split server port: %v", err)
	}

	ep, tr, lookup, err := buildMinIOEndpointAndTransport("trackeado_minio-trackeado:"+port, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep != "trackeado-minio-trackeado:"+port {
		t.Fatalf("got sanitized endpoint %q, want %q", ep, "trackeado-minio-trackeado:"+port)
	}
	if lookup != minio.BucketLookupPath {
		t.Fatalf("got lookup %v, want BucketLookupPath", lookup)
	}

	httpTr := tr.(*http.Transport)
	origDial := httpTr.DialContext
	var dialedTarget string
	httpTr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == "trackeado-minio-trackeado:"+port {
			dialedTarget = "trackeado_minio-trackeado:" + port
			return (&net.Dialer{}).DialContext(ctx, network, "127.0.0.1:"+port)
		}
		return origDial(ctx, network, addr)
	}

	cli, err := minio.New(ep, &minio.Options{
		Creds:        credentials.NewStaticV4("minio_test", "minio_secret_12345", ""),
		Secure:       false,
		BucketLookup: lookup,
		Transport:    httpTr,
	})
	if err != nil {
		t.Fatalf("minio.New failed: %v", err)
	}

	ms := &MediaStorage{client: cli, bucket: "whatsapp-media"}
	if err := ms.EnsureBucket(context.Background(), 0); err != nil {
		t.Fatalf("EnsureBucket failed: %v", err)
	}

	if dialedTarget != "trackeado_minio-trackeado:"+port {
		t.Fatalf("expected dial target to map to underscore service host, got %q", dialedTarget)
	}
	if strings.Contains(receivedHost, "_") || receivedHost != "trackeado-minio-trackeado:"+port {
		t.Fatalf("expected sanitized Host header without underscore, got %q", receivedHost)
	}
	if !strings.HasPrefix(receivedPath, "/whatsapp-media") {
		t.Fatalf("expected path-style bucket lookup, got %q", receivedPath)
	}
}
