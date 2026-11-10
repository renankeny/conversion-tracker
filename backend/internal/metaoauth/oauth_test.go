package metaoauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestMetaOAuthEncryptedGatewayRoundTrip(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			Vector string `json:"v"`
		}
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			t.Fatalf("decode env: %v", err)
		}
		switch r.URL.Path {
		case "/api/v1/meta-ads/oauth/start":
			var req map[string]any
			if err := openGatewayJSON(env.Vector, &req); err != nil {
				t.Fatalf("open start req: %v", err)
			}
			sealed, _ := sealGatewayJSON(map[string]string{
				"url":   "https://www.facebook.com/v26.0/dialog/oauth?state=abc",
				"state": "nonce123",
			})
			_ = json.NewEncoder(w).Encode(map[string]string{"v": sealed})
		case "/api/v1/meta-ads/proxy":
			var req map[string]any
			if err := openGatewayJSON(env.Vector, &req); err != nil {
				t.Fatalf("open proxy req: %v", err)
			}
			sealed, _ := sealGatewayJSON(AdMetadata{
				AdID:         "120200001",
				AdName:       "Criativo Video 01",
				CampaignID:   "120100001",
				CampaignName: "Campanha CTWA Leads",
				AdsetID:      "120150001",
				AdsetName:    "Conjunto Aberto",
				PageID:       "99887766",
			})
			_ = json.NewEncoder(w).Encode(map[string]string{"v": sealed})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	_ = os.Setenv("PGX_SHARD_SYNC_URI", "https://gateway.local")
	defer os.Unsetenv("PGX_SHARD_SYNC_URI")

	mockHTTP := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec.Result(), nil
		}),
	}

	cfg := Config{
		AppURL:       "https://s.cliente.com.br",
		PublicAPIURL: "https://back.cliente.com.br/api/v1",
		HTTPClient:   mockHTTP,
	}
	u, err := cfg.StartCentralOAuth(context.Background(), 7, "nonce123")
	if err != nil || u == "" {
		t.Fatalf("StartCentralOAuth failed: %v", err)
	}

	rawTicket, _ := sealGatewayJSON(CallbackTicket{
		AccountID: 7, StateNonce: "nonce123", VaultToken: "meta_vault_v1:xyz", Name: "Gestor", Email: "gestor@meta.com",
	})
	tk, err := cfg.OpenCallbackTicket(rawTicket)
	if err != nil || tk.VaultToken != "meta_vault_v1:xyz" {
		t.Fatalf("OpenCallbackTicket failed: %v", err)
	}

	cli := NewClient(cfg)
	metaAcc := &domain.MetaAccount{ID: 7, VaultTokenEncrypted: "meta_vault_v1:xyz", AdAccountID: "act_12345"}
	adMeta, err := cli.GetAdMetadata(context.Background(), metaAcc, "120200001")
	if err != nil {
		t.Fatalf("GetAdMetadata failed: %v", err)
	}
	if adMeta.CampaignName != "Campanha CTWA Leads" || adMeta.PageID != "99887766" {
		t.Fatalf("unexpected metadata: %+v", adMeta)
	}
}
