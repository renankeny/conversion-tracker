package mcpserver

import (
	"fmt"
	"net/http"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func buildPixelsAndAdsTools() []ToolDef {
	return []ToolDef{
		{
			Name:        "list_pixels",
			Scope:       domain.APIScopePixels,
			Description: "Lista todos os pixels configurados na empresa com domínios e integrações vinculadas.",
			InputSchema: objSchema(nil),
			Handler: func(s *Server, origReq *http.Request, _ map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/pixels", nil, nil)
			},
		},
		{
			Name:        "list_trackable_links",
			Scope:       domain.APIScopePixels,
			Description: "Lista os links rastreáveis (Link na Bio / redirecionamento WhatsApp) de um pixel com métricas de cliques e vendas.",
			InputSchema: objSchema(map[string]any{
				"pixel_id": propInt("ID numérico do pixel"),
			}, "pixel_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "pixel_id")
				if !ok || id <= 0 {
					return `{"error":"pixel_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodGet, fmt.Sprintf("/api/v1/admin/pixels/%d/trackable-links", id), nil, nil)
			},
		},
		{
			Name:        "create_trackable_link",
			Scope:       domain.APIScopePixels,
			IsWrite:     true,
			Description: "Cria um novo link rastreável para WhatsApp (com rodízio de instâncias) ou página externa.",
			InputSchema: objSchema(map[string]any{
				"pixel_id":              propInt("ID numérico do pixel"),
				"name":                  propStr("Nome interno do link (ex: Link na Bio Instagram)"),
				"destination_type":      propStr("Tipo de destino: whatsapp ou url"),
				"destination_url":       propStr("URL final quando destination_type = url"),
				"prefilled_message":     propStr("Mensagem inicial pré-preenchida no WhatsApp"),
				"whatsapp_instance_ids": propArrInt("IDs das instâncias de WhatsApp para atendimento/rodízio"),
			}, "pixel_id", "name", "destination_type"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "pixel_id")
				if !ok || id <= 0 {
					return `{"error":"pixel_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodPost, fmt.Sprintf("/api/v1/admin/pixels/%d/trackable-links", id), nil, pickBody(
					args, "name", "destination_type", "destination_url", "prefilled_message", "whatsapp_instance_ids",
				))
			},
		},
		{
			Name:        "get_trackable_links_stats",
			Scope:       domain.APIScopePixels,
			Description: "Retorna estatísticas consolidadas dos links rastreáveis no período.",
			InputSchema: objSchema(map[string]any{
				"pixel_name": propStr("Slug do pixel (opcional)"),
				"from":       propStr("Data inicial RFC3339"),
				"to":         propStr("Data final RFC3339"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/trackable-links/stats", buildQuery(args, "pixel_name", "from", "to"), nil)
			},
		},
		{
			Name:        "get_dashboard_stats",
			Scope:       domain.APIScopeAnalytics,
			Description: "Retorna visão geral do dashboard (receita rastreada, vendas, ticket médio, funil e canais).",
			InputSchema: objSchema(map[string]any{
				"pixel":          propStr("Slug do pixel (opcional)"),
				"traffic_source": propStr("Origem de tráfego opcional"),
				"from":           propStr("Data inicial RFC3339"),
				"to":             propStr("Data final RFC3339"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/stats", buildQuery(args, "pixel", "traffic_source", "from", "to"), nil)
			},
		},
		{
			Name:        "get_event_counters",
			Scope:       domain.APIScopeAnalytics,
			Description: "Retorna contadores consolidados por tipo de evento no período.",
			InputSchema: objSchema(map[string]any{
				"pixel": propStr("Slug do pixel (opcional)"),
				"event": propStr("Nome do evento opcional"),
				"from":  propStr("Data inicial RFC3339"),
				"to":    propStr("Data final RFC3339"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/counters/summary", buildQuery(args, "pixel", "event", "from", "to"), nil)
			},
		},
		{
			Name:        "get_geo_stats",
			Scope:       domain.APIScopeAnalytics,
			Description: "Retorna o mapa de distribuição geográfica (estados, cidades e países compradores).",
			InputSchema: objSchema(map[string]any{
				"pixel": propStr("Slug do pixel (opcional)"),
				"from":  propStr("Data inicial RFC3339"),
				"to":    propStr("Data final RFC3339"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/stats/geo", buildQuery(args, "pixel", "from", "to"), nil)
			},
		},
		{
			Name:        "get_meta_ads_insights",
			Scope:       domain.APIScopeAnalytics,
			Description: "Consulta investimento, cliques, CPC, CPM e métricas ao vivo do Meta Ads para um pixel no período.",
			InputSchema: objSchema(map[string]any{
				"pixel_name": propStr("Slug do pixel com conta Meta Ads vinculada"),
				"from":       propStr("Data inicial YYYY-MM-DD ou RFC3339"),
				"to":         propStr("Data final YYYY-MM-DD ou RFC3339"),
			}, "pixel_name", "from", "to"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/dashboard/meta/live-insights", buildQuery(args, "pixel_name", "from", "to"), nil)
			},
		},
		{
			Name:        "get_meta_creative_ranking",
			Scope:       domain.APIScopeAnalytics,
			Description: "Retorna o ranking dos criativos campeões do Meta Ads cruzados com vendas e ROAS real rastreados.",
			InputSchema: objSchema(map[string]any{
				"pixel_name": propStr("Slug do pixel com conta Meta Ads vinculada"),
				"from":       propStr("Data inicial YYYY-MM-DD ou RFC3339"),
				"to":         propStr("Data final YYYY-MM-DD ou RFC3339"),
			}, "pixel_name", "from", "to"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/dashboard/meta/creative-ranking", buildQuery(args, "pixel_name", "from", "to"), nil)
			},
		},
		{
			Name:        "get_google_ads_insights",
			Scope:       domain.APIScopeAnalytics,
			Description: "Consulta investimento, cliques e métricas ao vivo do Google Ads para um pixel no período.",
			InputSchema: objSchema(map[string]any{
				"pixel_name": propStr("Slug do pixel com conta Google Ads vinculada"),
				"from":       propStr("Data inicial YYYY-MM-DD ou RFC3339"),
				"to":         propStr("Data final YYYY-MM-DD ou RFC3339"),
			}, "pixel_name", "from", "to"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/dashboard/google/live-insights", buildQuery(args, "pixel_name", "from", "to"), nil)
			},
		},
		{
			Name:        "get_google_keyword_ranking",
			Scope:       domain.APIScopeAnalytics,
			Description: "Retorna o ranking de palavras-chave do Google Ads cruzadas com conversões e ROAS real rastreados.",
			InputSchema: objSchema(map[string]any{
				"pixel_name": propStr("Slug do pixel com conta Google Ads vinculada"),
				"from":       propStr("Data inicial YYYY-MM-DD ou RFC3339"),
				"to":         propStr("Data final YYYY-MM-DD ou RFC3339"),
			}, "pixel_name", "from", "to"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/dashboard/google/keyword-ranking", buildQuery(args, "pixel_name", "from", "to"), nil)
			},
		},
	}
}
