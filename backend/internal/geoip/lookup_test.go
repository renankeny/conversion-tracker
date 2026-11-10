package geoip

import (
	"os"
	"path/filepath"
	"testing"
)

// resolveTestDB sobe a árvore de diretórios procurando backend/data/GeoLite2-City.mmdb.
// Permite rodar o teste com `go test ./internal/geoip/...` sem hardcoding de path absoluto.
func resolveTestDB(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "data", "GeoLite2-City.mmdb")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func TestNew_EmptyPathReturnsNoopLookup(t *testing.T) {
	l, err := New("")
	if err != nil {
		t.Fatalf("New(\"\") returned err: %v", err)
	}
	defer l.Close()
	res, err := l.Resolve("8.8.8.8")
	if err != nil {
		t.Fatalf("Resolve on noop: %v", err)
	}
	if (res != Result{}) {
		t.Fatalf("expected empty result, got %+v", res)
	}
}

func TestResolve_KnownIPs(t *testing.T) {
	dbPath := resolveTestDB(t)
	if dbPath == "" {
		t.Skip("GeoLite2-City.mmdb not found under backend/data/, skipping live test")
	}
	l, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q): %v", dbPath, err)
	}
	defer l.Close()

	cases := []struct {
		name        string
		ip          string
		wantCountry string
	}{
		{"google_dns_us", "8.8.8.8", "US"},
		// 200.160.2.3 = NIC.br (registro.br). Garantia de país BR; cidade/estado
		// dependem da base. Uso aqui pra validar lookup realista do nosso domínio
		// principal (Brasil).
		{"nicbr_brazil", "200.160.2.3", "BR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := l.Resolve(tc.ip)
			if err != nil {
				t.Fatalf("Resolve(%s): %v", tc.ip, err)
			}
			if res.Country != tc.wantCountry {
				t.Fatalf("ip=%s: country=%q want=%q (full result %+v)",
					tc.ip, res.Country, tc.wantCountry, res)
			}
			t.Logf("ip=%s result=%+v", tc.ip, res)
		})
	}
}

func TestResolve_InvalidIP(t *testing.T) {
	dbPath := resolveTestDB(t)
	if dbPath == "" {
		t.Skip("GeoLite2-City.mmdb not found, skipping")
	}
	l, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()

	if _, err := l.Resolve("not-an-ip"); err == nil {
		t.Fatalf("expected error for invalid IP, got nil")
	}
}
