package domain

import "time"

// ThemePalette representa as cores principais de um modo (Dark ou Light).
type ThemePalette struct {
	Brand       string `json:"brand"`
	Surface1    string `json:"surface_1"`
	Surface2    string `json:"surface_2"`
	SurfaceDeep string `json:"surface_deep"`
}

// BrandingThemeConfig armazena o preset selecionado e as paletas Dark e Light.
type BrandingThemeConfig struct {
	Preset string       `json:"preset"`
	Dark   ThemePalette `json:"dark"`
	Light  ThemePalette `json:"light"`
}

// PlatformBranding representa a configuração Whitelabel global da plataforma.
type PlatformBranding struct {
	PlatformName    string              `json:"platform_name"`
	Tagline         string              `json:"tagline"`
	FooterText      string              `json:"footer_text"`
	LogoDisplayMode string              `json:"logo_display_mode"` // "icon_text" | "logo_only" | "logo_and_text"
	LogoDarkURL     string              `json:"logo_dark_url"`
	LogoLightURL    string              `json:"logo_light_url"`
	FaviconURL      string              `json:"favicon_url"`
	ThemeConfig     BrandingThemeConfig `json:"theme_config"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

// DefaultPlatformBranding retorna os valores padrão da plataforma caso o banco ainda não tenha registro.
func DefaultPlatformBranding() PlatformBranding {
	return PlatformBranding{
		PlatformName:    "Sua Plataforma",
		Tagline:         "Intelligence & Attribution",
		FooterText:      "",
		LogoDisplayMode: "icon_text",
		LogoDarkURL:     "",
		LogoLightURL:    "",
		FaviconURL:      "",
		ThemeConfig: BrandingThemeConfig{
			Preset: "emerald",
			Dark: ThemePalette{
				Brand:       "#A2F07C",
				Surface1:    "#29292E",
				Surface2:    "#323238",
				SurfaceDeep: "#202024",
			},
			Light: ThemePalette{
				Brand:       "#0B4B4D",
				Surface1:    "#FFFFFF",
				Surface2:    "#FFFFFF",
				SurfaceDeep: "#F1F5F9",
			},
		},
		UpdatedAt: time.Now(),
	}
}

