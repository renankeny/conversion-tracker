package util

import (
	"math"
	"testing"
)

func TestExtractMoneyFromMessage_AllVariations(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		trigger string
		want    float64
		wantOK  bool
	}{
		// 1. Formato brasileiro clássico com R$
		{"R$ com ponto e vírgula", "Pagamento confirmado! Valor R$ 1.500,00", "pagamento confirmado", 1500.00, true},
		{"R$ colado sem espaço", "Pagamento confirmado R$1.250,50", "pagamento confirmado", 1250.50, true},
		{"R$ sem centavos com ponto de milhar", "Venda realizada R$ 1.500", "venda realizada", 1500.00, true},
		{"R$ inteiro sem centavos", "Pagamento confirmado R$ 347", "pagamento confirmado", 347.00, true},
		{"R$ com 1 dígito decimal", "Pix confirmado R$ 99,9", "pix confirmado", 99.90, true},
		{"R$ em negrito no WhatsApp", "Pagamento confirmado *R$ 497,00*", "pagamento confirmado", 497.00, true},
		{"R$ com dois pontos de separador", "Venda fechada R$ 1.500.00", "venda fechada", 1500.00, true},
		{"R$ com duas vírgulas de separador", "Venda fechada R$ 1,500,00", "venda fechada", 1500.00, true},
		{"R$ com espaço no milhar", "Pagamento confirmado R$ 1 500,00", "pagamento confirmado", 1500.00, true},

		// 2. Sufixos escritos ("reais", "mil", "k", "conto", "pila")
		{"Escrito reais inteiro", "Pagamento confirmado de 350 reais", "pagamento confirmado", 350.00, true},
		{"Escrito reais com milhar", "Fechado 1.500 reais hoje", "fechado", 1500.00, true},
		{"Escrito reais e centavos", "Pagamento confirmado 150 reais e 50 centavos", "pagamento confirmado", 150.50, true},
		{"Multiplicador mil com vírgula", "Venda fechada por 1,5 mil", "venda fechada", 1500.00, true},
		{"Multiplicador mil com R$", "Pagamento confirmado R$ 2 mil", "pagamento confirmado", 2000.00, true},
		{"Multiplicador mil reais", "Fechado 2,5 mil reais", "fechado", 2500.00, true},
		{"Multiplicador k inteiro", "Venda confirmada 2k", "venda confirmada", 2000.00, true},
		{"Multiplicador k decimal ponto", "Venda confirmada 1.5k", "venda confirmada", 1500.00, true},
		{"Multiplicador k decimal vírgula", "Venda confirmada 1,8k", "venda confirmada", 1800.00, true},
		{"Gíria conto/pila", "Pix confirmado 300 conto", "pix confirmado", 300.00, true},

		// 3. Ancoragem de desconto ("de X por Y")
		{"De X por Y com R$", "Pagamento confirmado! Promoção de R$ 497,00 por R$ 297,00", "pagamento confirmado", 297.00, true},
		{"De X por Y sem R$", "Venda fechada de 1.000 por 750", "venda fechada", 750.00, true},
		{"De X por apenas Y", "Fechado de 500 por apenas 347,90", "fechado", 347.90, true},

		// 4. Parcelamento vs Total
		{"Total explícito + parcelas (prioriza total)", "Pagamento confirmado! Total R$ 1.000,00 em 4x de R$ 250,00", "pagamento confirmado", 1000.00, true},
		{"Apenas parcelamento 3x de 150 (calcula total)", "Venda fechada em 3x de 150", "venda fechada", 450.00, true},
		{"Apenas parcelamento 12x R$ 97,90", "Pagamento confirmado 12x de R$ 97,90", "pagamento confirmado", 1174.80, true},

		// 5. Palavras-chave contextuais sem R$ e sem centavos
		{"Valor: 347", "Compra aprovada! Valor: 347", "compra aprovada", 347.00, true},
		{"Valor 1500", "Pagamento confirmado valor 1500", "pagamento confirmado", 1500.00, true},
		{"Pix de 250", "Venda realizada, pix de 250", "venda realizada", 250.00, true},
		{"Total 490", "Pagamento confirmado total 490", "pagamento confirmado", 490.00, true},
		{"Fechado por 800", "Negócio fechado por 800", "negócio fechado", 800.00, true},
		{"Pago 197", "Pix confirmado, pago 197", "pix confirmado", 197.00, true},

		// 6. Números formatados sem R$
		{"Apenas vírgula e centavos", "Pagamento confirmado 297,90", "pagamento confirmado", 297.90, true},
		{"Apenas ponto e centavos", "Pagamento confirmado 297.90", "pagamento confirmado", 297.90, true},
		{"Apenas milhar com ponto", "Pagamento confirmado 1.250", "pagamento confirmado", 1250.00, true},

		// 7. Número puro junto ao gatilho
		{"Número puro após gatilho", "pagamento confirmado 347", "pagamento confirmado", 347.00, true},
		{"Número puro antes do gatilho", "347 pagamento confirmado", "pagamento confirmado", 347.00, true},

		// 8. Blindagem contra falsos positivos (datas, horas, telefones, protocolos, %)
		{"Ignora data e hora e pega valor", "Pagamento confirmado dia 10/11/2026 às 14:30 valor 397", "pagamento confirmado", 397.00, true},
		{"Ignora protocolo e telefone e pega valor", "Protocolo: ABC1234567 tel (11) 98888-7777 pagamento confirmado 450", "pagamento confirmado", 450.00, true},
		{"Ignora porcentagem e pega valor", "Com 20% de desconto, pagamento confirmado 400", "pagamento confirmado", 400.00, true},
		{"Sem valor na mensagem (só data/hora/protocolo)", "Pagamento confirmado dia 10/11 às 14:30! Protocolo: XYZ987654", "pagamento confirmado", 0, false},
		{"Gatilho com número não vira valor", "Etapa 2 concluída com sucesso", "etapa 2", 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ExtractMoneyFromMessage(tc.text, tc.trigger)
			if ok != tc.wantOK {
				t.Fatalf("ExtractMoneyFromMessage(%q) ok = %v, want %v (got val=%v)", tc.text, ok, tc.wantOK, got)
			}
			if tc.wantOK && math.Abs(got-tc.want) > 0.001 {
				t.Fatalf("ExtractMoneyFromMessage(%q) = %.2f, want %.2f", tc.text, got, tc.want)
			}
		})
	}
}

