package util

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Mascaramento de falsos positivos (URLs, protocolos, datas, horários, telefones, CPF/CNPJ/CEP, %, durações/quantidades).
	rxURLMask      = regexp.MustCompile(`(?i)https?://\S+`)
	rxProtoMask    = regexp.MustCompile(`(?i)\b(?:protocolo(?:\s+de\s+atendimento)?|c[oó]d(?:igo)?(?:\s+de\s+atendimento)?|ref(?:er[eê]ncia)?|pedido|id)\s*[:#\-]?\s*\*?[a-z0-9_-]{4,36}\*?`)
	rxCPFMask      = regexp.MustCompile(`\b\d{3}\.\d{3}\.\d{3}-\d{2}\b|\b\d{2}\.\d{3}\.\d{3}/\d{4}-\d{2}\b|\b\d{5}-\d{3}\b`)
	rxDateMask     = regexp.MustCompile(`(?i)(?:\bdia\s+)?\b\d{1,2}/\d{1,2}(?:/\d{2,4})?\b|\b\d{4}-\d{2}-\d{2}\b|\bdia\s+\d{1,2}\b`)
	rxTimeMask     = regexp.MustCompile(`(?i)\b\d{1,2}:\d{2}(?::\d{2})?\b|\b\d{1,2}\s*h(?:s|oras?)?(?:\s*\d{1,2}\s*(?:m|min)?)?\b|\b\d+\s*(?:min|minutos?|seg|segundos?)\b`)
	rxPhoneMask    = regexp.MustCompile(`(?:\+?55\s*)?\(?\d{2}\)?\s*9\d{4}[-\s]?\d{4}\b|\b55\d{10,11}\b|\b\d{10,14}\b`)
	rxPercentMask  = regexp.MustCompile(`\b\d+(?:[.,]\d+)?\s*%`)
	rxUnitMask     = regexp.MustCompile(`(?i)\b\d+\s*(?:dias?|mes(?:es)?|anos?|semanas?|unidades?|unid|itens|item|pe[cç]as?|vagas?|lotes?|mg|ml|kg|km|º|ª|°)\b`)
	rxSpaceThous   = regexp.MustCompile(`\b(\d{1,3})\s(\d{3}(?:,\d{1,2})\b)`)
	rxInstallMask  = regexp.MustCompile(`(?i)\b\d{1,2}\s*x\s*(?:de\s*)?(?:R\$\s*|BRL\s*|\$\s*)?\d+(?:[.,]\d+)*(?:\s*(?:mil|k|reais|real))?`)

	// Padrões de extração em ordem de especificidade.
	numTokenPat = `\d{1,3}(?:[.,]\d{3})+(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,3})?`

	// 1. "X reais e Y centavos"
	rxReaisECentavos = regexp.MustCompile(`(?i)\b(` + numTokenPat + `)\s*reais?\s+e\s+(\d{1,2})\s*centavos?\b`)

	// 2. "de X por Y" (ancoragem de preço / desconto -> captura o valor final após "por")
	rxDePor = regexp.MustCompile(`(?i)\bde\s+(?:R\$\s*|BRL\s*|\$\s*)?(?:` + numTokenPat + `)(?:\s*(?:mil|k|reais|real))?\s+por\s+(?:apenas\s+)?(?:R\$\s*|BRL\s*|\$\s*)?(` + numTokenPat + `)\s*(mil|k)?(?:\s*(?:reais|real|conto|contos|pila))?`)

	// 3. Moeda explícita: R$ 1.500,00 | R$ 1500 | R$ 1,5 mil | R$ 2k | BRL 350 | US$ 100
	rxExplicitPrefix = regexp.MustCompile(`(?i)(?:R\$|BRL|US\$|\$)\s*\*?(` + numTokenPat + `)\*?\s*(mil|k)?\b`)

	// 4. Sufixo explícito de moeda ou multiplicador: 350 reais | 1.500 reais | 1,5 mil | 2k | 300 conto | 250 pila
	rxExplicitSuffix = regexp.MustCompile(`(?i)\b(` + numTokenPat + `)\s*(mil|k)?\s*(reais|real|brl|conto|contos|pila)\b|\b(` + numTokenPat + `)\s*(mil|k)\b`)

	// 5. Palavra-chave de contexto financeiro: valor: 347 | total 1.500 | pix de 250 | fechado por 1200 | pago 300
	rxContextKeyword = regexp.MustCompile(`(?i)\b(?:valor(?:\s+total|\s+final|\s+pago|\s+l[ií]quido)?|vlr|total(?:\s+pago|\s+final|\s+geral)?|pre[cç]o(?:\s+final)?|pix(?:\s+de|\s+no\s+valor\s+de|\s+confirmado|\s+recebido)?|pago|pagamento(?:\s+de|\s+confirmado|\s+aprovado)?|recebido|fechado(?:\s+por|\s+em|\s+no\s+valor\s+de)?|fechou(?:\s+por|\s+em)?|por(?:\s+apenas)?|sinal(?:\s+de)?|entrada(?:\s+de)?|boleto(?:\s+de)?|cart[aã]o(?:\s+de)?|transfer[eê]ncia(?:\s+de)?|venda(?:\s+de|\s+fechada|\s+realizada|\s+confirmada)?|compra(?:\s+aprovada|\s+confirmada)?)\s*(?:de|em|por|no\s+valor\s+de|[:=\-])?\s*\*?(?:R\$\s*|BRL\s*|\$\s*)?(` + numTokenPat + `)\*?\s*(mil|k)?\b`)

	// 6. Parcelamento: "3x de 150" ou "12x R$ 97,90" -> multiplica N * valor da parcela quando não há total explícito
	rxInstallment = regexp.MustCompile(`(?i)\b(\d{1,2})\s*x\s*(?:de\s*)?(?:R\$\s*|BRL\s*|\$\s*)?(` + numTokenPat + `)\b`)

	// 7. Número formatado com centavos ou milhar brasileiro (ex: 1.500,00 | 347,90 | 347.00 | 1.500)
	rxFormattedNum = regexp.MustCompile(`\b(\d{1,3}(?:\.\d{3})+,\d{1,2}|\d+,\d{1,2}|\d{1,3}(?:,\d{3})+\.\d{1,2}|\d+\.\d{2}|\d{1,3}(?:\.\d{3})+)\b`)

	// 8. Número puro remanescente quando já sabemos que a mensagem bateu em um gatilho de conversão
	rxBareNumber = regexp.MustCompile(`\b(\d{1,6})\b`)
)

// ExtractMoneyFromMessage extrai de forma inteligente o valor monetário de uma mensagem de WhatsApp.
// Quando allowBareNumber é true (ex: gatilho de Purchase acionado), também aceita números inteiros
// soltos após remover datas, horários, telefones, protocolos e o próprio texto do gatilho.
func ExtractMoneyFromMessage(text, triggerPhrase string, allowBareNumber ...bool) (float64, bool) {
	if strings.TrimSpace(text) == "" {
		return 0, false
	}
	allowBare := true
	if len(allowBareNumber) > 0 {
		allowBare = allowBareNumber[0]
	}

	cleaned := sanitizeNonMonetaryTokens(text, triggerPhrase)

	// 1. "150 reais e 50 centavos"
	if m := rxReaisECentavos.FindStringSubmatch(cleaned); len(m) >= 3 {
		reais := parseFlexibleMoneyToken(m[1], "")
		cents, _ := strconv.ParseFloat(m[2], 64)
		if total := roundCents(reais + cents/100.0); total > 0 {
			return total, true
		}
	}

	// 2. "de R$ 497 por R$ 297" -> prioriza o preço final ("por")
	if matches := rxDePor.FindAllStringSubmatch(cleaned, -1); len(matches) > 0 {
		last := matches[len(matches)-1]
		if v := parseFlexibleMoneyToken(last[1], last[2]); isValidMoneyRange(v) {
			return v, true
		}
	}

	// Versão sem trechos "Nx de Y" para evitar pegar o valor de 1 parcela nos passos 3-5;
	// se só houver o parcelamento na frase, o passo 6 calcula N * parcela.
	targetText := strings.TrimSpace(rxInstallMask.ReplaceAllString(cleaned, " "))

	// 3. Prefixo explícito (R$, BRL, $)
	if matches := rxExplicitPrefix.FindAllStringSubmatch(targetText, -1); len(matches) > 0 {
		last := matches[len(matches)-1]
		if v := parseFlexibleMoneyToken(last[1], last[2]); isValidMoneyRange(v) {
			return v, true
		}
	}

	// 4. Sufixo explícito (reais, mil, k, conto, pila)
	if matches := rxExplicitSuffix.FindAllStringSubmatch(targetText, -1); len(matches) > 0 {
		last := matches[len(matches)-1]
		numStr, multStr := last[1], last[2]
		if numStr == "" {
			numStr, multStr = last[4], last[5]
		}
		if v := parseFlexibleMoneyToken(numStr, multStr); isValidMoneyRange(v) {
			return v, true
		}
	}

	// 5. Palavra-chave contextual ("valor 347", "pix de 250", "fechado por 1500", "pago 300")
	if matches := rxContextKeyword.FindAllStringSubmatch(targetText, -1); len(matches) > 0 {
		last := matches[len(matches)-1]
		if v := parseFlexibleMoneyToken(last[1], last[2]); isValidMoneyRange(v) {
			return v, true
		}
	}

	// 6. Parcelamento isolado ("3x de 150", "12x R$ 97,90") -> calcula o valor total N * parcela
	if matches := rxInstallment.FindAllStringSubmatch(cleaned, -1); len(matches) > 0 {
		last := matches[len(matches)-1]
		n, err := strconv.Atoi(last[1])
		installVal := parseFlexibleMoneyToken(last[2], "")
		if err == nil && n >= 1 && n <= 48 && installVal > 0 {
			if total := roundCents(float64(n) * installVal); isValidMoneyRange(total) {
				return total, true
			}
		}
	}

	// 7. Número formatado com casas decimais ou ponto de milhar (ex: "1.500,00", "347,90", "1.500")
	if matches := rxFormattedNum.FindAllStringSubmatch(cleaned, -1); len(matches) > 0 {
		last := matches[len(matches)-1]
		if v := parseFlexibleMoneyToken(last[1], ""); isValidMoneyRange(v) {
			return v, true
		}
	}

	// 8. Número inteiro simples junto ao gatilho (ex: "pagamento confirmado 347")
	if allowBare {
		if matches := rxBareNumber.FindAllStringSubmatch(cleaned, -1); len(matches) > 0 {
			last := matches[len(matches)-1]
			if v := parseFlexibleMoneyToken(last[1], ""); isValidMoneyRange(v) {
				return v, true
			}
		}
	}

	return 0, false
}

func sanitizeNonMonetaryTokens(text, triggerPhrase string) string {
	s := rxURLMask.ReplaceAllString(text, " ")
	s = rxProtoMask.ReplaceAllString(s, " ")
	s = rxCPFMask.ReplaceAllString(s, " ")
	s = rxPhoneMask.ReplaceAllString(s, " ")
	s = rxDateMask.ReplaceAllString(s, " ")
	s = rxTimeMask.ReplaceAllString(s, " ")
	s = rxPercentMask.ReplaceAllString(s, " ")
	s = rxUnitMask.ReplaceAllString(s, " ")
	s = rxSpaceThous.ReplaceAllString(s, "$1.$2")

	// Se o próprio gatilho configurado contiver dígitos (ex: "etapa 2"), remove apenas os dígitos do gatilho.
	for _, part := range strings.FieldsFunc(triggerPhrase, func(r rune) bool { return r == ',' || r == ';' || r == '|' }) {
		p := strings.TrimSpace(part)
		if p != "" && strings.ContainsAny(p, "0123456789") {
			if rx, err := regexp.Compile(`(?i)` + regexp.QuoteMeta(p)); err == nil {
				s = rx.ReplaceAllString(s, " ")
			}
		}
	}
	return s
}

func parseFlexibleMoneyToken(rawNum, rawMult string) float64 {
	s := strings.TrimSpace(rawNum)
	if s == "" {
		return 0
	}
	mult := 1.0
	mLower := strings.ToLower(strings.TrimSpace(rawMult))
	if mLower == "mil" || mLower == "k" {
		mult = 1000.0
	}

	hasComma := strings.Contains(s, ",")
	hasDot := strings.Contains(s, ".")

	switch {
	case mult == 1000.0:
		// Em "1,5 mil" ou "1.5k", vírgula ou ponto é separador decimal do milhar
		s = strings.ReplaceAll(s, ",", ".")
	case hasComma && hasDot:
		if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	case hasComma:
		if strings.Count(s, ",") > 1 {
			last := strings.LastIndex(s, ",")
			if len(s)-last-1 <= 2 {
				s = strings.ReplaceAll(s[:last], ",", "") + "." + s[last+1:]
			} else {
				s = strings.ReplaceAll(s, ",", "")
			}
		} else {
			after := len(s) - strings.IndexByte(s, ',') - 1
			if after == 3 {
				s = strings.ReplaceAll(s, ",", "")
			} else {
				s = strings.Replace(s, ",", ".", 1)
			}
		}
	case hasDot:
		if strings.Count(s, ".") > 1 {
			last := strings.LastIndex(s, ".")
			if len(s)-last-1 <= 2 {
				s = strings.ReplaceAll(s[:last], ".", "") + "." + s[last+1:]
			} else {
				s = strings.ReplaceAll(s, ".", "")
			}
		} else {
			after := len(s) - strings.IndexByte(s, '.') - 1
			if after == 3 {
				// No Brasil, 3 dígitos após ponto (ex: "1.500", "2.000") é milhar!
				s = strings.ReplaceAll(s, ".", "")
			}
		}
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil || val <= 0 {
		return 0
	}
	return roundCents(val * mult)
}

func roundCents(v float64) float64 {
	return math.Round(v*100) / 100
}

func isValidMoneyRange(v float64) bool {
	return v >= 0.5 && v <= 10_000_000
}
