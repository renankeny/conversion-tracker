package traffic

import (
	"net/url"
	"strings"
)

type Rule struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Impact      string `json:"impact"`
	Weight      int    `json:"weight"`
	Description string `json:"description"`
}

type Result struct {
	Type       string `json:"type"`
	Category   string `json:"category"`
	Score      int    `json:"score"`
	Confidence string `json:"confidence"`
	Browser    string `json:"browser"`
	OS         string `json:"os"`
	Device     string `json:"device"`
	Rules      []Rule `json:"rules"`
	Disclaimer string `json:"disclaimer"`
}

const DisclaimerText = "Esta classificação é uma estimativa probabilística baseada em heurísticas de cabeçalho HTTP e ambiente de navegação. Não constitui confirmação definitiva."

func Classify(rawUA string, rawReferrer string, isWebdriver bool) Result {
	ua := strings.ToLower(strings.TrimSpace(rawUA))
	ref := strings.ToLower(strings.TrimSpace(rawReferrer))

	var rules []Rule
	score := 0
	category := "none"

	browser := detectBrowser(ua)
	os := detectOS(ua)
	device := detectDevice(ua)

	if isAdVerifierUA(ua) {
		category = "ad_verifier"
		score -= 80
		rules = append(rules, Rule{
			ID:          "ua_ad_verifier",
			Label:       "Crawler de Plataforma de Anúncios / Pré-visualização",
			Impact:      "automated",
			Weight:      -80,
			Description: "User-Agent característico de robô de inspeção de anúncios ou gerador de links (Meta, Google Ads, TikTok, etc.).",
		})
	} else if isSearchBotUA(ua) {
		category = "search_engine"
		score -= 75
		rules = append(rules, Rule{
			ID:          "ua_search_bot",
			Label:       "Crawler de Motor de Busca",
			Impact:      "automated",
			Weight:      -75,
			Description: "User-Agent identificado como indexador de pesquisa web (Googlebot, Bingbot, etc.).",
		})
	} else if isHeadlessScraperUA(ua) {
		category = "headless_scraper"
		score -= 90
		rules = append(rules, Rule{
			ID:          "ua_headless_scraper",
			Label:       "Ambiente Headless / Automação / Scraper",
			Impact:      "automated",
			Weight:      -90,
			Description: "Identificado software de automação sem interface visual, biblioteca de raspagem ou bot de inteligência artificial.",
		})
	} else if isGenericBotUA(ua) {
		category = "generic_bot"
		score -= 60
		rules = append(rules, Rule{
			ID:          "ua_generic_bot",
			Label:       "Assinatura Genérica de Robô",
			Impact:      "automated",
			Weight:      -60,
			Description: "Cabeçalho User-Agent inclui termos explícitos de robô/rastreador web.",
		})
	}

	if ua == "" || len(ua) < 20 {
		score -= 50
		rules = append(rules, Rule{
			ID:          "ua_too_short",
			Label:       "User-Agent Ausente ou Anômalo",
			Impact:      "automated",
			Weight:      -50,
			Description: "Acesso sem identificação válida de navegador web moderno.",
		})
	}

	if isWebdriver {
		score -= 70
		if category == "none" {
			category = "headless_scraper"
		}
		rules = append(rules, Rule{
			ID:          "runtime_webdriver",
			Label:       "Controle Remoto de Browser Ativo (navigator.webdriver)",
			Impact:      "automated",
			Weight:      -70,
			Description: "A flag nativa de controle remoto automatizado do navegador foi detectada pelo script rastreador.",
		})
	}

	isKnownBrowserAndOS := browser != "Navegador Desconhecido" && os != "SO Desconhecido"
	if isKnownBrowserAndOS && category == "none" {
		score += 40
		rules = append(rules, Rule{
			ID:          "ua_legitimate_browser",
			Label:       "Navegador e Dispositivo Padrão Reconhecidos",
			Impact:      "human",
			Weight:      +40,
			Description: "Ambiente compatível com navegadores comerciais de uso pessoal (" + browser + " em " + os + ").",
		})
	}

	if ref != "" {
		if parsed, err := url.Parse(ref); err == nil && parsed.Host != "" {
			host := strings.ToLower(parsed.Host)
			if isSocialOrSearchHost(host) {
				score += 25
				rules = append(rules, Rule{
					ID:          "ref_social_ad_search",
					Label:       "Origem em Rede Social, Anúncio ou Buscador (" + host + ")",
					Impact:      "human",
					Weight:      +25,
					Description: "A navegação é derivada de um ecossistema reconhecido de tráfego de usuários.",
				})
			} else {
				score += 10
				rules = append(rules, Rule{
					ID:          "ref_external",
					Label:       "Referrer Externo (" + host + ")",
					Impact:      "human",
					Weight:      +10,
					Description: "A navegação possui origem externa declarada com URL válida.",
				})
			}
		}
	} else if isKnownBrowserAndOS && category == "none" {
		score += 20
		rules = append(rules, Rule{
			ID:          "ref_direct_known_browser",
			Label:       "Acesso Direto / Digitação de URL / Favoritos",
			Impact:      "human",
			Weight:      +20,
			Description: "Visitante acessou digitando o endereço diretamente, usando favoritos ou abrindo link em aplicativo privado.",
		})
	} else {
		rules = append(rules, Rule{
			ID:          "ref_direct_empty",
			Label:       "Acesso Direto Sem Referrer",
			Impact:      "neutral",
			Weight:      0,
			Description: "Nenhum cabeçalho de origem fornecido.",
		})
	}

	if score > 100 {
		score = 100
	} else if score < -100 {
		score = -100
	}

	trafficType, confidence := "inconclusive", "baixa"
	if score >= 35 {
		trafficType = "human"
		if score >= 60 {
			confidence = "alta"
		} else {
			confidence = "media"
		}
	} else if score <= -30 {
		trafficType = "bot"
		if score <= -60 {
			confidence = "alta"
		} else {
			confidence = "media"
		}
	}

	return Result{
		Type:       trafficType,
		Category:   category,
		Score:      score,
		Confidence: confidence,
		Browser:    browser,
		OS:         os,
		Device:     device,
		Rules:      rules,
		Disclaimer: DisclaimerText,
	}
}
