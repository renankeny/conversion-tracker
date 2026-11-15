<template>
  <BaseModal
    :model-value="modelValue"
    title="Transferir Pixel"
    :description="pixel ? `Mover o pixel &quot;${pixel.label || pixel.name}&quot; para outra empresa.` : ''"
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="pixel" class="transfer-content">
      <div v-if="loading" class="loading-state">Carregando empresas disponíveis...</div>
      <div v-else-if="!destinationOptions.length" class="empty-box">
        Nenhuma outra empresa disponível para receber este pixel.
      </div>
      <template v-else>
        <BaseDropdown
          v-model="targetAccountId"
          label="Empresa de destino"
          placeholder="Selecione uma empresa..."
          :options="destinationOptions"
          hint="O pixel e seu histórico de leads e conversões passarão a pertencer à empresa selecionada."
        />
        <p v-if="hasLinkedAdAccounts" class="unlink-notice">
          Vínculos OAuth da empresa anterior (Meta Ads ou Google Ads) serão desvinculados automaticamente para manter o isolamento entre empresas.
        </p>
      </template>
    </div>

    <template #footer>
      <BaseButton variant="ghost" :disabled="saving" @click="$emit('update:modelValue', false)">Cancelar</BaseButton>
      <BaseButton
        variant="primary"
        :disabled="!targetAccountId || !destinationOptions.length"
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

const props = withDefaults(defineProps<{
  modelValue: boolean
  pixel: Pixel | null
  saving?: boolean
}>(), {
  saving: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: [pixelId: number, targetAccountId: number]
}>()

const accountsAPI = useAccounts()
const accounts = ref<Array<{ id: number; name: string }>>([])
const loading = ref(false)
const targetAccountId = ref<number | ''>('')

const destinationOptions = computed(() =>
  accounts.value
    .filter((a) => a.id !== props.pixel?.account_id)
    .map((a) => ({ value: a.id, label: a.name })),
)

const hasLinkedAdAccounts = computed(() =>
  Boolean(props.pixel?.meta_account_id || props.pixel?.google_ads_account_id),
)

watch(() => props.modelValue, async (isOpen) => {
  if (!isOpen) return
  targetAccountId.value = ''
  loading.value = true
  try {
    const res = await accountsAPI.list()
    accounts.value = (res.data || []).map((a) => ({ id: a.id, name: a.name }))
  } catch {
    accounts.value = []
  } finally {
    loading.value = false
  }
})

const onConfirm = () => {
  if (!props.pixel || !targetAccountId.value) return
  emit('confirm', props.pixel.id, Number(targetAccountId.value))
}
</script>

<style scoped>
.transfer-content { display: flex; flex-direction: column; gap: 0.875rem; padding: 0.5rem 0; }
.loading-state { font-size: var(--fs-md); color: var(--text-3); padding: 0.5rem 0; }
.empty-box { padding: 1rem; border-radius: 8px; border: 1px dashed var(--border); color: var(--text-3); font-size: var(--fs-base); text-align: center; }
.unlink-notice { font-size: var(--fs-sm); color: var(--warning); background: var(--warning-soft); border: 1px solid var(--warning-border); padding: 0.625rem 0.75rem; border-radius: 8px; line-height: 1.45; margin: 0; }
</style>
