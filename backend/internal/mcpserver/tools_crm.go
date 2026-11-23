package mcpserver

import (
	"fmt"
	"net/http"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func buildCRMTools() []ToolDef {
	return []ToolDef{
		{
			Name:        "list_crm_stages",
			Scope:       domain.APIScopeCRM,
			Description: "Lista todas as etapas (colunas) do funil de vendas no CRM e seus gatilhos de conversão.",
			InputSchema: objSchema(nil),
			Handler: func(s *Server, origReq *http.Request, _ map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/ext-api/crm/stages", nil, nil)
			},
		},
		{
			Name:        "get_crm_board",
			Scope:       domain.APIScopeCRM,
			Description: "Retorna o quadro completo do CRM (todas as etapas e leads ativos, opcionalmente filtrados por pixel).",
			InputSchema: objSchema(map[string]any{
				"pixel": propStr("Slug do pixel para filtrar os leads (opcional)"),
			}),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/crm/board", buildQuery(args, "pixel"), nil)
			},
		},
		{
			Name:        "get_crm_lead_details",
			Scope:       domain.APIScopeCRM,
			Description: "Retorna os detalhes 360º de um lead do CRM (dados de contato, UTMs, notas e tags).",
			InputSchema: objSchema(map[string]any{
				"lead_id": propInt("ID numérico do lead no CRM"),
			}, "lead_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "lead_id")
				if !ok || id <= 0 {
					return `{"error":"lead_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodGet, fmt.Sprintf("/api/v1/admin/crm/leads/%d", id), nil, nil)
			},
		},
		{
			Name:        "upsert_crm_lead",
			Scope:       domain.APIScopeCRM,
			IsWrite:     true,
			Description: "Cria ou atualiza um lead no CRM posicionando-o na etapa desejada (dispara conversão automática se a etapa tiver gatilho).",
			InputSchema: objSchema(map[string]any{
				"stage_id":    propInt("ID da etapa de destino no funil (obrigatório)"),
				"first_name":  propStr("Primeiro nome do contato"),
				"last_name":   propStr("Sobrenome do contato"),
				"email":       propStr("E-mail do contato"),
				"phone":       propStr("Telefone/WhatsApp com DDI+DDD (ex: 5511999999999)"),
				"value":       propNum("Valor do negócio (obrigatório se a etapa disparar Purchase)"),
				"external_id": propStr("Identificador externo opcional do visitante"),
				"pixel_name":  propStr("Slug do pixel associado (opcional)"),
			}, "stage_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodPost, "/api/v1/admin/ext-api/crm/leads", nil, pickBody(args, "stage_id", "first_name", "last_name", "email", "phone", "value", "external_id", "pixel_name"))
			},
		},
		{
			Name:        "move_crm_lead",
			Scope:       domain.APIScopeCRM,
			IsWrite:     true,
			Description: "Move um lead existente para outra coluna do CRM (disparando os eventos configurados na coluna para Meta Ads, Google Ads, TikTok e GA4).",
			InputSchema: objSchema(map[string]any{
				"lead_id":  propInt("ID numérico do lead no CRM"),
				"stage_id": propInt("ID numérico da nova etapa de destino"),
				"value":    propNum("Valor monetário do negócio (obrigatório em etapas de venda)"),
			}, "lead_id", "stage_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "lead_id")
				if !ok || id <= 0 {
					return `{"error":"lead_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodPut, fmt.Sprintf("/api/v1/admin/crm/leads/%d/move", id), nil, pickBody(args, "stage_id", "value"))
			},
		},
		{
			Name:        "update_crm_lead",
			Scope:       domain.APIScopeCRM,
			IsWrite:     true,
			Description: "Atualiza dados cadastrais (nome, e-mail, telefone) e valor de um lead existente no CRM.",
			InputSchema: objSchema(map[string]any{
				"lead_id":    propInt("ID numérico do lead no CRM"),
				"first_name": propStr("Nome"),
				"last_name":  propStr("Sobrenome"),
				"email":      propStr("E-mail"),
				"phone":      propStr("Telefone"),
				"value":      propNum("Valor monetário"),
			}, "lead_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "lead_id")
				if !ok || id <= 0 {
					return `{"error":"lead_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodPut, fmt.Sprintf("/api/v1/admin/crm/leads/%d", id), nil, pickBody(args, "first_name", "last_name", "email", "phone", "value"))
			},
		},
		{
			Name:        "add_crm_lead_note",
			Scope:       domain.APIScopeCRM,
			IsWrite:     true,
			Description: "Adiciona uma anotação interna no histórico de atendimento de um lead no CRM.",
			InputSchema: objSchema(map[string]any{
				"lead_id": propInt("ID numérico do lead no CRM"),
				"content": propStr("Texto da nota comercial"),
			}, "lead_id", "content"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				id, ok := argInt64(args, "lead_id")
				if !ok || id <= 0 {
					return `{"error":"lead_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodPost, fmt.Sprintf("/api/v1/admin/crm/leads/%d/notes", id), nil, pickBody(args, "content"))
			},
		},
		{
			Name:        "list_crm_tags",
			Scope:       domain.APIScopeCRM,
			Description: "Lista todas as etiquetas (tags) cadastradas no CRM da empresa.",
			InputSchema: objSchema(nil),
			Handler: func(s *Server, origReq *http.Request, _ map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodGet, "/api/v1/admin/crm/tags", nil, nil)
			},
		},
		{
			Name:        "create_crm_tag",
			Scope:       domain.APIScopeCRM,
			IsWrite:     true,
			Description: "Cria uma nova etiqueta (tag) no CRM.",
			InputSchema: objSchema(map[string]any{
				"name":  propStr("Nome da tag"),
				"color": propStr("Cor hexadecimal opcional (ex: #22C55E)"),
			}, "name"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				return s.callAPI(origReq, http.MethodPost, "/api/v1/admin/crm/tags", nil, pickBody(args, "name", "color"))
			},
		},
		{
			Name:        "assign_crm_lead_tag",
			Scope:       domain.APIScopeCRM,
			IsWrite:     true,
			Description: "Vincula uma tag existente a um lead do CRM.",
			InputSchema: objSchema(map[string]any{
				"lead_id": propInt("ID numérico do lead"),
				"tag_id":  propInt("ID numérico da tag"),
			}, "lead_id", "tag_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				leadID, ok1 := argInt64(args, "lead_id")
				tagID, ok2 := argInt64(args, "tag_id")
				if !ok1 || !ok2 || leadID <= 0 || tagID <= 0 {
					return `{"error":"lead_id ou tag_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodPost, fmt.Sprintf("/api/v1/admin/crm/leads/%d/tags/%d", leadID, tagID), nil, map[string]any{})
			},
		},
		{
			Name:        "remove_crm_lead_tag",
			Scope:       domain.APIScopeCRM,
			IsWrite:     true,
			Description: "Remove uma tag de um lead do CRM.",
			InputSchema: objSchema(map[string]any{
				"lead_id": propInt("ID numérico do lead"),
				"tag_id":  propInt("ID numérico da tag"),
			}, "lead_id", "tag_id"),
			Handler: func(s *Server, origReq *http.Request, args map[string]any) (string, bool) {
				leadID, ok1 := argInt64(args, "lead_id")
				tagID, ok2 := argInt64(args, "tag_id")
				if !ok1 || !ok2 || leadID <= 0 || tagID <= 0 {
					return `{"error":"lead_id ou tag_id inválido"}`, true
				}
				return s.callAPI(origReq, http.MethodDelete, fmt.Sprintf("/api/v1/admin/crm/leads/%d/tags/%d", leadID, tagID), nil, nil)
			},
		},
	}
}
