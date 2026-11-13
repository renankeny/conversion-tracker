<template>
  <div class="rules-card">
    <div class="rules-head">
      <div>
        <h3 class="card-title">Regras aprendidas por contestação</h3>
        <p class="card-hint">
          Sempre que você clicar em <strong>Contestar</strong> em uma qualificação ou venda da IA lá no chat do WhatsApp, o motivo entra automaticamente aqui e passa a guiar todas as próximas análises.
        </p>
      </div>
      <span class="count-pill">{{ rules.length }} {{ rules.length === 1 ? 'regra' : 'regras' }}</span>
    </div>

    <div v-if="!rules.length" class="empty-rules">
      Nenhuma contestação registrada até o momento. Você também pode adicionar uma regra preventiva abaixo.
    </div>

    <ul v-else class="rules-list">
      <li v-for="(item, idx) in rules" :key="item.id || idx" class="rule-row">
        <span class="rule-badge" :class="item.event_type === 'purchase' ? 'is-sale' : 'is-qual'">
          {{ item.event_type === 'purchase' ? 'Venda' : 'Qualificação' }}
        </span>
        <p class="rule-text">{{ item.rule }}</p>
        <button
          type="button"
          class="remove-btn"
          title="Remover regra"
          @click="$emit('remove', idx)"
        >
          <Trash2 :size="14" />
        </button>
      </li>
    </ul>

    <div class="add-rule-row">
      <div class="type-col">
        <BaseDropdown
          :model-value="newType"
          :options="typeOptions"
          size="sm"
          @update:model-value="newType = $event === 'qualified' ? 'qualified' : 'purchase'"
        />
      </div>
      <div class="input-col">
        <BaseInput
          v-model="newText"
          placeholder="Adicionar regra manual (ex: Boleto gerado sem comprovante pago não é venda)"
          @keydown.enter.prevent="submitManual"
        />
      </div>
      <BaseButton variant="outline" size="sm" :disabled="!newText.trim()" @click="submitManual">
        <Plus :size="14" />
        <span>Adicionar</span>
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Trash2 } from 'lucide-vue-next'
import type { AILearnedRule } from '#shared/types'

defineProps<{
  rules: AILearnedRule[]
}>()

const emit = defineEmits<{
  remove: [index: number]
  add: [rule: AILearnedRule]
}>()

const newType = ref<'purchase' | 'qualified'>('purchase')
const newText = ref('')

const typeOptions = [
  { value: 'purchase', label: 'Regra p/ Venda' },
  { value: 'qualified', label: 'Regra p/ Qualificação' },
]

const submitManual = () => {
  const txt = newText.value.trim()
  if (!txt) return
  emit('add', {
    id: `manual_${Date.now()}`,
    event_type: newType.value,
    rule: txt,
    created_at: new Date().toISOString(),
  })
  newText.value = ''
}
</script>

<style scoped>
.rules-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem; }
.rules-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; }
.card-title { font-family: var(--font-sans); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0 0 0.35rem; }
.card-hint { font-size: var(--fs-md); color: var(--text-2); margin: 0; line-height: 1.5; max-width: 680px; }
.count-pill { padding: 0.25rem 0.65rem; border-radius: 999px; background: var(--surface-deep); border: 1px solid var(--border-soft); color: var(--text-2); font-size: var(--fs-xs); font-weight: var(--fw-semibold); white-space: nowrap; }
.empty-rules { padding: 1rem; border-radius: 8px; background: var(--surface-deep); border: 1px dashed var(--border); color: var(--text-3); font-size: var(--fs-sm); text-align: center; }
.rules-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 0.5rem; }
.rule-row { display: flex; align-items: center; gap: 0.75rem; padding: 0.65rem 0.85rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 8px; }
.rule-badge { padding: 0.18rem 0.55rem; border-radius: 999px; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.04em; flex-shrink: 0; }
.rule-badge.is-sale { background: var(--surface-soft); color: var(--success); border: 1px solid var(--border-soft); }
.rule-badge.is-qual { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.rule-text { flex: 1; margin: 0; font-size: var(--fs-base); color: var(--text-1); line-height: 1.45; }
.remove-btn { background: transparent; border: none; color: var(--text-3); cursor: pointer; padding: 0.25rem; border-radius: 6px; display: inline-flex; transition: color 0.15s ease, background 0.15s ease; }
.remove-btn:hover { color: var(--danger); background: var(--surface-soft); }
.add-rule-row { display: flex; align-items: center; gap: 0.65rem; padding-top: 0.25rem; }
.type-col { width: 175px; flex-shrink: 0; }
.input-col { flex: 1; min-width: 0; }
@media (max-width: 680px) {
  .add-rule-row { flex-direction: column; align-items: stretch; }
  .type-col { width: 100%; }
}
</style>
