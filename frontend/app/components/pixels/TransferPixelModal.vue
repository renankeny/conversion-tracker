<template>
  <BaseModal
    :model-value="modelValue"
    title="Transferir Pixel"
    :description="pixel ? `Mover o pixel \&quot;${pixel.name}\&quot; para outra empresa.` : ''"
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="pixel" class="transfer-content">
      <div v-if="loading" class="loading-state">Carregando empresas...</div>
      <BaseDropdown
        v-else
        v-model="targetAccountId"
        label="Empresa Destino"
        placeholder="Selecione uma empresa..."
        :options="accounts.map(a => ({ value: a.id, label: a.name }))"
        hint="Todos os eventos futuros deste pixel passarão a ser registrados na empresa selecionada."
      />
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">Cancelar</BaseButton>
      <BaseButton 
        variant="primary" 
        :disabled="!targetAccountId || targetAccountId === pixel?.account_id" 
        :loading="saving" 
        @click="onConfirm"
      >
        Transferir Pixel
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { Pixel } from '#shared/types'

const props = defineProps<{
  modelValue: boolean
  pixel: Pixel | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'confirm': [pixelId: number, targetAccountId: number]
}>()

const accountsAPI = useAccounts()
const accounts = ref<Array<{ id: number; name: string }>>([])
const loading = ref(false)
const saving = ref(false)

const targetAccountId = ref<number | ''>('')

watch(() => props.modelValue, async (isOpen) => {
  if (isOpen) {
    targetAccountId.value = ''
    saving.value = false
    loading.value = true
    try {
      const res = await accountsAPI.list()
      accounts.value = (res.data || []).map((a) => ({ id: a.id, name: a.name }))
    } catch (err) {
      console.error('Falha ao carregar empresas', err)
    } finally {
      loading.value = false
    }
  }
})

const onConfirm = () => {
  if (!props.pixel || !targetAccountId.value) return
  saving.value = true
  emit('confirm', props.pixel.id, Number(targetAccountId.value))
  setTimeout(() => {
    saving.value = false
  }, 1500)
}
</script>

<style scoped>
.transfer-content {
  padding: 1rem 0;
}
.form-field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}
.form-label {
  font-size: var(--fs-md);
  font-weight: var(--fw-medium);
  color: var(--text-strong);
}
.form-select {
  background: var(--surface-2);
  border: 1px solid var(--border);
  color: var(--text-strong);
  padding: 0.625rem;
  border-radius: 8px;
  font-size: var(--fs-md);
  outline: none;
  transition: border-color 0.2s;
}
.form-select:focus {
  border-color: var(--brand);
}
.form-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.form-hint {
  font-size: var(--fs-base);
  color: var(--text-3);
  margin-top: 0.25rem;
}
.loading-state {
  font-size: var(--fs-md);
  color: var(--text-3);
  padding: 0.5rem 0;
}
</style>
