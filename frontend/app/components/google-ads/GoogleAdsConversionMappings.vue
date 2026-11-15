<template>
  <section class="form-section">
    <header class="section-header">
      <div class="header-row">
        <div>
          <h4>3. Mapeamento de Ações de Conversão</h4>
          <p class="section-hint">
            Vincule os eventos do {{ brandName }} às Ações de Conversão correspondentes no Google Ads.
          </p>
        </div>
        <div v-if="canLoadRemoteActions" class="header-actions">
          <BaseButton
            v-if="remoteActions.length"
            type="button"
            variant="ghost"
            size="sm"
            @click="autoMapRecommended"
          >
            Auto-mapear recomendadas
          </BaseButton>
          <BaseButton
            type="button"
            variant="ghost"
            size="sm"
            :loading="loadingActions"
            @click="$emit('refresh-actions', true)"
          >
            Atualizar ações
          </BaseButton>
        </div>
      </div>
    </header>

    <div v-if="actions.length === 0" class="actions-empty">
      Nenhuma Ação de Conversão mapeada. Clique em "+ Adicionar Ação de Conversão" abaixo.
    </div>

    <div v-else class="actions-list">
      <div
        v-for="(ca, idx) in actions"
        :key="idx"
        class="action-row"
        :class="{ disabled: !ca.enabled }"
      >
        <div class="action-fields">
          <BaseDropdown
            v-model="ca.event_type"
            :label="`Evento ${brandName}`"
            :options="eventTypeOptions"
          />

          <BaseDropdown
            :model-value="ca.conversion_action_id"
            label="Ação de Conversão (Google Ads)"
            :options="actionOptionsForRow(ca)"
            :placeholder="actionPlaceholder"
            :disabled="!canLoadRemoteActions || loadingActions"
            @update:model-value="(val) => onPickRemoteAction(ca, String(val))"
          />
        </div>

        <div class="action-controls">
          <div class="control-left">
            <BaseCheckbox
              v-model="ca.enabled"
              size="sm"
              :label="ca.enabled ? 'Ativa' : 'Pausada'"
            />
            <span v-if="ca.conversion_action_id" class="action-id-pill">
              {{ ca.name ? `${ca.name} · ` : '' }}ID {{ ca.conversion_action_id }}
            </span>
          </div>
          <button
            type="button"
            class="remove-btn"
            title="Remover Ação de Conversão"
            @click="removeAction(idx)"
          >
            Remover
          </button>
        </div>
      </div>
    </div>

    <div v-if="hasExactDuplicates" class="dup-warn">
      Existem linhas duplicadas com o mesmo evento e a mesma Ação de Conversão. Remova uma delas para evitar envios repetidos.
    </div>

    <button type="button" class="add-action-btn" @click="addAction">
      + Adicionar Ação de Conversão
    </button>
  </section>
</template>

<script setup lang="ts">
import type {
  GoogleAdsConversionAction,
  GoogleAdsEventType,
  GoogleAdsRemoteConversionAction,
} from '#shared/types'

const props = defineProps<{
  actions: GoogleAdsConversionAction[]
  remoteActions: GoogleAdsRemoteConversionAction[]
  loadingActions?: boolean
  canLoadRemoteActions?: boolean
}>()

defineEmits<{
  'refresh-actions': [forceRefresh: boolean]
}>()

const { brandName } = useBranding()
const toast = useToast()

const eventTypeOptions: Array<{ value: GoogleAdsEventType; label: string }> = [
  { value: 'Purchase', label: 'Purchase (Vendas)' },
  { value: 'Lead', label: 'Lead' },
  { value: 'CompleteRegistration', label: 'CompleteRegistration (Cadastro)' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout' },
  { value: 'AddToCart', label: 'AddToCart' },
  { value: 'ViewContent', label: 'ViewContent' },
  { value: 'PageView', label: 'PageView' },
]

const categoryToEvent: Record<string, GoogleAdsEventType> = {
  PURCHASE: 'Purchase',
  SUBMIT_LEAD_FORM: 'Lead',
  CONTACT: 'Lead',
  PHONE_CALL_LEAD: 'Lead',
  IMPORTED_LEAD: 'Lead',
  QUALIFIED_LEAD: 'Lead',
  CONVERTED_LEAD: 'Lead',
  SIGNUP: 'CompleteRegistration',
  SUBSCRIBE_PAID: 'CompleteRegistration',
  BEGIN_CHECKOUT: 'InitiateCheckout',
  ADD_TO_CART: 'AddToCart',
  PAGE_VIEW: 'PageView',
}

const actionPlaceholder = computed(() => {
  if (!props.canLoadRemoteActions) return 'Selecione a conta Google Ads no passo 2 primeiro...'
  if (props.loadingActions) return 'Carregando ações de conversão...'
  if (props.remoteActions.length === 0) return 'Nenhuma ação encontrada nesta conta'
  return 'Escolha a ação de conversão...'
})

const remoteActionOptions = computed(() =>
  props.remoteActions.map((ra) => ({
    value: ra.id,
    label: `${ra.name} · ID ${ra.id} (${ra.category || ra.type})`,
  })),
)

const actionOptionsForRow = (row: GoogleAdsConversionAction) => {
  const list = [...remoteActionOptions.value]
  if (row.conversion_action_id && !list.some((o) => o.value === row.conversion_action_id)) {
    list.unshift({
      value: row.conversion_action_id,
      label: row.name ? `${row.name} · ID ${row.conversion_action_id}` : `ID ${row.conversion_action_id}`,
    })
  }
  return list
}

const hasExactDuplicates = computed(() => {
  const seen = new Set<string>()
  for (const ca of props.actions) {
    if (!ca.conversion_action_id) continue
    const key = `${ca.event_type}:${ca.conversion_action_id}`
    if (seen.has(key)) return true
    seen.add(key)
  }
  return false
})

const onPickRemoteAction = (row: GoogleAdsConversionAction, actionId: string) => {
  row.conversion_action_id = actionId
  const found = props.remoteActions.find((r) => r.id === actionId)
  if (found) row.name = found.name
}

const autoMapRecommended = () => {
  const existingIds = new Set(props.actions.map((a) => a.conversion_action_id).filter(Boolean))
  const added: GoogleAdsConversionAction[] = []
  for (const ra of props.remoteActions) {
    if (ra.status && ra.status !== 'ENABLED') continue
    if (existingIds.has(ra.id)) continue
    const ev = categoryToEvent[(ra.category || '').toUpperCase()]
    if (!ev) continue
    added.push({ event_type: ev, conversion_action_id: ra.id, name: ra.name, enabled: true })
    existingIds.add(ra.id)
  }
  if (!added.length) {
    toast.info('Nada novo para mapear', 'Todas as ações compatíveis já estão mapeadas.')
    return
  }
  if (props.actions.length === 1 && !props.actions[0]!.conversion_action_id) {
    props.actions.splice(0, 1, ...added)
  } else {
    props.actions.push(...added)
  }
  toast.success('Mapeamento aplicado', `${added.length} ${added.length === 1 ? 'ação mapeada' : 'ações mapeadas'} automaticamente.`)
}

const addAction = () => {
  const used = new Set(props.actions.map((a) => a.event_type))
  const nextEvent = eventTypeOptions.find((o) => !used.has(o.value))?.value || 'Purchase'
  props.actions.push({ event_type: nextEvent, conversion_action_id: '', name: '', enabled: true })
}

const removeAction = (idx: number) => {
  props.actions.splice(idx, 1)
}
</script>

<style scoped>
.form-section { display: flex; flex-direction: column; gap: 0.75rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.header-row { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.header-actions { display: flex; align-items: center; gap: 0.45rem; flex-wrap: wrap; }
.section-header h4 { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); margin: 0.2rem 0 0; line-height: 1.5; }
.actions-empty { font-size: var(--fs-base); color: var(--text-2); background: var(--surface-soft); border: 1px dashed var(--border); border-radius: 8px; padding: 0.875rem; }
.actions-list { display: flex; flex-direction: column; gap: 0.75rem; }
.action-row { display: flex; flex-direction: column; gap: 0.75rem; padding: 0.875rem; background: var(--surface-soft); border: 1px solid var(--border); border-radius: 8px; }
.action-row.disabled { opacity: 0.55; }
.action-fields { display: grid; grid-template-columns: 1fr; gap: 0.75rem; }
@media (min-width: 640px) { .action-fields { grid-template-columns: 1fr 1.4fr; } }
.action-controls { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.control-left { display: flex; align-items: center; gap: 0.625rem; flex-wrap: wrap; }
.action-id-pill { font-size: var(--fs-xs); font-family: var(--font-mono); color: var(--text-2); background: var(--surface-2); border: 1px solid var(--border-faint); padding: 0.15rem 0.5rem; border-radius: 999px; }
.remove-btn { background: transparent; border: 1px solid var(--border); color: var(--danger); font-size: var(--fs-sm); font-weight: var(--fw-medium); padding: 0.4rem 0.75rem; border-radius: 6px; cursor: pointer; }
.dup-warn { font-size: var(--fs-sm); color: var(--warning); background: var(--warning-soft); border: 1px solid var(--warning-border); padding: 0.55rem 0.75rem; border-radius: 8px; }
.add-action-btn { align-self: flex-start; background: transparent; border: 1px dashed var(--border); color: var(--brand); font-size: var(--fs-md); font-weight: var(--fw-medium); padding: 0.625rem 1rem; border-radius: 8px; cursor: pointer; }
</style>
