package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
)

// ToolHandler executa uma tool MCP recebendo os argumentos e devolvendo (payloadTexto, isError).
type ToolHandler func(s *Server, origReq *http.Request, args map[string]any) (string, bool)

// ToolDef descreve uma ferramenta exposta via MCP `tools/list`.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Scope       string         `json:"-"`
	IsWrite     bool           `json:"-"`
	Handler     ToolHandler    `json:"-"`
}

func (s *Server) isToolAllowed(r *http.Request, t ToolDef) bool {
	c, ok := auth.ClaimsFrom(r.Context())
	if !ok || c == nil {
		return true
	}
	return c.HasAPIScope(t.Scope, t.IsWrite)
}

func (s *Server) visibleToolsFor(r *http.Request) []ToolDef {
	out := make([]ToolDef, 0, len(s.tools))
	for _, t := range s.tools {
		if s.isToolAllowed(r, t) {
			out = append(out, t)
		}
	}
	return out
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) callAPI(origReq *http.Request, method, path string, query url.Values, body any) (string, bool) {
	fullPath := path
	if len(query) > 0 {
		fullPath = path + "?" + query.Encode()
	}
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Sprintf(`{"error":"failed to encode request: %v"}`, err), true
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(origReq.Context(), method, fullPath, bodyReader)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to create request: %v"}`, err), true
	}
	req.Host = origReq.Host
	req.RemoteAddr = origReq.RemoteAddr
	req.Header.Set("Content-Type", "application/json")
	for _, hdr := range []string{"X-API-Key", "Authorization", "X-Account-ID"} {
		if v := origReq.Header.Get(hdr); v != "" {
			req.Header.Set(hdr, v)
		}
	}

	rec := httptest.NewRecorder()
	s.dispatcher.ServeHTTP(rec, req)
	out := strings.TrimSpace(rec.Body.String())
	if out == "" {
		out = fmt.Sprintf(`{"status_code":%d}`, rec.Code)
	}
	return out, rec.Code >= 400
}

func writeRPCJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func objSchema(props map[string]any, required ...string) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	out := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func propStr(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func propInt(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func propNum(desc string) map[string]any {
	return map[string]any{"type": "number", "description": desc}
}

func propArrStr(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}

func argStr(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func argInt64(args map[string]any, key string) (int64, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int64:
		return t, true
	case int:
		return int64(t), true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func buildQuery(args map[string]any, keys ...string) url.Values {
	q := url.Values{}
	for _, k := range keys {
		if val := argStr(args, k); val != "" {
			q.Set(k, val)
		}
	}
	return q
}

func pickBody(args map[string]any, keys ...string) map[string]any {
	out := make(map[string]any, len(keys))
	for _, k := range keys {
		if v, ok := args[k]; ok && v != nil {
			out[k] = v
		}
	}
	return out
}
