<template>
  <div class="rules-sections">
    <!-- 5. Page rules -->
    <section class="config-section">
      <header class="section-header">
        <div class="section-title-row">
          <h4 class="section-title">Regras por página</h4>
          <BaseButton type="button" variant="ghost" size="sm" @click="addPageRule">
            + Adicionar regra
          </BaseButton>
        </div>
        <p class="section-hint">
          Dispara um evento quando a URL contém a substring informada.
        </p>
      </header>

      <div v-if="!config.page_rules.length" class="empty-rules">
        Nenhuma regra cadastrada.
      </div>

      <div v-for="(rule, idx) in config.page_rules" :key="idx" class="rule-row">
        <div class="rule-grid">
          <BaseInput
            :model-value="rule.page"
            label="Substring na URL"
            placeholder="ex: /obrigado"
            @update:model-value="(v) => updateRuleField(idx, 'page', v)"
          />
          <BaseInput
            :model-value="rule.event"
            label="Evento"
            placeholder="ex: Lead, Purchase"
            @update:model-value="(v) => updateRuleField(idx, 'event', v)"
          />
          <BaseInput
            :model-value="formatPrice(rule.value)"
            label="Valor (opcional)"
            type="number"
            placeholder="0"
            @update:model-value="(v) => updateRuleValue(idx, v)"
          />
          <BaseInput
            :model-value="rule.currency || ''"
            label="Moeda (opcional)"
            placeholder="BRL"
            @update:model-value="(v) => updateRuleField(idx, 'currency', v)"
          />
        </div>
        <button
          type="button"
          class="row-remove"
          aria-label="Remover regra"
          @click="removePageRule(idx)"
        >
          ×
        </button>
      </div>
    </section>

    <!-- 6. Form mappings -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Mapeamento de campos de formulário</h4>
        <p class="section-hint">
          Atributos <code>name</code> dos inputs que o tracker procura para extrair
          os dados do usuário automaticamente. Separe por vírgula.
        </p>
      </header>

      <div class="form-grid">
        <BaseInput
          v-model="formEmail"
          label="Campos de e-mail"
          placeholder="email, form_fields[email]"
        />
        <BaseInput
          v-model="formPhone"
          label="Campos de telefone"
          placeholder="tel, telefone, phone"
        />
        <BaseInput
          v-model="formFirstName"
          label="Campos de primeiro nome"
          placeholder="nome, name, first_name"
        />
        <BaseInput
          v-model="formLastName"
          label="Campos de sobrenome"
          placeholder="sobrenome, lastname"
        />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import type { TrackerConfig } from '#shared/types'

const props = defineProps<{
  modelValue: TrackerConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: TrackerConfig]
}>()

const config = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const formatPrice = (n: number | undefined | null): string => {
  if (n === undefined || n === null || typeof n !== 'number' || Number.isNaN(n)) return ''
  return n === 0 ? '' : String(n)
}

const parsePrice = (raw: string): number => {
  if (!raw) return 0
  const n = parseFloat(String(raw).replace(',', '.').trim())
  return Number.isNaN(n) ? 0 : n
}

const updateRuleField = (idx: number, field: 'page' | 'event' | 'currency', value: string) => {
  const rules = [...(config.value.page_rules || [])]
  if (!rules[idx]) return
  rules[idx] = { ...rules[idx], [field]: value }
  config.value = { ...config.value, page_rules: rules }
}

const updateRuleValue = (idx: number, raw: string) => {
  const rules = [...(config.value.page_rules || [])]
  if (!rules[idx]) return
  const v = raw.trim() === '' ? undefined : parsePrice(raw)
  rules[idx] = { ...rules[idx], value: v }
  config.value = { ...config.value, page_rules: rules }
}

const addPageRule = () => {
  config.value = {
    ...config.value,
    page_rules: [...(config.value.page_rules || []), { page: '', event: '', currency: 'BRL' }],
  }
}

const removePageRule = (idx: number) => {
  const next = [...(config.value.page_rules || [])]
  next.splice(idx, 1)
  config.value = { ...config.value, page_rules: next }
}

const csvToArray = (s: string): string[] => s.split(',').map((x) => x.trim()).filter(Boolean)
const emptyFormMappings = () => ({ email: [], phone: [], first_name: [], last_name: [] })

const makeMappingComputed = (key: 'email' | 'phone' | 'first_name' | 'last_name') =>
  computed({
    get: () => (config.value.form_mappings?.[key] || []).join(', '),
    set: (s: string) => {
      config.value = {
        ...config.value,
        form_mappings: { ...(config.value.form_mappings || emptyFormMappings()), [key]: csvToArray(s) },
      }
    },
  })

const formEmail = makeMappingComputed('email')
const formPhone = makeMappingComputed('phone')
const formFirstName = makeMappingComputed('first_name')
const formLastName = makeMappingComputed('last_name')
</script>

<style scoped>
.rules-sections { display: flex; flex-direction: column; gap: 1.5rem; }
.config-section { display: flex; flex-direction: column; gap: 0.75rem; padding: 1rem; background: var(--surface-soft-2); border: 1px solid var(--border-soft); border-radius: 12px; }
.section-header { display: flex; flex-direction: column; gap: 0.25rem; }
.section-title-row { display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); }
.section-hint { font-size: var(--fs-base); color: var(--text-2); line-height: 1.5; }
.section-hint code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: var(--surface-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
@media (max-width: 640px) { .form-grid { grid-template-columns: 1fr; } }
.rule-row { position: relative; display: flex; align-items: flex-end; gap: 0.5rem; background: var(--surface-2); padding: 0.75rem; border: 1px solid var(--border-soft); border-radius: 10px; margin-top: 0.5rem; }
.rule-grid { flex: 1; display: grid; grid-template-columns: 2fr 1fr 1fr 1fr; gap: 0.5rem; }
@media (max-width: 768px) { .rule-grid { grid-template-columns: 1fr 1fr; } }
.row-remove { background: transparent; border: 1px solid var(--border); color: var(--text-3); font-size: var(--fs-2xl); line-height: 1; cursor: pointer; padding: 0.375rem 0.625rem; border-radius: 6px; transition: all 0.15s; flex-shrink: 0; align-self: flex-end; }
.row-remove:hover { background: var(--danger-soft); border-color: var(--danger); color: var(--danger); }
.empty-rules { font-size: var(--fs-base); color: var(--text-3); font-style: italic; padding: 0.5rem 0; }
</style>
