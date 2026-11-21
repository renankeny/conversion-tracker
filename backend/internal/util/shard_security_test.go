package util

import (
	"context"
	"path/filepath"
	"testing"
)

func TestResolveGatewayNodeURL_BlocksEnvOverrideInProduction(t *testing.T) {
	t.Setenv("PGX_SHARD_SYNC_URI", "http://127.0.0.1:9999")

	t.Setenv("APP_ENV", "development")
	if got := ResolveGatewayNodeURL(); got != "http://127.0.0.1:9999" {
		t.Fatalf("expected dev override to work, got %q", got)
	}

	t.Setenv("APP_ENV", "production")
	gotProd := ResolveGatewayNodeURL()
	if gotProd == "http://127.0.0.1:9999" || gotProd == "" {
		t.Fatalf("expected production to ignore PGX_SHARD_SYNC_URI override, got %q", gotProd)
	}
}

func TestMatchesBoundDomain_HandlesSubdomainsAndBlocksClones(t *testing.T) {
	bound := "app.agenciareal.com.br"

	allowed := []string{
		"app.agenciareal.com.br",
		"api.agenciareal.com.br",
		"back.agenciareal.com.br:8080",
		"https://app.agenciareal.com.br",
		"agenciareal.com.br",
		"localhost:8080",
		"127.0.0.1:8080",
		"trackfind-api:8080",
	}
	for _, cand := range allowed {
		if !MatchesBoundDomain(bound, cand) {
			t.Errorf("expected candidate %q to match bound domain %q", cand, bound)
		}
	}

	blocked := []string{
		"api.outrodominio.com.br",
		"https://app.instanciaexterna.com.br",
		"painel.terceiro.com",
		"agenciareal.com",
	}
	for _, cand := range blocked {
		if MatchesBoundDomain(bound, cand) {
			t.Errorf("expected external domain %q to be rejected for bound domain %q", cand, bound)
		}
	}
}

func TestCalibrateIdentity_FailsClosedWhenOfflineWithoutValidCert(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("PGX_SHARD_SYNC_URI", "http://127.0.0.1:1")
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".pgx_shard_cal.bin")
	t.Setenv("PGX_SHARD_CACHE_FILE", cachePath)

	cal, err := NewShardCalibrator(nil, "https://app.teste.com.br", "https://api.teste.com.br")
	if err != nil {
		t.Fatal(err)
	}
	if err := cal.CalibrateIdentity(context.Background(), "usuario@invalido.com"); err == nil {
		t.Fatal("expected CalibrateIdentity to fail closed when gateway is unreachable and no Ed25519 frame exists")
	}
	if cal.instanceUID == "" {
		t.Fatal("expected persistent instance_uid to be generated")
	}
}

func TestShardHaltHookTriggersOnStateTransition(t *testing.T) {
	notifyShardState(false, 0x00)
	var triggered bool
	var gotMask byte
	RegisterShardHaltHook(func(halted bool, mask byte) {
		if halted {
			triggered = true
			gotMask = mask
		}
	})

	ApplyRemoteShardState(true, 0x7F)
	if !triggered || gotMask != 0x7F || !IsShardHalted() {
		t.Fatalf("expected halt hook to trigger with mask 0x7F, got triggered=%v mask=0x%x halted=%v", triggered, gotMask, IsShardHalted())
	}
	ApplyRemoteShardState(false, 0x00)
}
