// APIKey é um token server-to-server vinculado a uma conta.
// O plaintext (`trk_<base64>`) só aparece uma vez, no retorno da criação.
export interface APIKey {
  id: number
  account_id: number
  name: string
  key_prefix: string
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
}

// Resposta especial da criação: contém o plaintext que precisa ser copiado
// imediatamente; nunca aparece de novo.
export interface CreateAPIKeyResponse {
  key: APIKey
  plaintext: string
}
