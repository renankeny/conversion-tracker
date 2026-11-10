<template>
  <section class="form-section">
    <header class="section-header">
      <div class="header-row">
        <h4>3. Mapeamento de Ações de Conversão</h4>
        <BaseButton
          v-if="canLoadRemoteActions"
          type="button"
          variant="ghost"
          size="sm"
          :loading="loadingActions"
          @click="$emit('refresh-actions')"
        >
          Atualizar ações
        </BaseButton>
      </div>
      <p class="section-hint">
        Escolha qual evento do {{ brandName }} dispara cada Ação de Conversão da sua conta Google Ads.
      </p>
    </header>

    <div v-if="actions.length === 0" class="actions-empty">
      Nenhuma Ação de Conversão mapeada. Clique em "+ Adicionar Conversion Action" abaixo.
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
            title="Remover Conversion Action"
            @click="removeAction(idx)"
          >
            Remover
          </button>
        </div>
      </div>
    </div>

    <button type="button" class="add-action-btn" @click="addAction">
      + Adicionar Conversion Action
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
  'refresh-actions': []
}>()

const { brandName } = useBranding()

const eventTypeOptions: Array<{ value: GoogleAdsEventType; label: string }> = [
  { value: 'Purchase', label: 'Purchase (Vendas)' },
  { value: 'Lead', label: 'Lead' },
  { value: 'CompleteRegistration', label: 'CompleteRegistration (Cadastro)' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout' },
  { value: 'AddToCart', label: 'AddToCart' },
  { value: 'ViewContent', label: 'ViewContent' },
  { value: 'PageView', label: 'PageView' },
]

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

const onPickRemoteAction = (row: GoogleAdsConversionAction, actionId: string) => {
  row.conversion_action_id = actionId
  const found = props.remoteActions.find((r) => r.id === actionId)
  if (found) {
    row.name = found.name
  }
}

const addAction = () => {
  props.actions.push({
    event_type: 'Purchase',
    conversion_action_id: '',
    name: '',
    enabled: true,
  })
}

const removeAction = (idx: number) => {
  props.actions.splice(idx, 1)
}
</script>

<style scoped>
.form-section { display: flex; flex-direction: column; gap: 0.75rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.header-row { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; }
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
.add-action-btn { align-self: flex-start; background: transparent; border: 1px dashed var(--border); color: var(--brand); font-size: var(--fs-md); font-weight: var(--fw-medium); padding: 0.625rem 1rem; border-radius: 8px; cursor: pointer; }
</style>
