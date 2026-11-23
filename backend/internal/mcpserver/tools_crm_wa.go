package mcpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func buildWhatsAppTools() []ToolDef {
	return []ToolDef{
		{
			Name:        "whoami_workspace",
			Description: "Retorna o contexto da empresa autenticada pela API Key (account_id, nome da empresa e permissões ativas da chave).",
			InputSchema: objSchema(nil),
			Handler: func(s *Server, origReq *http.Request, _ map[string]any) (string, bool) {
				c, _ := auth.ClaimsFrom(origReq.Context())
				var accID int64
				var actor string
				var scopes []string
				var readOnly bool
				if c != nil {
					accID = c.AccountID
					actor = c.Email
					scopes = c.APIKeyScopes
					readOnly = c.APIKeyReadOnly
				}
				var accountName string
				if s.pool != nil && accID > 0 {
					_ = s.pool.QueryRow(origReq.Context(), `SELECT name FROM accounts WHERE id = $1`, accID).Scan(&accountName)
				}
				var pixels any
				if c == nil || c.HasAPIScope(domain.APIScopePixels, false) {
					pixelsRaw, _ := s.callAPI(origReq, http.MethodGet, "/api/v1/admin/pixels", nil, nil)
					var pixelsObj map[string]any
					_ = json.Unmarshal([]byte(pixelsRaw), &pixelsObj)
					pixels = pixelsObj["data"]
				}
				b, _ := json.Marshal(map[string]any{
					"account_id":    accID,
					"account_name":  accountName,
					"authenticated": actor,
					"scopes":        scopes,
					"read_only":     readOnly,
					"pixels":        pixels,
				})
				return string(b), false
			},
		},
		{
			Name:        "list_whatsapp_instances",
			Scope:       domain.APIScopeWhatsApp,
			Description: "Lista os números/instâncias de WhatsApp conectados na empresa e o status de conexão de cada um.",
			InputSchema: objSchema(nil),
			Handler: func(s *Server, origReq *http.Request, _ map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/whatsapp/instances", nil, nil)
			},
		},
		{
			Name:        "list_whatsapp_chats",
			Scope:       domain.APIScopeWhatsApp,
			Description: "Lista as conversas comerciais do WhatsApp da empresa com dados do contato, anúncio de origem CTWA e última mensagem.",
			InputSchema: objSchema(nil),
			Handler: func(s *Server, origReq *http.Request, _ map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/whatsapp/chats", nil, nil)
			},
		},
		{
			Name:        "get_whatsapp_chat_messages",
			Scope:       domain.APIScopeWhatsApp,
			Description: "Carrega o histórico de mensagens de uma conversa do WhatsApp (incluindo transcrições de áudio/imagem e eventos da IA).",
			InputSchema: objSchema(map[string]any{
				"chat_id":   propInt("ID numérico da conversa do WhatsApp"),
				"limit":     propInt("Quantidade de mensagens (1 a 100, padrão 20)"),
				"before_id": propInt("Cursor opcional para paginação reversa"),
			}, "chat_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "chat_id")
				if !ok || id <= 0 {
					return `{"error":"chat_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodGet, fmt.Sprintf("/api/v1/admin/whatsapp/chats/%d/messages", id), buildQuery(args, "limit", "before_id"), nil)
			},
		},
		{
			Name:        "get_whatsapp_chat_by_lead",
			Scope:       domain.APIScopeWhatsApp,
			Description: "Busca a conversa e mensagens do WhatsApp vinculadas a um card/lead específico do CRM.",
			InputSchema: objSchema(map[string]any{
				"lead_id": propInt("ID numérico do lead no CRM"),
				"limit":   propInt("Quantidade de mensagens (1 a 100, padrão 20)"),
			}, "lead_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "lead_id")
				if !ok || id <= 0 {
					return `{"error":"lead_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodGet, fmt.Sprintf("/api/v1/admin/whatsapp/chats/by-lead/%d", id), buildQuery(args, "limit"), nil)
			},
		},
		{
			Name:        "get_whatsapp_chat_by_user",
			Scope:       domain.APIScopeWhatsApp,
			Description: "Busca a conversa e mensagens do WhatsApp vinculadas a um usuário rastreado (tracked_user_id).",
			InputSchema: objSchema(map[string]any{
				"user_id": propInt("ID numérico do usuário rastreado"),
				"limit":   propInt("Quantidade de mensagens (1 a 100, padrão 20)"),
			}, "user_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "user_id")
				if !ok || id <= 0 {
					return `{"error":"user_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodGet, fmt.Sprintf("/api/v1/admin/whatsapp/chats/by-user/%d", id), buildQuery(args, "limit"), nil)
			},
		},
		{
			Name:        "convert_whatsapp_chat",
			Scope:       domain.APIScopeWhatsApp,
			IsWrite:     true,
			Description: "Registra uma conversão oficial (ex: Purchase, Lead, QualifiedLead, Schedule) a partir de uma conversa do WhatsApp.",
			InputSchema: objSchema(map[string]any{
				"chat_id":    propInt("ID numérico da conversa do WhatsApp"),
				"event_name": propStr("Evento a disparar: Purchase, Lead, QualifiedLead, InitiateCheckout, Schedule, Contact"),
				"value":      propNum("Valor monetário da venda (obrigatório quando event_name = Purchase)"),
				"currency":   propStr("Moeda ISO (padrão BRL)"),
			}, "chat_id", "event_name"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "chat_id")
				if !ok || id <= 0 {
					return `{"error":"chat_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodPost, fmt.Sprintf("/api/v1/admin/whatsapp/chats/%d/convert", id), nil, pickBody(args, "event_name", "value", "currency"))
			},
		},
		{
			Name:        "run_whatsapp_ai_now",
			Scope:       domain.APIScopeWhatsApp,
			IsWrite:     true,
			Description: "Dispara imediatamente uma varredura da IA comercial nas conversas recentes do WhatsApp.",
			InputSchema: objSchema(nil),
			Handler: func(s *Server, origReq *http.Request, _ map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodPost, "/api/v1/admin/whatsapp/ai-settings/run-now", nil, map[string]any{})
			},
		},
		{
			Name:        "get_whatsapp_stats",
			Scope:       domain.APIScopeWhatsApp,
			Description: "Retorna métricas consolidadas de conversas, leads CTWA e vendas do WhatsApp no período.",
			InputSchema: objSchema(map[string]any{
				"pixel_name": propStr("Filtrar por slug do pixel (opcional)"),
				"from":       propStr("Data/hora inicial ISO RFC3339"),
				"to":         propStr("Data/hora final ISO RFC3339"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/whatsapp-stats/overview", buildQuery(args, "pixel_name", "from", "to"), nil)
			},
		},
	}
}
