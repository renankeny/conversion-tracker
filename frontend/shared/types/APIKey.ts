export type APIKeyModuleScope =
  | 'events'
  | 'crm'
  | 'users'
  | 'whatsapp'
  | 'pixels'
  | 'analytics'
  | 'sales'
  | 'webhooks'

export interface APIKeyScopeOption {
  value: APIKeyModuleScope
  label: string
  hint: string
}

export const API_KEY_SCOPE_OPTIONS: APIKeyScopeOption[] = [
  { value: 'events', label: 'Eventos e Reenvio', hint: 'Disparar conversões, listar auditoria e reprocessar falhas' },
  { value: 'crm', label: 'CRM e Funil 360', hint: 'Etapas, quadro Kanban, leads, movimentação, notas e tags' },
  { value: 'users', label: 'Usuários e Jornada', hint: 'Buscar contatos, jornada multi-sessão e enriquecer dados' },
  { value: 'whatsapp', label: 'WhatsApp e IA', hint: 'Instâncias, leitura de conversas, rodar IA e converter chats' },
  { value: 'pixels', label: 'Pixels e Links na Bio', hint: 'Gerenciar pixels e criar links rastreáveis com rodízio' },
  { value: 'analytics', label: 'Relatórios e ROAS', hint: 'Dashboard, mapa geográfico, Meta Ads e Google Ads' },
  { value: 'sales', label: 'Confirmação de Vendas', hint: 'Listar e aprovar pedidos pendentes de PIX ou boleto' },
  { value: 'webhooks', label: 'Checkouts e URLs', hint: 'Histórico de checkouts, replay e mapeamento de páginas' },
]

export interface APIKey {
  id: number
  account_id: number
  name: string
  key_prefix: string
  plaintext?: string
  scopes: string[]
  read_only: boolean
  last_used_at?: string | null
  revoked_at?: string | null
  created_at: string
  updated_at: string
}

export interface APIKeysListResponse {
  data: APIKey[]
}

export interface CreateAPIKeyInput {
  name: string
  account_id?: number
  scopes: string[]
  read_only: boolean
}

export interface UpdateAPIKeyInput {
  name: string
  scopes: string[]
  read_only: boolean
}

export interface CreateAPIKeyResponse {
  key: APIKey
  plaintext: string
}

export interface UpdateAPIKeyResponse {
  key: APIKey
}
