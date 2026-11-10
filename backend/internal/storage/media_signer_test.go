package storage

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/config"
)

func TestSignAndVerifyMediaURL(t *testing.T) {
	secret := "super-secret-jwt-key-for-hmac-test"
	signed := SignMediaURL("https://api.example.com/api/v1/", secret, 7, 421, 1*time.Hour)

	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("invalid signed url: %v", err)
	}
	acc, _ := strconv.ParseInt(u.Query().Get("acc"), 10, 64)
	exp, _ := strconv.ParseInt(u.Query().Get("exp"), 10, 64)
	sig := u.Query().Get("sig")

	if !VerifyMediaSignature(secret, acc, 421, exp, sig, time.Now()) {
		t.Fatalf("expected valid signature")
	}
	// Tentativa de cross-tenant (outro account_id ou outro message_id) deve falhar
	if VerifyMediaSignature(secret, 8, 421, exp, sig, time.Now()) {
		t.Fatalf("expected cross-tenant account_id tamper to fail")
	}
	if VerifyMediaSignature(secret, 7, 422, exp, sig, time.Now()) {
		t.Fatalf("expected message_id tamper to fail")
	}
	// Expirado após TTL
	if VerifyMediaSignature(secret, 7, 421, exp, sig, time.Now().Add(2*time.Hour)) {
		t.Fatalf("expected expired signature to fail")
	}
}

func TestEasyPanelConfigNormalization(t *testing.T) {
	ep, ssl := config.ParseMinIOEndpoint("https://s3.cliente.com.br/", "false")
	if ep != "s3.cliente.com.br" || !ssl {
		t.Fatalf("expected s3.cliente.com.br with ssl=true, got %q %v", ep, ssl)
	}
	ep2, ssl2 := config.ParseMinIOEndpoint("http://trackeado_minio:9000/", "")
	if ep2 != "trackeado_minio:9000" || ssl2 {
		t.Fatalf("expected trackeado_minio:9000 with ssl=false, got %q %v", ep2, ssl2)
	}
	ep3, ssl3 := config.ParseMinIOEndpoint("projeto_minio:9000", "false")
	if ep3 != "projeto_minio:9000" || ssl3 {
		t.Fatalf("expected projeto_minio:9000 with ssl=false, got %q %v", ep3, ssl3)
	}

	cfg := &config.Config{}
	config.ApplyRedisURL(cfg, "redis://default:minhasenha@projeto_redis:6379/2")
	if cfg.RedisAddr != "projeto_redis:6379" || cfg.RedisUsername != "default" || cfg.RedisPassword != "minhasenha" || cfg.RedisDB != 2 {
		t.Fatalf("unexpected redis url parse: %+v", cfg)
	}
}

