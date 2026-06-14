// Plataformas (gateways) com webhooks suportados pelo TrackFind.
//
// A URL final é construída em runtime: `${API_URL}${webhookPath}`. Por exemplo,
// se API_URL = "https://api.trackfind.com.br/api/v1" e webhookPath =
// "/webhooks/kiwify", a URL final é
// "https://api.trackfind.com.br/api/v1/webhooks/kiwify".
//
// Cada gateway só precisa que o cliente cole essa URL nas configurações de
// "compra aprovada" na plataforma — o resto é automático.

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
    id: 'kiwify',
    name: 'Kiwify',
    description: 'Gateway de infoprodutos brasileiro com checkout próprio.',
    webhookPath: '/webhooks/kiwify',
    brandColor: '#9F7AEA',
    events: ['Compra aprovada (purchase)'],
    setupSteps: [
      'Acesse o painel Kiwify → Apps → Webhooks',
      'Clique em "Criar webhook"',
      'Cole a URL ao lado',
      'Marque o evento "Compra aprovada"',
      'Salve',
    ],
    notes: 'A Kiwify envia o valor em centavos; o TrackFind converte automaticamente.',
  },
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
    notes: 'O Hotmart manda PII junto (nome, email, telefone) — capturado automaticamente.',
  },
  {
    id: 'eduzz',
    name: 'Eduzz',
    description: 'Plataforma de venda de produtos digitais e físicos.',
    webhookPath: '/webhooks/eduzz',
    brandColor: '#F59E0B',
    events: ['Fatura paga (myeduzz.invoice_paid)'],
    setupSteps: [
      'Painel Eduzz → Configurações → API/Webhooks',
      'Crie um novo webhook',
      'Cole a URL ao lado',
      'Selecione o evento "myeduzz.invoice_paid"',
      'Salve',
    ],
  },
  {
    id: 'yampi',
    name: 'Yampi',
    description: 'Plataforma de e-commerce com checkout integrado.',
    webhookPath: '/webhooks/yampi',
    brandColor: '#06B6D4',
    needsHmac: true,
    hmacEnvVar: 'YAMPI_WEBHOOK_SECRET',
    events: ['order.paid (Purchase)', 'order.created (AddPaymentInfo)'],
    setupSteps: [
      'Configurações da loja Yampi → Notificações → Webhooks',
      'Clique em "Adicionar webhook"',
      'Cole a URL ao lado',
      'Marque os eventos "order.paid" e "order.created"',
      'Copie o secret gerado e configure a env YAMPI_WEBHOOK_SECRET no servidor',
      'Salve',
    ],
    notes:
      'A Yampi assina cada request com HMAC-SHA256. Configure YAMPI_WEBHOOK_SECRET nas envs para o TrackFind validar a assinatura. Sem isso, o webhook é rejeitado em produção.',
  },
  {
    id: 'cakto',
    name: 'Cakto',
    description: 'Gateway de checkout para infoprodutos.',
    webhookPath: '/webhooks/cakto',
    brandColor: '#10B981',
    events: ['pedido_aprovado'],
    setupSteps: [
      'Painel Cakto → Integrações → Webhooks',
      'Clique em "Adicionar URL"',
      'Cole a URL ao lado',
      'Selecione "Pedido aprovado"',
      'Salve',
    ],
  },
  {
    id: 'braip',
    name: 'Braip',
    description: 'Plataforma de venda online de produtos físicos e digitais.',
    webhookPath: '/webhooks/braip',
    brandColor: '#8B5CF6',
    events: ['Status: paid / approved'],
    setupSteps: [
      'Painel Braip → Configurações → Postback / Webhook',
      'Insira a URL ao lado no campo "URL Postback"',
      'Marque a opção de "Vendas aprovadas"',
      'Salve',
    ],
  },
  {
    id: 'monetizze',
    name: 'Monetizze',
    description: 'Plataforma de venda de produtos digitais com afiliados.',
    webhookPath: '/webhooks/monetizze',
    brandColor: '#3B82F6',
    events: ['status_descricao = Aprovada'],
    setupSteps: [
      'Painel Monetizze → Mais → Postback',
      'Adicionar nova URL',
      'Cole a URL ao lado',
      'Marque "Venda Aprovada"',
      'Salve',
    ],
  },
]
