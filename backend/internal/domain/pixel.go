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
// (somente leitura) da dashboard de um pixel: 15 dígitos, começando pelo
// próprio ID do pixel e completados com dígitos aleatórios.
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
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Label       string `json:"label"`
	PublicToken string `json:"public_token"`

	PixelIDs      []string      `json:"pixel_ids"`
	AccessTokens  []string      `json:"access_tokens"`
	TestCodes     []string      `json:"test_codes"`
	Domains       []string      `json:"domains"`
	TrackerConfig TrackerConfig `json:"tracker_config"`
	Enabled       bool          `json:"enabled"`
	TrackingHost  string        `json:"tracking_host"`
	AccountID     int64         `json:"account_id"`

	// TikTok Events API (arrays paralelos por índice)
	TikTokPixelIDs     []string `json:"tiktok_pixel_ids"`
	TikTokAccessTokens []string `json:"tiktok_access_tokens"`
	TikTokTestCodes    []string `json:"tiktok_test_codes"`

	// Google Ads (vínculo OAuth via google_ads_accounts + Customer ID específico do pixel)
	GoogleAdsAccountID       *int64 `json:"google_ads_account_id,omitempty"`
	GoogleAdsCustomerID      string `json:"google_ads_customer_id"`
	GoogleAdsCustomerName    string `json:"google_ads_customer_name"`
	GoogleAdsLoginCustomerID string `json:"google_ads_login_customer_id"`

	// Campos legados mantidos apenas para leitura compatível com o schema
	GoogleAdsConversionActionID string `json:"-"`
	GoogleAdsRefreshToken       string `json:"-"`
	GoogleAdsOAuthEmail         string `json:"-"`

	// GA4 Measurement Protocol
	GA4MeasurementID string `json:"ga4_measurement_id"`
	GA4APISecret     string `json:"ga4_api_secret"`

	// Meta Permanent Token, Ad Account & OAuth Account
	MetaPermanentToken string `json:"meta_permanent_token"`
	MetaAdAccountID    string `json:"meta_ad_account_id"`
	MetaAccountID      *int64 `json:"meta_account_id,omitempty"`

	// Dashboard web por pixel e link público de visualização
	DashboardConfig     DashboardConfig `json:"dashboard_config"`
	DashboardShareToken string          `json:"dashboard_share_token"`

	// Agrupamento de Conversões (Order Bump & Upsell)
	AggregationEnabled       bool `json:"aggregation_enabled"`
	AggregationWindowMinutes int  `json:"aggregation_window_minutes"`

	// Meta de Vendas por Pixel ("revenue" = valor financeiro | "count" = quantidade de vendas)
	SalesGoalEnabled bool               `json:"sales_goal_enabled"`
	SalesGoalType    string             `json:"sales_goal_type"`
	SalesGoalTarget  float64            `json:"sales_goal_target"`
	SalesGoalMonths  map[string]float64 `json:"sales_goal_months"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NormalizeSalesGoalType(t string) string {
	if t == "count" {
		return "count"
	}
	return "revenue"
}

func (p Pixel) HasTikTok() bool {
	return len(p.TikTokPixelIDs) > 0 && len(p.TikTokAccessTokens) > 0
}

func (p Pixel) HasGoogleAds() bool {
	return p.GoogleAdsAccountID != nil && *p.GoogleAdsAccountID > 0
}

func (p Pixel) HasMetaOAuth() bool {
	return p.MetaAccountID != nil && *p.MetaAccountID > 0
}

// PublicView mascara segredos sensíveis antes de enviar ao painel administrativo.
func (p Pixel) PublicView() map[string]any {
	return map[string]any{
		"id":                         p.ID,
		"name":                       p.Name,
		"label":                      p.Label,
		"public_token":               p.PublicToken,
		"pixel_ids":                  p.PixelIDs,
		"access_tokens":              maskAll(p.AccessTokens),
		"test_codes":                 p.TestCodes,
		"domains":                    p.Domains,
		"tracker_config":             p.TrackerConfig,
		"enabled":                    p.Enabled,
		"tracking_host":              p.TrackingHost,
		"account_id":                 p.AccountID,
		"pixel_count":                len(p.PixelIDs),
		"tiktok_pixel_ids":           p.TikTokPixelIDs,
		"tiktok_access_tokens":       maskAll(p.TikTokAccessTokens),
		"tiktok_test_codes":          p.TikTokTestCodes,
		"tiktok_pixel_count":         len(p.TikTokPixelIDs),
		"google_ads_account_id":        p.GoogleAdsAccountID,
		"google_ads_customer_id":       p.GoogleAdsCustomerID,
		"google_ads_customer_name":     p.GoogleAdsCustomerName,
		"google_ads_login_customer_id": p.GoogleAdsLoginCustomerID,
		"ga4_measurement_id":         p.GA4MeasurementID,
		"ga4_api_secret":             mask(p.GA4APISecret),
		"meta_permanent_token":       mask(p.MetaPermanentToken),
		"meta_ad_account_id":         p.MetaAdAccountID,
		"meta_account_id":            p.MetaAccountID,
		"dashboard_config":           p.DashboardConfig,
		"dashboard_share_token":      p.DashboardShareToken,
		"aggregation_enabled":        p.AggregationEnabled,
		"aggregation_window_minutes": p.AggregationWindowMinutes,
		"sales_goal_enabled":         p.SalesGoalEnabled,
		"sales_goal_type":            NormalizeSalesGoalType(p.SalesGoalType),
		"sales_goal_target":          p.SalesGoalTarget,
		"sales_goal_months":          NormalizeSalesGoalMonths(p.SalesGoalMonths),
		"created_at":                 p.CreatedAt,
		"updated_at":                 p.UpdatedAt,
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
		if s == "" {
			out[i] = ""
			continue
		}
		if len(s) <= 8 {
			out[i] = "****"
			continue
		}
		out[i] = s[:4] + "..." + s[len(s)-4:]
	}
	return out
}
