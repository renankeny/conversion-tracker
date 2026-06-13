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
	TrackingHost string    `json:"tracking_host"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PublicView esconde tokens (lista do painel mostra só prefixo+sufixo).
func (p Pixel) PublicView() map[string]any {
	return map[string]any{
		"id":             p.ID,
		"name":           p.Name,
		"label":          p.Label,
		"pixel_ids":      p.PixelIDs,
		"access_tokens":  maskAll(p.AccessTokens),
		"test_codes":     p.TestCodes,
		"domains":        p.Domains,
		"tracker_config": p.TrackerConfig,
		"enabled":        p.Enabled,
		"tracking_host":  p.TrackingHost,
		"pixel_count":    len(p.PixelIDs),
		"created_at":     p.CreatedAt,
		"updated_at":     p.UpdatedAt,
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
