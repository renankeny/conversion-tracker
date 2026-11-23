package mcpserver

import (
	"net/http"
	"net/url"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func buildEventsAndUsersTools() []ToolDef {
	return []ToolDef{
		{
			Name:        "send_conversion_event",
			Scope:       domain.APIScopeEvents,
			IsWrite:     true,
			Description: "Envia um evento de conversão (Purchase, Lead, InitiateCheckout, AddToCart, etc.) para Meta Ads, Google Ads, TikTok Ads e GA4.",
			InputSchema: objSchema(map[string]any{
				"pixel_name":   propStr("Slug do pixel cadastrado (obrigatório)"),
				"event_name":   propStr("Nome do evento: Purchase, Lead, InitiateCheckout, AddToCart, ViewContent, etc."),
				"external_id":  propStr("ID único do visitante/contato (obrigatório)"),
				"value":        propNum("Valor monetário (ex: 497.00)"),
				"currency":     propStr("Moeda ISO (ex: BRL)"),
				"email":        propStr("E-mail do comprador/lead"),
				"phone":        propStr("Telefone com DDI+DDD"),
				"first_name":   propStr("Primeiro nome"),
				"last_name":    propStr("Sobrenome"),
				"order_id":     propStr("ID único do pedido para deduplicação"),
				"content_name": propStr("Nome do produto ou oferta"),
			}, "pixel_name", "event_name", "external_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodPost, "/api/v1/send-event", nil, pickBody(
					args, "pixel_name", "event_name", "external_id", "value", "currency",
					"email", "phone", "first_name", "last_name", "order_id", "content_name",
				))
			},
		},
		{
			Name:        "list_sent_events",
			Scope:       domain.APIScopeEvents,
			Description: "Lista os eventos enviados às plataformas (Meta, Google, TikTok, GA4) com filtros por pixel, tipo de evento, status e período.",
			InputSchema: objSchema(map[string]any{
				"pixel":  propStr("Filtrar por slug do pixel"),
				"event":  propStr("Filtrar por tipo de evento (ex: Purchase,Lead)"),
				"status": propStr("Filtrar por status (ok ou error)"),
				"search": propStr("Buscar por nome, e-mail, telefone ou event_id"),
				"from":   propStr("Data inicial RFC3339"),
				"to":     propStr("Data final RFC3339"),
				"limit":  propInt("Limite de registros (padrão 25)"),
				"offset": propInt("Deslocamento de paginação"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/sent-events/grouped", buildQuery(args, "pixel", "event", "status", "search", "from", "to", "limit", "offset"), nil)
			},
		},
		{
			Name:        "get_event_by_id",
			Scope:       domain.APIScopeEvents,
			Description: "Inspeciona o payload completo e a resposta de cada canal (Meta, Google, TikTok, GA4) para um event_id específico.",
			InputSchema: objSchema(map[string]any{
				"event_id": propStr("ID do evento"),
				"pixel":    propStr("Slug do pixel (opcional)"),
			}, "event_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				eid := argStr(args, "event_id")
				if eid == "" {
					return `{"error":"event_id obrigatório"}`, true
				}
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/sent-events/by-event-id/"+url.PathEscape(eid), buildQuery(args, "pixel"), nil)
			},
		},
		{
			Name:        "retry_failed_events",
			Scope:       domain.APIScopeEvents,
			IsWrite:     true,
			Description: "Reenfileira até 100 eventos com erro para reprocessamento imediato nas plataformas.",
			InputSchema: objSchema(map[string]any{
				"event_ids": propArrStr("Lista de event_ids a reenviar"),
			}, "event_ids"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodPost, "/api/v1/admin/sent-events/retry", nil, pickBody(args, "event_ids"))
			},
		},
		{
			Name:        "list_pending_sales",
			Scope:       domain.APIScopeSales,
			Description: "Lista pedidos/boletos/PIX gerados aguardando confirmação de pagamento.",
			InputSchema: objSchema(map[string]any{
				"pixel":  propStr("Slug do pixel (opcional)"),
				"limit":  propInt("Limite de itens"),
				"offset": propInt("Offset de paginação"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/sales-confirmations", buildQuery(args, "pixel", "limit", "offset"), nil)
			},
		},
		{
			Name:        "confirm_pending_sale",
			Scope:       domain.APIScopeSales,
			IsWrite:     true,
			Description: "Confirma o pagamento de um pedido pendente, disparando o evento oficial Purchase atribuído à origem original.",
			InputSchema: objSchema(map[string]any{
				"event_id":   propStr("event_id do pedido pendente"),
				"pixel_name": propStr("Slug do pixel (opcional)"),
				"value":      propNum("Valor confirmado da compra"),
				"currency":   propStr("Moeda ISO (padrão BRL)"),
			}, "event_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodPost, "/api/v1/admin/sales-confirmations/confirm", nil, pickBody(args, "event_id", "pixel_name", "value", "currency"))
			},
		},
		{
			Name:        "lookup_tracked_user",
			Scope:       domain.APIScopeUsers,
			Description: "Localiza um visitante/lead rastreado por id, external_id, email ou phone, retornando UTMs, cookies e LTV.",
			InputSchema: objSchema(map[string]any{
				"id":          propInt("ID numérico interno"),
				"external_id": propStr("External ID do visitante"),
				"email":       propStr("E-mail do visitante"),
				"phone":       propStr("Telefone/WhatsApp do visitante"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/tracked-users/lookup", buildQuery(args, "id", "external_id", "email", "phone"), nil)
			},
		},
		{
			Name:        "list_tracked_users",
			Scope:       domain.APIScopeUsers,
			Description: "Lista visitantes e leads rastreados com busca por nome/e-mail/telefone e filtro por pixel ou segmento.",
			InputSchema: objSchema(map[string]any{
				"pixel":   propStr("Slug do pixel"),
				"segment": propStr("Segmento opcional (ex: buyers, leads)"),
				"search":  propStr("Termo de busca"),
				"limit":   propInt("Limite de itens"),
				"offset":  propInt("Offset"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/tracked-users", buildQuery(args, "pixel", "segment", "search", "limit", "offset"), nil)
			},
		},
		{
			Name:        "get_tracked_user_journey",
			Scope:       domain.APIScopeUsers,
			Description: "Retorna a jornada completa multi-sessão de um visitante (primeiro toque, dispositivos, UTMs e todos os eventos disparados).",
			InputSchema: objSchema(map[string]any{
				"external_id": propStr("External ID do visitante"),
			}, "external_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/tracked-users/journey", buildQuery(args, "external_id"), nil)
			},
		},
	}
}
