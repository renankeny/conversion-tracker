import type { BrandingPreset, PlatformBranding, ThemePalette } from '#shared/types'

export const BRANDING_PRESETS: BrandingPreset[] = [
  {
    id: 'emerald',
    name: 'Emerald Neon',
    description: 'Verde de alta conversão sobre grafite escuro',
    dark: { brand: '#A2F07C', surface_1: '#29292E', surface_2: '#323238', surface_deep: '#202024' },
    light: { brand: '#0B4B4D', surface_1: '#FFFFFF', surface_2: '#FFFFFF', surface_deep: '#F1F5F9' },
  },
  {
    id: 'ocean',
    name: 'Ocean Blue',
    description: 'Azul executivo clássico e confiável',
    dark: { brand: '#1C87C1', surface_1: '#18181B', surface_2: '#27272A', surface_deep: '#09090B' },
    light: { brand: '#0284C7', surface_1: '#F8FAFC', surface_2: '#FFFFFF', surface_deep: '#F1F5F9' },
  },
  {
    id: 'violet',
    name: 'Royal Violet',
    description: 'Roxo moderno inspirado em SaaS premium',
    dark: { brand: '#8B5CF6', surface_1: '#13111C', surface_2: '#1E1B2E', surface_deep: '#0C0A12' },
    light: { brand: '#6D28D9', surface_1: '#FAFAFF', surface_2: '#FFFFFF', surface_deep: '#F3F0FF' },
  },
  {
    id: 'amber',
    name: 'Amber Gold',
    description: 'Dourado energético para infoprodutos e lançamentos',
    dark: { brand: '#F59E0B', surface_1: '#18181B', surface_2: '#232329', surface_deep: '#101013' },
    light: { brand: '#D97706', surface_1: '#FFFDF9', surface_2: '#FFFFFF', surface_deep: '#FEF3C7' },
  },
  {
    id: 'crimson',
    name: 'Crimson Rose',
    description: 'Rosa avermelhado vibrante e marcante',
    dark: { brand: '#F43F5E', surface_1: '#18181B', surface_2: '#242429', surface_deep: '#111114' },
    light: { brand: '#E11D48', surface_1: '#FFF9FA', surface_2: '#FFFFFF', surface_deep: '#FFE4E6' },
  },
  {
    id: 'cyan',
    name: 'Cyber Cyan',
    description: 'Ciano tecnológico sobre azul marinho profundo',
    dark: { brand: '#06B6D4', surface_1: '#0F172A', surface_2: '#1E293B', surface_deep: '#090E1A' },
    light: { brand: '#0891B2', surface_1: '#F8FAFC', surface_2: '#FFFFFF', surface_deep: '#ECFEFF' },
  },
]

export const DEFAULT_BRANDING: PlatformBranding = {
  platform_name: 'Sua Plataforma',
  tagline: 'Intelligence & Attribution',
  footer_text: '',
  logo_display_mode: 'icon_text',
  logo_dark_url: '',
  logo_light_url: '',
  favicon_url: '',
  theme_config: {
    preset: 'emerald',
    dark: { ...BRANDING_PRESETS[0]!.dark },
    light: { ...BRANDING_PRESETS[0]!.light },
  },
}

export function normalizeHexColor(raw: string, fallback = '#1C87C1'): string {
  let clean = (raw || '').trim().toUpperCase()
  if (!clean) return fallback
  if (!clean.startsWith('#')) clean = `#${clean}`
  if (/^#[0-9A-F]{3}$/.test(clean)) {
    clean = `#${clean[1]}${clean[1]}${clean[2]}${clean[2]}${clean[3]}${clean[3]}`
  }
  return /^#[0-9A-F]{6}$/.test(clean) ? clean : fallback
}

export function isValidHexColor(raw: string): boolean {
  let clean = (raw || '').trim().toUpperCase()
  if (!clean) return false
  if (!clean.startsWith('#')) clean = `#${clean}`
  if (/^#[0-9A-F]{3}$/.test(clean)) {
    clean = `#${clean[1]}${clean[1]}${clean[2]}${clean[2]}${clean[3]}${clean[3]}`
  }
  return /^#[0-9A-F]{6}$/.test(clean)
}

export function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  const clean = (hex || '').replace('#', '').trim()
  if (!/^[0-9A-Fa-f]{6}$/.test(clean)) return null
  return {
    r: parseInt(clean.slice(0, 2), 16),
    g: parseInt(clean.slice(2, 4), 16),
    b: parseInt(clean.slice(4, 6), 16),
  }
}

export function adjustBrightness(hex: string, amount: number): string {
  const rgb = hexToRgb(hex)
  if (!rgb) return hex
  const clamp = (v: number) => Math.max(0, Math.min(255, Math.round(v)))
  const r = clamp(rgb.r + amount)
  const g = clamp(rgb.g + amount)
  const b = clamp(rgb.b + amount)
  return `#${[r, g, b].map((x) => x.toString(16).padStart(2, '0')).join('')}`
}

export function getContrastTextColor(hex: string): string {
  const rgb = hexToRgb(hex)
  if (!rgb) return '#0F172A'
  // Luminância perceptiva (YIQ)
  const yiq = (rgb.r * 299 + rgb.g * 587 + rgb.b * 114) / 1000
  return yiq >= 150 ? '#0F172A' : '#FFFFFF'
}

export function computePaletteVars(palette: ThemePalette): Record<string, string> {
  const rgb = hexToRgb(palette.brand) || { r: 162, g: 240, b: 124 }
  const brandStrong = adjustBrightness(palette.brand, -18)
  const brandLight = adjustBrightness(palette.brand, 25)
  const brandText = getContrastTextColor(palette.brand)

  return {
    '--brand': palette.brand,
    '--brand-strong': brandStrong,
    '--brand-light': brandLight,
    '--brand-soft': `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.15)`,
    '--brand-soft-2': `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.10)`,
    '--brand-border': `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.32)`,
    '--brand-text': brandText,
    '--info': palette.brand,
    '--info-soft': `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.15)`,
    '--success': palette.brand,
    '--success-soft': `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.15)`,
    '--surface-1': palette.surface_1,
    '--surface-2': palette.surface_2,
    '--surface-3': palette.surface_2,
    '--surface-deep': palette.surface_deep,
  }
}

export function applyPaletteToDocument(palette: ThemePalette): void {
  if (!import.meta.client) return
  const root = document.documentElement
  const vars = computePaletteVars(palette)
  for (const [k, v] of Object.entries(vars)) {
    root.style.setProperty(k, v)
  }
}

