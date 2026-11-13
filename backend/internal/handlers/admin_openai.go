package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

type AdminOpenAI struct {
	users *repo.AdminUserRepo
}

func NewAdminOpenAI(pool *pgxpool.Pool) *AdminOpenAI {
	return &AdminOpenAI{users: repo.NewAdminUserRepo(pool)}
}

type GenerateMappingInput struct {
	Payload string `json:"payload"`
}

func (h *AdminOpenAI) GenerateMapping(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var in GenerateMappingInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	apiKey := loadDecryptedOpenAIKey(r.Context(), h.users, claims.UserID)
	if apiKey == "" {
		writeError(w, http.StatusBadRequest, "Chave da OpenAI não configurada. Vá em Configurações > Meu Perfil para configurar.")
		return
	}

	// Chamar OpenAI
	prompt := `Você é um assistente especialista em integrações de Webhooks. 
Seu objetivo é ler um Payload JSON de exemplo de um webhook de pagamento e retornar os caminhos JSONPath válidos para os dados solicitados.

CONTEXTO IMPORTANTE DO SISTEMA (TRACKFIND):
O nosso sistema rastreia os usuários usando um identificador único chamado "userId" (ou "external_id", gerado no script do frontend do cliente).
Nos fluxos de checkout, esse "userId" geralmente é injetado dinamicamente nos parâmetros de rastreio das plataformas de venda (como parâmetros de URL). 
Por isso, extrair o campo equivalente ao "utm_source" ou parâmetros de rastreio (como "src", "sck", "tracking_code", "metadata.utm_source", "metadata.src", "custom_params", etc) do payload do webhook é EXTREMAMENTE CRÍTICO. 

ATENÇÃO À SINTAXE JSONPATH:
A extração no nosso backend utiliza a biblioteca padrão de JSONPath.
É OBRIGATÓRIO que todos os caminhos comecem com "$." e incluam índices de array como "[0]" caso os dados reais estejam dentro de uma lista!

Você deve retornar ESTRITAMENTE um objeto JSON com as seguintes chaves:
- first_name (caminho para o primeiro nome ou nome completo do comprador)
- last_name (caminho para o sobrenome do comprador. SE o payload possuir apenas um campo com o NOME COMPLETO, coloque o MESMO CAMPO usado em first_name aqui também! Não deixe vazio.)
- email (caminho para o email do comprador)
- phone (caminho para o telefone/celular do comprador)
- transaction_id (caminho para o ID único da transação/venda)
- amount (caminho para o valor/preço da transação)
- status (caminho para o status da compra, ex: "approved", "waiting_payment")
- utm_source (CRÍTICO: caminho para o utm_source, src, sck, param de afiliado ou metadados de rastreio de origem onde estaria o nosso userId. Ex: "$.metadata.utm_source" ou "$.data[0].utm_source")
- status_map (um objeto onde a chave é o valor do status original no JSON que significa venda aprovada/paga ou boleto/pix gerado, mapeando para "Purchase" ou "InitiateCheckout").

Exemplo de formato de resposta caso os dados estejam dentro de um array "data":
{
  "first_name": "$.data[0].customer.first_name",
  "last_name": "$.data[0].customer.last_name",
  "email": "$.data[0].customer.email",
  "phone": "$.data[0].customer.mobile",
  "transaction_id": "$.data[0].transaction.id",
  "amount": "$.data[0].transaction.price",
  "status": "$.data[0].status",
  "utm_source": "$.data[0].metadata.utm_source",
  "status_map": {
    "approved": "Purchase",
    "waiting_payment": "InitiateCheckout"
  }
}
Retorne os caminhos exatos na sintaxe JSONPath padrão (iniciando com $). Não invente campos que não existem no payload recebido. Retorne SOMENTE JSON, sem tags markdown.`

	openAIReqBody := map[string]any{
		"model": "gpt-4o",
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": prompt},
			{"role": "user", "content": "Aqui está o payload:\n" + in.Payload},
		},
	}

	bodyBytes, _ := json.Marshal(openAIReqBody)
	req, _ := http.NewRequestWithContext(r.Context(), "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao conectar na OpenAI: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Erro da OpenAI (%d): %s", resp.StatusCode, string(respBody)))
		return
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao ler resposta da OpenAI")
		return
	}

	if len(openAIResp.Choices) == 0 {
		writeError(w, http.StatusInternalServerError, "resposta vazia da OpenAI")
		return
	}

	var finalResult map[string]any
	if err := json.Unmarshal([]byte(openAIResp.Choices[0].Message.Content), &finalResult); err != nil {
		writeError(w, http.StatusInternalServerError, "IA não retornou um JSON válido")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"mapping": finalResult,
	})
}
