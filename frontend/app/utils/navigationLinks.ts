import type { Component } from 'vue'
import { MetaAdsIcon, GoogleAdsIcon, WhatsAppIcon, PixelBrainIcon } from './adIcons'

export interface SubNavLink {
  to: string
  label: string
  exact?: boolean
  icon?: Component
}

export const campaignSubLinks: SubNavLink[] = [
  { to: '/campanhas', label: 'Performance', exact: true },
  { to: '/campanhas/regras', label: 'Regras Automáticas' },
]

export const integrationLinks: SubNavLink[] = [
  { to: '/integracoes/meta-ads', label: 'Meta Ads', icon: MetaAdsIcon },
  { to: '/integracoes/google-ads', label: 'Google Ads', icon: GoogleAdsIcon },
  { to: '/integracoes/pixels', label: 'Pixel', icon: PixelBrainIcon },
  { to: '/integracoes/whatsapp', label: 'WhatsApp', icon: WhatsAppIcon },
  { to: '/integracoes/plataformas', label: 'Checkouts (infoprodutos)' },
  { to: '/integracoes/custom', label: 'Checkouts Personalizados' },
  { to: '/integracoes/ecommerce/confirmacao-vendas', label: 'Confirmação de Vendas' },
]

export const settingsSubLinks: SubNavLink[] = [
  { to: '/settings', label: 'Geral', exact: true },
  { to: '/settings/webhooks', label: 'Webhooks' },
  { to: '/settings/api', label: 'API' },
]
