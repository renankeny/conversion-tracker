import type { Component } from 'vue'
import {
  Eye, Layers, UserPlus, Phone, ShoppingCart, Heart,
  Lock, CreditCard, CheckCircle, Receipt, AlertTriangle, DollarSign,
} from 'lucide-vue-next'

// Catálogo de eventos web mapeados pela plataforma (exclui internos e
// business messaging/WhatsApp — essa dashboard é só web/pixel). Compartilhado
// entre a dashboard autenticada e a dashboard pública view-only, já que
// ambas renderizam os mesmos cards/funil a partir da mesma config por pixel.
export const FAILED_CARD_ID = '__failed__'
export const FAILED_CARD_ICON = AlertTriangle
export const FAILED_CARD_LABEL = 'Eventos com Falha'

// Card de valor vendido — não é um "evento" (é a soma de Purchase.value no
// período), mas entra no mesmo catálogo pra poder ser renomeado/reordenado
// junto dos demais cards.
export const REVENUE_CARD_ID = '__revenue__'
export const REVENUE_CARD_ICON = DollarSign
export const REVENUE_CARD_LABEL = 'Valor Vendido no Período'

export interface EventCatalogEntry {
  id: string
  label: string
  icon: Component
  variant: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  color: string
}

export const EVENT_CATALOG: EventCatalogEntry[] = [
  { id: 'PageView', label: 'Visitas (PageView)', icon: Eye, variant: 'primary', color: 'var(--brand)' },
  { id: 'ViewContent', label: 'View Content', icon: Layers, variant: 'primary', color: 'var(--brand-strong)' },
  { id: 'Lead', label: 'Leads', icon: UserPlus, variant: 'primary', color: 'var(--brand)' },
  { id: 'Contact', label: 'Contato', icon: Phone, variant: 'primary', color: 'var(--brand)' },
  { id: 'AddToCart', label: 'Add to Cart', icon: ShoppingCart, variant: 'primary', color: 'var(--brand)' },
  { id: 'AddToWishlist', label: 'Lista de Desejos', icon: Heart, variant: 'primary', color: 'var(--brand)' },
  { id: 'InitiateCheckout', label: 'Initiate Checkout', icon: Lock, variant: 'primary', color: 'var(--brand)' },
  { id: 'AddPaymentInfo', label: 'Add Payment Info', icon: CreditCard, variant: 'primary', color: 'var(--brand)' },
  { id: 'CompleteRegistration', label: 'Cadastro Completo', icon: CheckCircle, variant: 'primary', color: 'var(--brand)' },
  { id: 'Purchase', label: 'Vendas (Purchase)', icon: Receipt, variant: 'primary', color: 'var(--brand)' },
]

export const CATALOG_IDS = EVENT_CATALOG.map((e) => e.id)
export const ALL_CARD_IDS = [...CATALOG_IDS, FAILED_CARD_ID, REVENUE_CARD_ID]
export const DEFAULT_FUNNEL_EVENTS = ['PageView', 'ViewContent', 'AddToCart', 'InitiateCheckout', 'Purchase']

export const catalogById = new Map(EVENT_CATALOG.map((e) => [e.id, e]))
