import { useAuthStore } from '~/stores/useAuthStore'
import { errorMessage } from '~/utils/errorMessage'

interface EasyPanelStatusResponse {
  linked: boolean
  hint_url?: string
  version?: string
}

interface EasyPanelConnectResponse {
  ok: boolean
  linked: boolean
  project?: string
}

export const useEasyPanelGate = () => {
  const api = useAPI()
  const authStore = useAuthStore()

  const checked = useState<boolean>('ep_gate_checked', () => false)
  const linked = useState<boolean>('ep_gate_linked', () => true)
  const hintUrl = useState<string>('ep_gate_hint_url', () => '')
  const version = useState<string>('ep_gate_version', () => '')
  const checking = useState<boolean>('ep_gate_checking', () => false)
  const submitting = useState<boolean>('ep_gate_submitting', () => false)
  const errorMsg = useState<string>('ep_gate_error', () => '')

  const checkEasyPanelStatus = async (force = false): Promise<void> => {
    if (!authStore.isAuthenticated) {
      checked.value = false
      linked.value = true
      return
    }
    if (checked.value && !force) return
    if (checking.value) return

    checking.value = true
    try {
      const res = await api.get<EasyPanelStatusResponse>('/admin/system/easypanel')
      linked.value = !!res?.linked
      hintUrl.value = res?.hint_url || ''
      version.value = res?.version || ''
      checked.value = true
    } catch {
      // Em caso de falha transitória de rede, não bloqueia indevidamente até o próximo check
      linked.value = true
      checked.value = false
    } finally {
      checking.value = false
    }
  }

  const connectEasyPanel = async (
    panelUrl: string,
    apiToken: string,
    skipEasyPanel = false,
  ): Promise<boolean> => {
    errorMsg.value = ''
    const cleanUrl = panelUrl.trim()
    const cleanTok = apiToken.trim()
    if (!skipEasyPanel && (!cleanUrl || !cleanTok)) {
      errorMsg.value = 'Informe a URL do seu EasyPanel e a API Key.'
      return false
    }

    submitting.value = true
    try {
      const res = await api.post<EasyPanelConnectResponse>('/admin/system/easypanel', {
        panel_url: cleanUrl,
        api_token: cleanTok,
        skip_easypanel: skipEasyPanel,
      })
      if (res?.linked) {
        linked.value = true
        checked.value = true
        return true
      }
      errorMsg.value = 'Não foi possível confirmar a configuração.'
      return false
    } catch (err: unknown) {
      errorMsg.value = errorMessage(err, 'Falha ao validar URL ou API Key no EasyPanel.')
      return false
    } finally {
      submitting.value = false
    }
  }

  const resetGate = (): void => {
    checked.value = false
    linked.value = true
    errorMsg.value = ''
  }

  return {
    checked,
    linked,
    hintUrl,
    version,
    checking,
    submitting,
    errorMsg,
    checkEasyPanelStatus,
    connectEasyPanel,
    resetGate,
  }
}
