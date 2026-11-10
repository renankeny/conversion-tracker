export type LogoDisplayMode = 'icon_text' | 'logo_only' | 'logo_and_text'

export interface ThemePalette {
  brand: string
  surface_1: string
  surface_2: string
  surface_deep: string
}

export interface BrandingThemeConfig {
  preset: string
  dark: ThemePalette
  light: ThemePalette
}

export interface PlatformBranding {
  platform_name: string
  tagline: string
  footer_text: string
  logo_display_mode: LogoDisplayMode
  logo_dark_url: string
  logo_light_url: string
  favicon_url: string
  theme_config: BrandingThemeConfig
  updated_at?: string
}

export interface BrandingPreset {
  id: string
  name: string
  description: string
  dark: ThemePalette
  light: ThemePalette
}

