package domain

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"strconv"
	"time"
)

// GeneratePublicToken gera um token opaco de 32 chars hex (128 bits de entropia).
// Usado como identificador público do pixel no snippet HTML.
func GeneratePublicToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateDashboardShareToken gera o token numérico do link público
// (view-only) da dashboard de um pixel: 15 dígitos, sempre começando pelo
// próprio ID do pixel (facilita suporte/depuração) e completados com dígitos
// aleatórios. Curto e só numérico de propósito — é um link pra compartilhar
// com o cliente, não um segredo de autenticação.
func GenerateDashboardShareToken(pixelID int64) string {
	const totalLen = 15
	idStr := strconv.FormatInt(pixelID, 10)
	if len(idStr) >= totalLen {
		return idStr
	}
	digits := make([]byte, totalLen-len(idStr))
	ten := big.NewInt(10)
	for i := range digits {
		n, err := rand.Int(rand.Reader, ten)
		if err != nil {
			digits[i] = '0'
			continue
		}
		digits[i] = byte('0') + byte(n.Int64())
	}
	return idStr + string(digits)
}

type Pixel struct {
	ID           int64         `json:"id"`
	Name         string        `json:"name"`
	Label        string        `json:"label"`
	// PublicToken é o identificador opaco usado no snippet HTML (?pixel=<token>).
	// Aceito nos endpoints de eventos como alternativa ao Name.
	PublicToken  string        `json:"public_token"`
	PixelIDs     []string      `json:"pixel_ids"`
	AccessTokens []string      `json:"access_tokens"`
	TestCodes    []string      `json:"test_codes"`
	Domains       []string      `json:"domains"`
	TrackerConfig TrackerConfig `json:"tracker_config"`
	Enabled       bool          `json:"enabled"`
	// TrackingHost é o subdomínio first-party do cliente (ex: "tracking.cliente.com").
	// Quando preenchido, o snippet usa esse host em vez do host padrão da API.
	TrackingHost string `json:"tracking_host"`
	// AccountID vincula o pixel a uma conta (multi-tenant). Default 1 = conta principal.
	AccountID int64 `json:"account_id"`

	// === TikTok Events API ============================================
	// Arrays paralelos: índice i de TikTokPixelIDs casa com i de
	// TikTokAccessTokens e TikTokTestCodes. Quando vazios, eventos não
	// são enviados pro TikTok (apenas pra Meta).
	TikTokPixelIDs     []string `json:"tiktok_pixel_ids"`
	TikTokAccessTokens []string `json:"tiktok_access_tokens"`
	TikTokTestCodes    []string `json:"tiktok_test_codes"`

	// === Google Ads (Conversões offline via API) ======================
	// Conexao com Google Ads vive em google_ads_accounts (entidade
	// propria — Google Ads trabalha por GCLID, nao por pixel). Aqui
	// guardamos so a referencia opcional. Os campos google_ads_*
	// abaixo sao legados da migration 0011 e ficam pra compat
	// reversa, sem uso na pratica.
	GoogleAdsAccountID *int64 `json:"google_ads_account_id,omitempty"`

	// Deprecated: usar GoogleAdsAccountID + tabela google_ads_accounts.
	GoogleAdsCustomerID         string `json:"-"`
	GoogleAdsLoginCustomerID    string `json:"-"`
	GoogleAdsConversionActionID string `json:"-"`
	GoogleAdsRefreshToken       string `json:"-"`
	GoogleAdsOAuthEmail         string `json:"-"`

	// === GA4 Measurement Protocol =====================================
	GA4MeasurementID string `json:"ga4_measurement_id"`
	GA4APISecret     string `json:"ga4_api_secret"`

	// === Meta Permanent Token, Ad Account & OAuth Account =============
	MetaPermanentToken string `json:"meta_permanent_token"`
	MetaAdAccountID    string `json:"meta_ad_account_id"`
	MetaAccountID      *int64 `json:"meta_account_id,omitempty"`

	// === Dashboard web (cards, ordem, funil, nomes) — por pixel =========
	DashboardConfig DashboardConfig `json:"dashboard_config"`
	// DashboardShareToken é o token opaco do link público de visualização
	// (view-only, sem login). Vazio = link não gerado / desativado.
	DashboardShareToken string `json:"dashboard_share_token"`

	// === Agrupamento de Conversões (Order Bump & Upsell) ===============
	AggregationEnabled       bool `json:"aggregation_enabled"`
	AggregationWindowMinutes int  `json:"aggregation_window_minutes"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HasTikTok retorna true se o pixel tem ao menos uma credencial TikTok configurada.
func (p Pixel) HasTikTok() bool {
	return len(p.TikTokPixelIDs) > 0 && len(p.TikTokAccessTokens) > 0
}

// HasGoogleAds retorna true se o pixel esta vinculado a uma conta
// Google Ads. A verificacao de conexao real (refresh_token valido)
// fica no GoogleAdsAccount.IsConnected().
func (p Pixel) HasGoogleAds() bool {
	return p.GoogleAdsAccountID != nil && *p.GoogleAdsAccountID > 0
}

func (p Pixel) HasMetaOAuth() bool {
	return p.MetaAccountID != nil && *p.MetaAccountID > 0
}

// PublicView esconde tokens (lista do painel mostra só prefixo+sufixo).
func (p Pixel) PublicView() map[string]any {
	return map[string]any{
		"id":                   p.ID,
		"name":                 p.Name,
		"label":                p.Label,
		"public_token":         p.PublicToken,
		"pixel_ids":            p.PixelIDs,
		"access_tokens":        maskAll(p.AccessTokens),
		"test_codes":           p.TestCodes,
		"domains":              p.Domains,
		"tracker_config":       p.TrackerConfig,
		"enabled":              p.Enabled,
		"tracking_host":        p.TrackingHost,
		"account_id":           p.AccountID,
		"pixel_count":          len(p.PixelIDs),
		"tiktok_pixel_ids":     p.TikTokPixelIDs,
		"tiktok_access_tokens": maskAll(p.TikTokAccessTokens),
		"tiktok_test_codes":    p.TikTokTestCodes,
		"tiktok_pixel_count":   len(p.TikTokPixelIDs),

		// Google Ads: vinculado por google_ads_account_id (entidade separada).
		"google_ads_account_id": p.GoogleAdsAccountID,

		// GA4 Measurement Protocol
		"ga4_measurement_id": p.GA4MeasurementID,
		"ga4_api_secret":     mask(p.GA4APISecret),

		// Meta Permanent Token, Ad Account & OAuth Account
		"meta_permanent_token": mask(p.MetaPermanentToken),
		"meta_ad_account_id":   p.MetaAdAccountID,
		"meta_account_id":      p.MetaAccountID,

		// Dashboard web (cards/ordem/funil/nomes) + link público view-only
		"dashboard_config":      p.DashboardConfig,
		"dashboard_share_token": p.DashboardShareToken,

		// Agrupamento de Conversões (Order Bump & Upsell)
		"aggregation_enabled":        p.AggregationEnabled,
		"aggregation_window_minutes": p.AggregationWindowMinutes,

		"created_at": p.CreatedAt,
		"updated_at": p.UpdatedAt,
	}
}

func mask(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func maskAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		if len(s) <= 8 {
			out[i] = "****"
			continue
		}
		out[i] = s[:4] + "..." + s[len(s)-4:]
	}
	return out
}
