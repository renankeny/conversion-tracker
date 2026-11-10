export interface Toast {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  title: string
  message: string
  duration?: number
}

export const useToast = () => {
  const toasts = useState<Toast[]>('toasts', () => [])

  const remove = (id: string) => {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  const add = (toast: Omit<Toast, 'id'>) => {
    const id = `toast-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
    const duration = toast.duration ?? 5000
    const newToast: Toast = { ...toast, id, duration }

    toasts.value = [...toasts.value, newToast]

    if (duration > 0 && import.meta.client) {
      setTimeout(() => remove(id), duration)
    }

    return id
  }

  const success = (title: string, message: string, duration?: number) => {
    return add({ type: 'success', title, message, duration })
  }

  const error = (title: string, message: string, duration?: number) => {
    return add({ type: 'error', title, message, duration })
  }

  const warning = (title: string, message: string, duration?: number) => {
    return add({ type: 'warning', title, message, duration })
  }

  const info = (title: string, message: string, duration?: number) => {
    return add({ type: 'info', title, message, duration })
  }

  return {
    toasts,
    add,
    remove,
    success,
    error,
    warning,
    info,
  }
}
