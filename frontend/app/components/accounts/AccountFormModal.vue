<template>
  <BaseModal
    :model-value="modelValue"
    :title="editing ? 'Editar Empresa' : 'Nova Empresa'"
    :description="editing ? 'Altere os dados, identificador, status ou regras de retenção desta empresa.' : 'Cadastre uma nova empresa para isolar pixels, eventos, leads e relatórios.'"
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <form class="account-form" @submit.prevent="onSubmit">
      <BaseInput
        v-model="form.name"
        label="Nome da empresa"
        placeholder="Ex: Cliente Acme Ltda"
        required
        @update:model-value="onNameChange"
      />

      <BaseInput
        v-model="form.slug"
        label="Identificador (Slug)"
        placeholder="ex: cliente-acme"
        hint="Usado internamente para identificação única (apenas letras minúsculas, números e hífen)."
      />

      <div class="form-section">
        <BaseCheckbox
          v-model="form.enabled"
          label="Empresa ativa"
          hint="Quando desativada, os pixels vinculados a esta empresa pausam o recebimento de novos eventos."
        />
      </div>

      <div class="form-section">
        <h4 class="section-title">Retenção de Dados (Opcional)</h4>
        <p class="section-hint">
          Defina por quantos dias manter o histórico desta empresa. Deixe em branco para usar o padrão global do servidor.
        </p>
        <div class="retention-grid">
          <BaseInput
            v-model="form.sentEventsDays"
            type="number"
            label="Eventos enviados (dias)"
            placeholder="Padrão global"
            hint="0 = nunca apagar automaticamente"
          />
          <BaseInput
            v-model="form.countersDays"
            type="number"
            label="Contadores da dashboard (dias)"
            placeholder="Padrão global"
            hint="0 = nunca apagar automaticamente"
          />
        </div>
      </div>
    </form>

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">
        Cancelar
      </BaseButton>
      <BaseButton variant="primary" :loading="saving" @click="onSubmit">
        {{ editing ? 'Salvar alterações' : 'Cadastrar empresa' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { Account, CreateAccountInput, UpdateAccountInput } from '#shared/types'

const props = defineProps<{
  modelValue: boolean
  editing: Account | null
  saving: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [payload: CreateAccountInput & UpdateAccountInput]
}>()

const slugTouched = ref(false)

const form = reactive({
  name: '',
  slug: '',
  enabled: true,
  sentEventsDays: '' as string | number,
  countersDays: '' as string | number,
})

const toSlug = (val: string): string =>
  val
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')

const onNameChange = (val: string | number) => {
  if (!props.editing && !slugTouched.value) {
    form.slug = toSlug(String(val))
  }
}

watch(
  () => [props.modelValue, props.editing] as const,
  ([open, acc]) => {
    if (!open) return
    if (acc) {
      form.name = acc.name
      form.slug = acc.slug
      form.enabled = acc.enabled
      form.sentEventsDays = acc.sent_events_retention_days != null ? String(acc.sent_events_retention_days) : ''
      form.countersDays = acc.event_counters_retention_days != null ? String(acc.event_counters_retention_days) : ''
      slugTouched.value = true
    } else {
      form.name = ''
      form.slug = ''
      form.enabled = true
      form.sentEventsDays = ''
      form.countersDays = ''
      slugTouched.value = false
    }
  },
  { immediate: true },
)

const parseOptionalInt = (raw: string | number): number | null => {
  const str = String(raw ?? '').trim()
  if (str === '') return null
  const n = Number.parseInt(str, 10)
  if (Number.isNaN(n) || n < 0) return null
  return n
}

const onSubmit = () => {
  const cleanSlug = toSlug(form.slug || form.name)
  emit('submit', {
    name: form.name.trim(),
    slug: cleanSlug || undefined,
    enabled: form.enabled,
    sent_events_retention_days: parseOptionalInt(form.sentEventsDays),
    event_counters_retention_days: parseOptionalInt(form.countersDays),
    update_retention: true,
  })
}
</script>

<style scoped>
.account-form {
  display: flex;
  flex-direction: column;
  gap: 1.125rem;
}
.form-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-top: 0.875rem;
  border-top: 1px solid var(--border-soft);
}
.section-title {
  font-family: var(--font-heading);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}
.section-hint {
  font-size: var(--fs-sm);
  color: var(--text-3);
  line-height: 1.45;
}
.retention-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem;
  margin-top: 0.25rem;
}
</style>

