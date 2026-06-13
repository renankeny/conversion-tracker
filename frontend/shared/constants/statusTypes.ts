export const STATUS_TYPES = {
  SENT: 'sent',
  PENDING: 'pending',
  FAILED: 'failed',
} as const

export type StatusType = typeof STATUS_TYPES[keyof typeof STATUS_TYPES]

export const STATUS_LABELS: Record<string, string> = {
  sent: 'Enviado',
  pending: 'Pendente',
  failed: 'Falhou',
}

export const STATUS_COLORS: Record<string, string> = {
  sent: 'success',
  pending: 'warning',
  failed: 'danger',
}
