import { computed, ref } from 'vue'
import type { ComputedRef, Ref } from 'vue'
import type { PlatformDef } from '#shared/constants/platforms'
import type { PlatformSummaryItem } from '#shared/types'
import { resolvePublicApiBase } from '~/utils/apiOrigin'
import { errorMessage } from '~/utils/errorMessage'

interface TokenStatusResponse {
  has_token: boolean
  masked_token: string
  updated_at?: string
}

export const usePlatformTokenModal = (deps: {
  effectiveAccountId: ComputedRef<string>
  accountHeaders: ComputedRef<Record<string, string>>
  summaryItems: Ref<Record<string, PlatformSummaryItem>>
}) => {
  const toast = useToast()
  const config = useRuntimeConfig()
  const api = useAPI()

  const showModal = ref(false)
  const selectedPlatform = ref<PlatformDef | null>(null)
  const selectedPixelSlug = ref('')
  const tokenInput = ref('')
  const hasSavedToken = ref(false)
  const maskedToken = ref('')
  const loadingToken = ref(false)
  const savingToken = ref(false)
  const copiedKey = ref<string | null>(null)

  const buildWebhookUrl = (path: string, pixelSlug = ''): string => {
    const base = resolvePublicApiBase(String(config.public.apiUrl || '')) + path
    const params = new URLSearchParams()
    if (deps.effectiveAccountId.value) params.set('acc', deps.effectiveAccountId.value)
    if (pixelSlug.trim()) params.set('pixel', pixelSlug.trim())
    const qs = params.toString()
    return qs ? `${base}?${qs}` : base
  }

  const modalWebhookUrl = computed(() =>
    selectedPlatform.value ? buildWebhookUrl(selectedPlatform.value.webhookPath, selectedPixelSlug.value) : '',
  )

  const loadSelectedPlatformToken = async (platformId: string) => {
    loadingToken.value = true
    tokenInput.value = ''
    try {
      const acc = deps.effectiveAccountId.value
      const accQuery = acc ? `?account_id=${encodeURIComponent(acc)}` : ''
      const res = await api.get<TokenStatusResponse>(
        `/admin/integrations/platforms/tokens/${platformId}${accQuery}`,
        { headers: deps.accountHeaders.value },
      )
      hasSavedToken.value = Boolean(res.has_token)
      maskedToken.value = res.masked_token || ''
    } catch {
      hasSavedToken.value = false
      maskedToken.value = ''
    } finally {
      loadingToken.value = false
    }
  }

  const openModal = async (p: PlatformDef) => {
    selectedPlatform.value = p
    selectedPixelSlug.value = ''
    showModal.value = true
    const cached = deps.summaryItems.value[p.id]
    hasSavedToken.value = Boolean(cached?.has_token)
    maskedToken.value = cached?.masked_token || ''
    await loadSelectedPlatformToken(p.id)
  }

  const persistToken = async (secret: string) => {
    if (!selectedPlatform.value) return
    const acc = deps.effectiveAccountId.value
    if (!acc) {
      toast.error('Empresa obrigatória', 'Selecione uma empresa antes de salvar o token.')
      return
    }
    savingToken.value = true
    try {
      const res = await api.put<TokenStatusResponse>(
        `/admin/integrations/platforms/tokens/${selectedPlatform.value.id}`,
        { token_secret: secret, account_id: Number(acc) },
        { headers: deps.accountHeaders.value },
      )
      hasSavedToken.value = Boolean(res.has_token)
      maskedToken.value = res.masked_token || ''
      tokenInput.value = ''
      const prev = deps.summaryItems.value[selectedPlatform.value.id] || {
        has_token: false,
        total_accepted: 0,
        total_failed: 0,
      }
      deps.summaryItems.value = {
        ...deps.summaryItems.value,
        [selectedPlatform.value.id]: {
          ...prev,
          has_token: Boolean(res.has_token),
          masked_token: res.masked_token || '',
        },
      }
      toast.success(
        res.has_token ? 'Token salvo' : 'Token removido',
        res.has_token
          ? `Validação de segurança ativa para ${selectedPlatform.value.name}.`
          : `Validação de token desativada na ${selectedPlatform.value.name}.`,
      )
    } catch (e: unknown) {
      toast.error('Erro ao salvar', errorMessage(e, 'Não foi possível atualizar o token.'))
    } finally {
      savingToken.value = false
    }
  }

  const saveToken = () => persistToken(tokenInput.value.trim())
  const clearToken = () => persistToken('')

  const copyText = async (url: string, key: string, platformName: string) => {
    try {
      await navigator.clipboard.writeText(url)
      copiedKey.value = key
      toast.success('URL copiada', `Cole no painel da ${platformName}.`)
      setTimeout(() => {
        if (copiedKey.value === key) copiedKey.value = null
      }, 2000)
    } catch {
      toast.error('Erro ao copiar', 'Não foi possível copiar automaticamente. Selecione a URL manualmente.')
    }
  }

  const copyCardUrl = (p: PlatformDef) => copyText(buildWebhookUrl(p.webhookPath), `card:${p.id}`, p.name)
  const copyModalUrl = () => {
    if (!selectedPlatform.value) return
    copyText(modalWebhookUrl.value, `modal:${selectedPlatform.value.id}`, selectedPlatform.value.name)
  }

  return {
    showModal,
    selectedPlatform,
    selectedPixelSlug,
    modalWebhookUrl,
    tokenInput,
    hasSavedToken,
    maskedToken,
    loadingToken,
    savingToken,
    copiedKey,
    loadSelectedPlatformToken,
    openModal,
    saveToken,
    clearToken,
    copyCardUrl,
    copyModalUrl,
  }
}
