package whatsappai

import (
	"fmt"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

const DefaultBasePrompt = `Você é um analista comercial sênior especialista em vendas pelo WhatsApp.
Sua função é analisar o histórico da conversa (textos, transcrições de áudios, imagens/prints e documentos PDF) entre o LEAD (cliente) e o ATENDENTE da empresa, identificando com precisão dois marcos:

1. QUALIFICAÇÃO DO LEAD (is_qualified):
- Marque true (confidence = "high") SOMENTE quando o lead interagiu de verdade, demonstrou interesse concreto no produto/serviço (tirou dúvidas reais, perguntou valores/planos/condições, respondeu perguntas de triagem ou pediu atendimento comercial) e tem perfil real de compra.
- NÃO qualifique se o contato mandou apenas 1 saudação automática e sumiu, se for engano, spam, cobrança de fornecedor, currículo ou suporte técnico sem intenção comercial.

2. VENDA REALIZADA (new_sales):
- RIGOR MÁXIMO: SÓ registre uma venda (confidence = "certain") quando houver CERTEZA ABSOLUTA de que a compra foi EFETIVAMENTE FECHADA E PAGA (ex.: comprovante de PIX/transferência/cartão pago em imagem ou PDF, confirmação clara de pagamento compensado, acesso liberado ou pedido confirmado como pago).
- NUNCA confunda ORÇAMENTO, proposta comercial, simulação de valores, envio de chave PIX aguardando pagamento, boleto gerado ou link de checkout ainda não pago com uma venda! Na menor dúvida se foi pago ou se é apenas orçamento, NÃO marque venda.
- Extraia o valor numérico exato pago (amount) do comprovante ou da confirmação na conversa (use 0 apenas se a venda estiver 100% confirmada, porém sem nenhum valor citado em texto, áudio, imagem ou PDF).
- Podem ocorrer múltiplas compras ao longo do tempo: registre apenas vendas NOVAS que ainda não constem na lista de vendas já registradas.`

type QualifiedDecision struct {
	IsQualified      bool   `json:"is_qualified"`
	Confidence       string `json:"confidence"`
	Reason           string `json:"reason"`
	TriggerMessageID int64  `json:"trigger_message_id"`
}

type SaleDecision struct {
	EvidenceMessageID int64   `json:"evidence_message_id"`
	Confidence        string  `json:"confidence"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	Reason            string  `json:"reason"`
}

type MediaSummary struct {
	MessageID int64  `json:"message_id"`
	Summary   string `json:"summary"`
}

type AnalysisResult struct {
	Qualified      QualifiedDecision `json:"qualified"`
	NewSales       []SaleDecision    `json:"new_sales"`
	MediaSummaries []MediaSummary    `json:"media_summaries"`
}

func EffectiveBasePrompt(custom string) string {
	if strings.TrimSpace(custom) != "" {
		return strings.TrimSpace(custom)
	}
	return DefaultBasePrompt
}

func BuildSystemPrompt(
	basePrompt string,
	rules []domain.AILearnedRule,
	alreadyQualified bool,
	priorSales []repo.PriorPurchaseRef,
	lastAnalyzedMsgID int64,
) string {
	var sb strings.Builder
	sb.WriteString(EffectiveBasePrompt(basePrompt))
	sb.WriteString("\n\n--- DIRETRIZES TÉCNICAS OBRIGATÓRIAS ---\n")
	if alreadyQualified {
		sb.WriteString("- ESTE LEAD JÁ FOI QUALIFICADO ANTERIORMENTE. Retorne obrigatoriamente qualified.is_qualified = false e avalie APENAS se surgiram NOVAS vendas confirmadas.\n")
	} else {
		sb.WriteString("- Este lead ainda não foi qualificado. Avalie tanto a qualificação (qualified) quanto eventuais vendas confirmadas (new_sales).\n")
	}
	if lastAnalyzedMsgID > 0 {
		fmt.Fprintf(&sb, "- Mensagens com ID <= %d servem apenas de contexto histórico; vendas novas devem ter evidência conclusiva nas mensagens novas (ID > %d).\n", lastAnalyzedMsgID, lastAnalyzedMsgID)
	}
	sb.WriteString("- Para cada imagem ou PDF inédito anexado nesta análise, inclua em media_summaries um resumo objetivo (ex.: 'Comprovante PIX pago R$ 297,00' ou 'Foto de dúvida sobre produto') com o respectivo message_id.\n")

	if len(priorSales) > 0 {
		sb.WriteString("\n--- VENDAS JÁ REGISTRADAS NESTA CONVERSA (NÃO REPETIR) ---\n")
		for _, p := range priorSales {
			fmt.Fprintf(&sb, "- Fonte: %s | MsgID: %d | Valor: %.2f | Data: %s\n",
				p.Source, p.MessageID, p.Amount, p.CreatedAt.Format("02/01/2006 15:04"))
		}
	}

	if len(rules) > 0 {
		sb.WriteString("\n--- REGRAS APRENDIDAS COM CONTESTAÇÕES DO ADMIN (PRIORIDADE MÁXIMA) ---\n")
		sb.WriteString("Respeite rigorosamente estas correções feitas pelo administrador da conta:\n")
		for i, r := range rules {
			if strings.TrimSpace(r.Rule) == "" {
				continue
			}
			label := "Geral"
			switch r.EventType {
			case domain.AIEventQualified:
				label = "Qualificação"
			case domain.AIEventPurchase:
				label = "Venda"
			}
			fmt.Fprintf(&sb, "%d. [%s] %s\n", i+1, label, strings.TrimSpace(r.Rule))
		}
	}
	return sb.String()
}

func responseJSONSchema() map[string]any {
	return map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   "whatsapp_commercial_analysis",
			"strict": true,
			"schema": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"qualified", "new_sales", "media_summaries"},
				"properties": map[string]any{
					"qualified": map[string]any{
						"type":                 "object",
						"additionalProperties": false,
						"required":             []string{"is_qualified", "confidence", "reason", "trigger_message_id"},
						"properties": map[string]any{
							"is_qualified":       map[string]any{"type": "boolean"},
							"confidence":         map[string]any{"type": "string", "enum": []string{"none", "low", "medium", "high"}},
							"reason":             map[string]any{"type": "string"},
							"trigger_message_id": map[string]any{"type": "integer"},
						},
					},
					"new_sales": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"required":             []string{"evidence_message_id", "confidence", "amount", "currency", "reason"},
							"properties": map[string]any{
								"evidence_message_id": map[string]any{"type": "integer"},
								"confidence":          map[string]any{"type": "string", "enum": []string{"uncertain", "quote_only", "certain"}},
								"amount":              map[string]any{"type": "number"},
								"currency":            map[string]any{"type": "string"},
								"reason":              map[string]any{"type": "string"},
							},
						},
					},
					"media_summaries": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type":                 "object",
							"additionalProperties": false,
							"required":             []string{"message_id", "summary"},
							"properties": map[string]any{
								"message_id": map[string]any{"type": "integer"},
								"summary":    map[string]any{"type": "string"},
							},
						},
					},
				},
			},
		},
	}
}

