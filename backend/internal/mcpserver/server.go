package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultProtocolVersion = "2025-03-26"
	legacyProtocolVersion  = "2024-11-05"
	serverName             = "trackeado-mcp"
	serverVersion          = "1.0.0"
)

// Server implementa o protocolo MCP Streamable HTTP (JSON-RPC 2.0) embutido na API Go.
type Server struct {
	dispatcher http.Handler
	pool       *pgxpool.Pool
	tools      []ToolDef
	byName     map[string]ToolDef
}

// New constrói o servidor MCP nativo despachando chamadas internamente sobre o router principal.
func New(dispatcher http.Handler, pool *pgxpool.Pool) *Server {
	s := &Server{
		dispatcher: dispatcher,
		pool:       pool,
		byName:     make(map[string]ToolDef, 48),
	}
	all := make([]ToolDef, 0, 40)
	all = append(all, buildWhatsAppTools()...)
	all = append(all, buildCRMTools()...)
	all = append(all, buildEventsAndUsersTools()...)
	all = append(all, buildPixelsAndAdsTools()...)
	s.tools = all
	for _, t := range all {
		s.byName[t.Name] = t
	}
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		w.WriteHeader(http.StatusOK)
		return
	case http.MethodGet:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	case http.MethodPost:
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	rawBody, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024))
	if err != nil || len(bytes.TrimSpace(rawBody)) == 0 {
		writeRPCJSON(w, http.StatusBadRequest, rpcResponse{
			JSONRPC: "2.0",
			ID:      json.RawMessage("null"),
			Error:   &rpcError{Code: -32700, Message: "Parse error: empty or unreadable body"},
		})
		return
	}

	trimmed := bytes.TrimSpace(rawBody)
	if trimmed[0] == '[' {
		var batch []rpcRequest
		if err := json.Unmarshal(trimmed, &batch); err != nil || len(batch) == 0 {
			writeRPCJSON(w, http.StatusBadRequest, rpcResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &rpcError{Code: -32700, Message: "Invalid JSON-RPC batch"},
			})
			return
		}
		responses := make([]rpcResponse, 0, len(batch))
		for _, req := range batch {
			if resp, hasResp := s.handleSingle(w, r, req); hasResp {
				responses = append(responses, resp)
			}
		}
		if len(responses) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeRPCJSON(w, http.StatusOK, responses)
		return
	}

	var req rpcRequest
	if err := json.Unmarshal(trimmed, &req); err != nil {
		writeRPCJSON(w, http.StatusBadRequest, rpcResponse{
			JSONRPC: "2.0",
			ID:      json.RawMessage("null"),
			Error:   &rpcError{Code: -32700, Message: "Parse error: invalid JSON"},
		})
		return
	}

	resp, hasResp := s.handleSingle(w, r, req)
	if !hasResp {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeRPCJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSingle(w http.ResponseWriter, r *http.Request, req rpcRequest) (rpcResponse, bool) {
	isNotification := len(req.ID) == 0 || string(bytes.TrimSpace(req.ID)) == ""
	if isNotification || strings.HasPrefix(req.Method, "notifications/") {
		return rpcResponse{}, false
	}

	switch req.Method {
	case "initialize":
		var initParams struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &initParams)
		negotiated := defaultProtocolVersion
		if initParams.ProtocolVersion == legacyProtocolVersion {
			negotiated = legacyProtocolVersion
		}
		if w.Header().Get("Mcp-Session-Id") == "" {
			w.Header().Set("Mcp-Session-Id", uuid.NewString())
		}
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": negotiated,
				"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
				"serverInfo":      map[string]any{"name": serverName, "version": serverVersion},
			},
		}, true

	case "ping":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}, true

	case "resources/list":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"resources": []any{}}}, true

	case "prompts/list":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"prompts": []any{}}}, true

	case "tools/list":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": s.visibleToolsFor(r)}}, true

	case "tools/call":
		var callParams struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil || strings.TrimSpace(callParams.Name) == "" {
			return rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32602, Message: "Invalid params: tool name is required"},
			}, true
		}
		tool, ok := s.byName[strings.TrimSpace(callParams.Name)]
		if !ok {
			return rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("Unknown tool: %s", callParams.Name)},
			}, true
		}
		if !s.isToolAllowed(r, tool) {
			return rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]any{
					"content": []map[string]any{{"type": "text", "text": `{"error":"esta API key não possui permissão para executar esta ferramenta (verifique os módulos liberados ou o modo somente leitura)"}`}},
					"isError": true,
				},
			}, true
		}
		if callParams.Arguments == nil {
			callParams.Arguments = map[string]any{}
		}
		text, isErr := tool.Handler(s, r, callParams.Arguments)
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"content": []map[string]any{{"type": "text", "text": text}},
				"isError": isErr,
			},
		}, true

	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}, true
	}
}
