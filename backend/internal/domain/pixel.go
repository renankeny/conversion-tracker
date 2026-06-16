package domain

import "time"

type Pixel struct {
	ID            int64         `json:"id"`
	Name          string        `json:"name"`
	Label         string        `json:"label"`
	PixelIDs      []string      `json:"pixel_ids"`
	AccessTokens  []string      `json:"access_tokens"`
	TestCodes     []string      `json:"test_codes"`
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

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HasTikTok retorna true se o pixel tem ao menos uma credencial TikTok configurada.
func (p Pixel) HasTikTok() bool {
	return len(p.TikTokPixelIDs) > 0 && len(p.TikTokAccessTokens) > 0
}

// PublicView esconde tokens (lista do painel mostra só prefixo+sufixo).
func (p Pixel) PublicView() map[string]any {
	return map[string]any{
		"id":                   p.ID,
		"name":                 p.Name,
		"label":                p.Label,
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
		"created_at":           p.CreatedAt,
		"updated_at":           p.UpdatedAt,
	}
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
