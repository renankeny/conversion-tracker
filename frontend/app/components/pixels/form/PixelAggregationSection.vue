<template>
  <div class="form-section aggregation-card">
    <div class="aggregation-header">
      <BaseCheckbox
        :model-value="enabled"
        @update:model-value="$emit('update:enabled', $event)"
      >
        <span class="aggregation-title">Agrupamento de Conversões (Order Bump &amp; Upsell)</span>
      </BaseCheckbox>
    </div>
    <p class="section-hint">
      Consolida compras consecutivas do mesmo cliente em um único evento de Purchase com a soma dos valores e contagem de itens para Meta CAPI, TikTok e Google Ads.
    </p>

    <div v-if="enabled" class="aggregation-slider-box">
      <div class="slider-header-row">
        <span class="slider-label">Janela de Agrupamento:</span>
        <span class="slider-badge">{{ windowMinutes }} {{ windowMinutes === 1 ? 'minuto' : 'minutos' }}</span>
      </div>
      <input
        :value="windowMinutes"
        type="range"
        min="1"
        max="30"
        step="1"
        class="aggregation-slider"
        @input="$emit('update:windowMinutes', Number(($event.target as HTMLInputElement).value))"
      />
      <div class="slider-ticks">
        <span>1 min</span>
        <span>10 min (recomendado)</span>
        <span>30 min</span>
      </div>
      <p class="form-hint" style="margin-top: 0.5rem; font-size: var(--fs-sm);">
        O {{ brandName }} aguardará este tempo após a 1ª compra antes de disparar o evento consolidado. O painel exibirá a contagem de todos os produtos comprados.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  enabled: boolean
  windowMinutes: number
}>()

defineEmits<{
  'update:enabled': [value: boolean]
  'update:windowMinutes': [value: number]
}>()

const { brandName } = useBranding()
</script>

<style scoped>
.form-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--border-soft);
}
.section-hint {
  font-size: var(--fs-base);
  color: var(--text-3);
}
.form-hint {
  font-size: var(--fs-sm);
  color: var(--text-3);
}
.aggregation-card {
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 1rem;
}
.aggregation-title {
  font-weight: var(--fw-semibold);
  color: var(--text-1);
}
.aggregation-slider-box {
  margin-top: 1rem;
  padding-top: 0.75rem;
  border-top: 1px dashed var(--border-soft);
}
.slider-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}
.slider-label {
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--text-2);
}
.slider-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.6rem;
  background: var(--brand-soft);
  color: var(--brand);
  border: 1px solid var(--brand-border);
  border-radius: 9999px;
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}
.aggregation-slider {
  width: 100%;
  height: 6px;
  border-radius: 3px;
  background: var(--surface-3);
  outline: none;
  cursor: pointer;
  accent-color: var(--brand);
}
.slider-ticks {
  display: flex;
  justify-content: space-between;
  font-size: var(--fs-xs);
  color: var(--text-3);
  margin-top: 0.25rem;
}
</style>
