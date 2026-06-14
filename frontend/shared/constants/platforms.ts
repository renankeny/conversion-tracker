// Plataformas (gateways) com webhooks suportados pelo TrackFind.
//
// A URL final é construída em runtime: `${API_URL}${webhookPath}`.
// Por exemplo, com API_URL = "https://api.trackfind.com.br/api/v1" e
// webhookPath = "/webhooks/cakto", a URL final é
// "https://api.trackfind.com.br/api/v1/webhooks/cakto".
//
// Cada gateway só precisa que o cliente cole essa URL nas configurações de
// "compra aprovada" na plataforma — o resto é automático.
//
// IMPORTANTE: só listamos gateways com payload validado em produção real.
// Pra adicionar uma plataforma nova, capture um payload real do gateway
// e ajuste o handler em backend/internal/handlers/webhook_<nome>.go.

export interface PlatformDef {
  /** id estável usado em paths e logs */
  id: string

  /** nome de exibição */
  name: string

  /** descrição curta (1 linha) */
  description: string

  /** path do webhook no backend (concatena com API_URL) */
  webhookPath: string

  /** marca da identidade visual — usada no card */
  brandColor: string

  /** se true, exige assinatura HMAC configurada via env */
  needsHmac?: boolean

  /** nome da env var quando needsHmac=true */
  hmacEnvVar?: string

  /** lista de eventos cobertos pelo webhook (informativo) */
  events: string[]

  /** instruções passo-a-passo curtas pra configuração na plataforma */
  setupSteps: string[]

  /** notas extras (opcional) */
  notes?: string
}

export const PLATFORMS: PlatformDef[] = [
  {
    id: 'hotmart',
    name: 'Hotmart',
    description: 'Plataforma de produtos digitais com afiliados e checkout.',
    webhookPath: '/webhooks/hotmart',
    brandColor: '#EF4444',
    events: ['PURCHASE_APPROVED'],
    setupSteps: [
      'Painel do Produtor → Ferramentas → Webhooks',
      'Clique em "Criar webhook"',
      'Cole a URL ao lado',
      'Selecione "Compra aprovada"',
      'Salve',
    ],
    notes:
      'Suporta v2.0 (sckPaymentLink) e v1 (origin.xcod). PII (nome, email, telefone) é capturado automaticamente.',
  },
  {
    id: 'cakto',
    name: 'Cakto',
    description: 'Gateway de checkout para infoprodutos.',
    webhookPath: '/webhooks/cakto',
    brandColor: '#10B981',
    events: ['purchase_approved (status=paid)'],
    setupSteps: [
      'Painel Cakto → Integrações → Webhooks',
      'Clique em "Adicionar URL"',
      'Cole a URL ao lado',
      'Selecione "Pedido aprovado"',
      'Salve',
    ],
    notes:
      'Usa baseAmount (preço da oferta) priorizando sobre amount (com juros do parcelamento). UTMs lidos de data.utm_source/utm_term/utm_medium/utm_content/utm_campaign/sck.',
  },
  {
    id: 'kirvano',
    name: 'Kirvano',
    description: 'Plataforma de venda de produtos digitais.',
    webhookPath: '/webhooks/kirvano',
    brandColor: '#14B8A6',
    events: ['SALE_APPROVED (status=APPROVED)'],
    setupSteps: [
      'Configurações → Integrações → Webhooks',
      'Clique em "Novo webhook"',
      'Cole a URL ao lado',
      'Selecione o evento "SALE_APPROVED"',
      'Salve',
    ],
    notes:
      'Lê valor de fiscal.total_value (numérico). UTMs em data.utm.{utm_source, utm_term, utm_medium, utm_content, utm_campaign, src}.',
  },
  {
    id: 'rkcursos',
    name: 'RKCursos',
    description: 'Plataforma de cursos online com checkout integrado.',
    webhookPath: '/webhooks/rkcursos',
    brandColor: '#0891B2',
    events: ['checkout.purchased'],
    setupSteps: [
      'Painel RKCursos → Configurações → Webhooks',
      'Cole a URL ao lado',
      'Selecione "checkout.purchased"',
      'Salve',
    ],
    notes:
      'Aceita UTMs (utm_source/term/medium/content/campaign/sck/src) E faz fallback por phone (variantes 55) e email quando não vem userId.',
  },
]
