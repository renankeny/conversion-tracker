<template>
  <BaseModal
    :model-value="modelValue"
    title="Taxas & Custos Operacionais"
    description="Configure taxas de gateway, impostos e custo do produto por Pixel para calcular o Lucro Líquido Real."
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="fin-form">
      <BaseDropdown
        v-model="targetPixelId"
        label="Pixel de referência"
        :options="pixelOptions"
        placeholder="Selecione um pixel"
      />

      <div class="fin-grid">
        <BaseInput
          v-model="gatewayFeePct"
          type="number"
          label="Taxa do Gateway (%)"
          placeholder="Ex: 4.99"
          hint="Percentual cobrado sobre a receita líquida"
        />
        <BaseInput
          v-model="gatewayFixedFee"
          type="number"
          label="Taxa Fixa por Venda (R$)"
          placeholder="Ex: 2.49"
          hint="Tarifa fixa por venda aprovada líquida"
        />
        <BaseInput
          v-model="taxPct"
          type="number"
          label="Imposto / Nota Fiscal (%)"
          placeholder="Ex: 6.00"
          hint="Alíquota sobre faturamento líquido"
        />
        <BaseInput
          v-model="productCost"
          type="number"
          label="Custo do Produto / CMV (R$)"
          placeholder="Ex: 45.00"
          hint="Custo unitário médio por venda líquida"
        />
      </div>

      <p class="formula-note">
        Lucro Líquido = Receita Líquida (após reembolsos e chargebacks) − Taxas − Impostos − CMV − Gasto em Anúncios.
      </p>
    </div>

    <template #footer>
      <BaseButton variant="outline" :disabled="saving" @click="emit('update:modelValue', false)">
        Cancelar
      </BaseButton>
      <BaseButton variant="primary" :loading="saving" :disabled="!targetPixelId" @click="onSubmit">
        Salvar Custos
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { Pixel, PixelFinancialConfig } from '#shared/types'

const props = defineProps<{
  modelValue: boolean
  pixels: Pixel[]
  activePixelName: string
  saving: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [pixelId: number, config: PixelFinancialConfig]
}>()

const targetPixelId = ref<number | string>('')
const gatewayFeePct = ref('0')
const gatewayFixedFee = ref('0')
const taxPct = ref('0')
const productCost = ref('0')

const pixelOptions = computed(() =>
  props.pixels.map((p) => ({
    value: p.id,
    label: p.label ? `${p.label} (${p.name})` : p.name,
  })),
)

const hydrateFromPixel = (px?: Pixel) => {
  gatewayFeePct.value = String(px?.gateway_fee_pct ?? 0)
  gatewayFixedFee.value = String(px?.gateway_fixed_fee ?? 0)
  taxPct.value = String(px?.tax_pct ?? 0)
  productCost.value = String(px?.product_cost ?? 0)
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    const match = props.pixels.find((p) => p.name === props.activePixelName) || props.pixels[0]
    targetPixelId.value = match?.id ?? ''
    hydrateFromPixel(match)
  },
)

watch(targetPixelId, (id) => {
  const match = props.pixels.find((p) => p.id === Number(id))
  if (match) hydrateFromPixel(match)
})

const toNum = (raw: string) => {
  const v = Number(String(raw || '0').replace(',', '.'))
  return Number.isFinite(v) && v >= 0 ? v : 0
}

const onSubmit = () => {
  const id = Number(targetPixelId.value)
  if (!id) return
  emit('save', id, {
    gateway_fee_pct: toNum(gatewayFeePct.value),
    gateway_fixed_fee: toNum(gatewayFixedFee.value),
    tax_pct: toNum(taxPct.value),
    product_cost: toNum(productCost.value),
  })
}
</script>

<style scoped>
.fin-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.fin-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.85rem;
}
.formula-note {
  margin: 0;
  padding: 0.65rem 0.8rem;
  border-radius: 8px;
  background: var(--surface-1);
  border: 1px solid var(--border);
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.45;
}
@media (max-width: 600px) {
  .fin-grid {
    grid-template-columns: 1fr;
  }
}
</style>
