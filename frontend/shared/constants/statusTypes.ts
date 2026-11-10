export const STATUS_TYPES = {
  SENT: 'sent',
  QUEUED: 'queued',
  PENDING: 'queued',
  FAILED: 'failed',
} as const

export type StatusType = typeof STATUS_TYPES[keyof typeof STATUS_TYPES]

export const STATUS_LABELS: Record<string, string> = {
  sent: 'Enviado',
  queued: 'Na fila',
  pending: 'Pendente',
  failed: 'Falhou',
}

export const STATUS_COLORS: Record<string, string> = {
  sent: 'success',
  queued: 'warning',
  pending: 'warning',
  failed: 'danger',
}
