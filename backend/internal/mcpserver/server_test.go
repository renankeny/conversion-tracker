package mcpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

func buildStubRouter(t *testing.T) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	r.Get("/api/v1/admin/pixels", func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-API-Key") != "trk_test_valid" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":10,"name":"clinica-prime","enabled":true}]}`))
	})
	r.Get("/api/v1/admin/ext-api/crm/stages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"stages":[{"id":1,"name":"Novo Lead"},{"id":2,"name":"Venda Realizada"}]}`))
	})
	r.Put("/api/v1/admin/crm/leads/{id}/move", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		var body map[string]any
		_ = json.NewDecoder(req.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "ok",
			"lead_id":  id,
			"stage_id": body["stage_id"],
			"value":    body["value"],
		})
	})
	r.Post("/api/v1/send-event", func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(req.Body).Decode(&body)
		if body["pixel_name"] == "inexistente" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"pixel not found"}`))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"queued":true,"event_id":"EVT999"}`))
	})
	return r
}

func callMCP(t *testing.T, srv *Server, payload string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "trk_test_valid")
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{
		Email:     "api-key:trk_test",
		Role:      "member",
		TokenType: auth.TokenTypeAPIKey,
		AccountID: 7,
	}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestMCPLifecycleAndToolDispatch(t *testing.T) {
	router := buildStubRouter(t)
	srv := New(router, nil)

	// 1. GET /api/v1/mcp -> 405 Method Not Allowed (Streamable HTTP stateless)
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/mcp", nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /mcp: got %d, want 405", getRec.Code)
	}

	// 2. initialize -> retorna protocolVersion e Mcp-Session-Id
	initRec := callMCP(t, srv, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`)
	if initRec.Code != http.StatusOK {
		t.Fatalf("initialize status: got %d", initRec.Code)
	}
	if initRec.Header().Get("Mcp-Session-Id") == "" {
		t.Fatal("expected Mcp-Session-Id header on initialize")
	}
	if !strings.Contains(initRec.Body.String(), `"trackeado-mcp"`) {
		t.Fatalf("unexpected initialize body: %s", initRec.Body.String())
	}

	// 3. notifications/initialized -> 202 Accepted sem corpo
	notifRec := callMCP(t, srv, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if notifRec.Code != http.StatusAccepted {
		t.Fatalf("notification status: got %d, want 202", notifRec.Code)
	}

	// 4. tools/list -> catálogo completo com inputSchema válido
	listRec := callMCP(t, srv, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	var listResp struct {
		Result struct {
			Tools []ToolDef `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal tools/list: %v", err)
	}
	if len(listResp.Result.Tools) < 30 {
		t.Fatalf("expected >= 30 tools, got %d", len(listResp.Result.Tools))
	}
	for _, tool := range listResp.Result.Tools {
		if tool.InputSchema["type"] != "object" {
			t.Fatalf("tool %s missing object inputSchema", tool.Name)
		}
	}

	// 5. tools/call -> whoami_workspace + move_crm_lead + send_conversion_event
	whoRec := callMCP(t, srv, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"whoami_workspace","arguments":{}}}`)
	if !strings.Contains(whoRec.Body.String(), `clinica-prime`) || !strings.Contains(whoRec.Body.String(), `\"account_id\":7`) {
		t.Fatalf("unexpected whoami_workspace output: %s", whoRec.Body.String())
	}

	moveRec := callMCP(t, srv, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"move_crm_lead","arguments":{"lead_id":42,"stage_id":2,"value":997}}}`)
	if !strings.Contains(moveRec.Body.String(), `"isError":false`) || !strings.Contains(moveRec.Body.String(), `\"lead_id\":\"42\"`) {
		t.Fatalf("unexpected move_crm_lead output: %s", moveRec.Body.String())
	}

	// 6. Erro HTTP upstream propaga isError: true para auto-correção do agente IA
	errCallRec := callMCP(t, srv, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"send_conversion_event","arguments":{"pixel_name":"inexistente","event_name":"Purchase","external_id":"u1"}}}`)
	if !strings.Contains(errCallRec.Body.String(), `"isError":true`) {
		t.Fatalf("expected isError:true on 404 upstream, got: %s", errCallRec.Body.String())
	}

	// 7. Chave restrita a CRM em modo read_only vê apenas tools de leitura de CRM (+ whoami) e bloqueia escrita
	roReq := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":6,"method":"tools/list"}`))
	roReq.Header.Set("Content-Type", "application/json")
	roReq.Header.Set("X-API-Key", "trk_test_valid")
	roReq = roReq.WithContext(auth.WithClaims(roReq.Context(), &auth.Claims{
		Role:           "member",
		TokenType:      auth.TokenTypeAPIKey,
		AccountID:      7,
		APIKeyScopes:   []string{"crm"},
		APIKeyReadOnly: true,
	}))
	roRec := httptest.NewRecorder()
	srv.ServeHTTP(roRec, roReq)
	if !strings.Contains(roRec.Body.String(), `"list_crm_stages"`) || strings.Contains(roRec.Body.String(), `"move_crm_lead"`) || strings.Contains(roRec.Body.String(), `"list_whatsapp_chats"`) {
		t.Fatalf("expected filtered read-only CRM tools in tools/list, got: %s", roRec.Body.String())
	}

	roWriteReq := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"move_crm_lead","arguments":{"lead_id":42,"stage_id":2}}}`))
	roWriteReq.Header.Set("Content-Type", "application/json")
	roWriteReq.Header.Set("X-API-Key", "trk_test_valid")
	roWriteReq = roWriteReq.WithContext(auth.WithClaims(roWriteReq.Context(), &auth.Claims{
		Role:           "member",
		TokenType:      auth.TokenTypeAPIKey,
		AccountID:      7,
		APIKeyScopes:   []string{"crm"},
		APIKeyReadOnly: true,
	}))
	roWriteRec := httptest.NewRecorder()
	srv.ServeHTTP(roWriteRec, roWriteReq)
	if !strings.Contains(roWriteRec.Body.String(), `"isError":true`) {
		t.Fatalf("expected isError:true when calling write tool with read_only key, got: %s", roWriteRec.Body.String())
	}
}

// TestMCPMountedUnderAPIV1Subrouter espelha a árvore real do httpserver (`root.Route("/api/v1", ...)`)
// para garantir que o RouteContext do chi da requisição `POST /api/v1/mcp` não vaze para os
// sub-requests internos de `callAPI` (regressão que causava 404 em todas as tools exceto whoami).
func TestMCPMountedUnderAPIV1Subrouter(t *testing.T) {
	root := chi.NewRouter()
	srv := New(root, nil)

	injectKey := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != "trk_test_valid" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := auth.WithClaims(r.Context(), &auth.Claims{
				Email:        "api-key:trk_test",
				Role:         "member",
				TokenType:    auth.TokenTypeAPIKey,
				AccountID:    3,
				APIKeyScopes: []string{"*"},
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	root.Route("/api/v1", func(r chi.Router) {
		r.Post("/send-event", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"queued":true,"event_id":"EVT999"}`))
		})
		r.Group(func(r chi.Router) {
			r.Use(injectKey)
			r.Post("/mcp", srv.ServeHTTP)
			r.Get("/admin/pixels", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"data":[{"id":10,"name":"clinica-prime"}]}`))
			})
			r.Get("/admin/ext-api/crm/stages", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"stages":[{"id":2,"name":"Venda Realizada"}]}`))
			})
			r.Put("/admin/crm/leads/{id}/move", func(w http.ResponseWriter, req *http.Request) {
				_, _ = w.Write([]byte(`{"status":"ok","lead_id":"` + chi.URLParam(req, "id") + `"}`))
			})
		})
	})

	postMCP := func(body string) string {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "trk_test_valid")
		rec := httptest.NewRecorder()
		root.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST /api/v1/mcp status=%d body=%s", rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	if out := postMCP(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"whoami_workspace","arguments":{}}}`); !strings.Contains(out, `clinica-prime`) {
		t.Fatalf("whoami_workspace should include pixels via internal dispatch, got: %s", out)
	}
	if out := postMCP(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_pixels","arguments":{}}}`); !strings.Contains(out, `"isError":false`) || !strings.Contains(out, `clinica-prime`) {
		t.Fatalf("list_pixels failed under /api/v1 subrouter: %s", out)
	}
	if out := postMCP(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"move_crm_lead","arguments":{"lead_id":42,"stage_id":2,"value":199}}}`); !strings.Contains(out, `"isError":false`) || !strings.Contains(out, `\"lead_id\":\"42\"`) {
		t.Fatalf("move_crm_lead failed under /api/v1 subrouter: %s", out)
	}
	batchOut := postMCP(`[{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_crm_stages","arguments":{}}},{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"send_conversion_event","arguments":{"pixel_name":"p1","event_name":"Purchase","external_id":"u1"}}}]`)
	if !strings.Contains(batchOut, `Venda Realizada`) || !strings.Contains(batchOut, `EVT999`) || strings.Contains(batchOut, `"isError":true`) {
		t.Fatalf("batch tool dispatch failed under /api/v1 subrouter: %s", batchOut)
	}
}
