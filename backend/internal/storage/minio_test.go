package storage

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestBuildMinIOEndpointAndTransport_WithoutUnderscore(t *testing.T) {
	ep, tr, lookup, err := buildMinIOEndpointAndTransport("trackeado-minio:9000", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep != "trackeado-minio:9000" {
		t.Fatalf("got endpoint %q, want %q", ep, "trackeado-minio:9000")
	}
	if tr != nil {
		t.Fatalf("expected nil custom transport when no underscore is present")
	}
	if lookup != minio.BucketLookupAuto {
		t.Fatalf("got lookup %v, want BucketLookupAuto", lookup)
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

	// Em ambiente de teste sem DNS Docker, redireciona o dial real para 127.0.0.1:<port>
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
