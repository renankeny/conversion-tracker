package metaoauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func TestMetaLeaseCaches6hInRAMAndDispatchesDirectlyToMeta(t *testing.T) {
	InvalidateAllLeases()
	util.ApplyRemoteShardState(false, 0x00)

	var centralLeaseCalls atomic.Int32
	var directMetaCAPICalls atomic.Int32

	centralSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		centralLeaseCalls.Add(1)
		var env struct {
			Vector string `json:"v"`
		}
		_ = json.NewDecoder(r.Body).Decode(&env)
		var req map[string]any
		if err := openGatewayJSON(env.Vector, &req); err != nil {
			t.Errorf("failed to open request: %v", err)
		}
		if req["act"] != "issue_lease" {
			t.Errorf("expected action issue_lease, got %v", req["act"])
		}
		sealed, _ := sealGatewayJSON(metaLeasePayload{
			AccessToken:    "EAAB_DIRECT_LEASE_TOK",
			AppSecretProof: "proof_hmac_123456",
			GraphVersion:   "v26.0",
			LeaseTTLSec:    21600,
		})
		_ = json.NewEncoder(w).Encode(map[string]string{"v": sealed})
	}))
	defer centralSrv.Close()

	metaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		directMetaCAPICalls.Add(1)
		if r.URL.Query().Get("appsecret_proof") != "proof_hmac_123456" {
			t.Errorf("missing or invalid appsecret_proof: %s", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer EAAB_DIRECT_LEASE_TOK" {
			t.Errorf("invalid Authorization header: %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"events_received":1}`))
	}))
	defer metaSrv.Close()

	t.Setenv("APP_ENV", "development")
	t.Setenv("PGX_SHARD_SYNC_URI", centralSrv.URL)
	prevBase := metaDirectBaseURL
	metaDirectBaseURL = metaSrv.URL
	defer func() { metaDirectBaseURL = prevBase }()

	cli := NewClient(Config{AppURL: "https://app.cliente.com.br", PublicAPIURL: "https://api.cliente.com.br"})
	acc := &domain.MetaAccount{ID: 1, VaultTokenEncrypted: "meta_vault_v1:sample_vault_ref"}

	for i := 0; i < 25; i++ {
		code, body, err := cli.SendCAPIEvent(context.Background(), acc, "99887766", []byte(`{"data":[]}`))
		if err != nil || code != http.StatusOK {
			t.Fatalf("iter %d failed: code=%d body=%s err=%v", i, code, body, err)
		}
	}

	if centralLeaseCalls.Load() != 1 {
		t.Fatalf("expected central server to be called ONLY ONCE for 25 CAPI events, got %d", centralLeaseCalls.Load())
	}
	if directMetaCAPICalls.Load() != 25 {
		t.Fatalf("expected all 25 CAPI events to go directly to Meta, got %d", directMetaCAPICalls.Load())
	}

	// Simula bloqueio instantâneo por Push Reverso: limpa RAM e impede novos disparos.
	InvalidateAllLeases()
	util.ApplyRemoteShardState(true, 0x7F)
	defer util.ApplyRemoteShardState(false, 0x00)

	if _, _, err := cli.SendCAPIEvent(context.Background(), acc, "99887766", []byte(`{"data":[]}`)); err == nil {
		t.Fatal("expected SendCAPIEvent to be blocked immediately when shard is halted")
	}
}
