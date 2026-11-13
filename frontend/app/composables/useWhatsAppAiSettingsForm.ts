import type { AILearnedRule } from '#shared/types'
import { useTenantStore } from '~/stores/useTenantStore'
import { formatDateTimeSP } from '~/utils/formatDate'
import { errorMessage } from '~/utils/errorMessage'

export const useWhatsAppAiSettingsForm = () => {
  const api = useAPI()
  const wa = useWhatsApp()
  const { hasOpenAiKey } = useAuth()
  const tenantStore = useTenantStore()
  const toast = useToast()

  const loading = ref(true)
  const saving = ref(false)
  const runningNow = ref(false)
  const defaultPrompt = ref('')
  const serverHasKey = ref(false)
  const lastRunAt = ref<string | null>(null)
  const stages = ref<Array<{ id: number; name: string }>>([])

  const form = reactive({
    enabled: false,
    model: 'gpt-4o-mini',
    qualified_stage_id: 0 as number,
    sale_stage_id: 0 as number,
    base_prompt: '',
    learned_rules: [] as AILearnedRule[],
  })

  const keyReady = computed(() => Boolean(hasOpenAiKey.value || serverHasKey.value))
  const lastRunLabel = computed(() => (lastRunAt.value ? formatDateTimeSP(lastRunAt.value) : ''))

  const qualStageOptions = computed(() => [
    { value: 0, label: 'Detectar etapa de Qualificado automaticamente' },
    ...stages.value.map((s) => ({ value: s.id, label: s.name })),
  ])
  const saleStageOptions = computed(() => [
    { value: 0, label: 'Detectar etapa de Venda automaticamente' },
    ...stages.value.map((s) => ({ value: s.id, label: s.name })),
  ])

  const loadAll = async () => {
    loading.value = true
    try {
      const [cfg, stRes] = await Promise.all([
        wa.getAISettings(),
        api
          .get<{ stages?: Array<{ id: number; name: string }> } | Array<{ id: number; name: string }>>(
            '/admin/ext-api/crm/stages',
          )
          .catch(() => []),
      ])
      stages.value = Array.isArray(stRes) ? stRes : (stRes?.stages ?? [])
      form.enabled = cfg.enabled
      form.model = cfg.model || 'gpt-4o-mini'
      form.qualified_stage_id = cfg.qualified_stage_id || 0
      form.sale_stage_id = cfg.sale_stage_id || 0
      form.base_prompt = cfg.base_prompt || cfg.default_prompt || ''
      form.learned_rules = [...(cfg.learned_rules || [])]
      defaultPrompt.value = cfg.default_prompt || cfg.base_prompt || ''
      serverHasKey.value = Boolean(cfg.has_openai_key)
      lastRunAt.value = cfg.last_run_at || null
    } catch (e: unknown) {
      toast.error('Erro ao carregar IA', errorMessage(e, 'Não foi possível buscar as configurações.'))
    } finally {
      loading.value = false
    }
  }

  watch(() => tenantStore.activeAccountId, () => { void loadAll() }, { immediate: true })

  const restoreDefaultPrompt = () => {
    if (defaultPrompt.value) form.base_prompt = defaultPrompt.value
  }

  const persistSettings = async (silentRuleToast = false) => {
    saving.value = true
    try {
      const saved = await wa.updateAISettings({
        enabled: form.enabled,
        model: form.model,
        qualified_stage_id: form.qualified_stage_id > 0 ? form.qualified_stage_id : null,
        sale_stage_id: form.sale_stage_id > 0 ? form.sale_stage_id : null,
        base_prompt: form.base_prompt,
        learned_rules: form.learned_rules,
      })
      serverHasKey.value = Boolean(saved.has_openai_key)
      toast.success(
        silentRuleToast ? 'Regras atualizadas' : 'Configurações salvas',
        'As preferências da IA foram gravadas.',
      )
    } catch (e: unknown) {
      toast.error('Erro ao salvar', errorMessage(e, 'Falha ao gravar configurações da IA.'))
    } finally {
      saving.value = false
    }
  }

  const removeRule = (idx: number) => {
    form.learned_rules.splice(idx, 1)
    void persistSettings(true)
  }
  const addRule = (r: AILearnedRule) => {
    form.learned_rules.push(r)
    void persistSettings(true)
  }
  const onSave = () => persistSettings(false)

  const onRunNow = async () => {
    runningNow.value = true
    try {
      const res = await wa.runAINow()
      lastRunAt.value = new Date().toISOString()
      toast.success(
        'Análise concluída',
        `${res.analyzed_chats} conversas lidas · ${res.qualifications} qualificações · ${res.sales} vendas.`,
      )
    } catch (e: unknown) {
      toast.error('Não foi possível analisar', errorMessage(e, 'Configure sua chave da OpenAI na aba Meu Perfil.'))
    } finally {
      runningNow.value = false
    }
  }

  return {
    loading, saving, runningNow, form,
    keyReady, lastRunLabel,
    qualStageOptions, saleStageOptions,
    restoreDefaultPrompt, removeRule, addRule, onSave, onRunNow,
  }
}
