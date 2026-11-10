export interface PlatformDef {
  /** id estável usado em paths, tokens e logs */
  id: string

  /** nome oficial de exibição */
  name: string

  /** descrição executiva curta (1 linha) */
  description: string

  /** path do webhook no backend (concatena com API_URL) */
  webhookPath: string

  /** cor institucional da marca (fallback visual) */
  brandColor: string

  /** caminho da logo oficial em /public/img/platforms */
  logoUrl: string

  /** lista de eventos processados pelo webhook */
  events: string[]

  /** passo a passo objetivo de configuração no painel da plataforma */
  setupSteps: string[]

  /** observação técnica de validação/rastreamento */
  notes?: string
}

