<template>
  <div class="tracker-config">
    <TrackerAutoCaptureSection v-model="config" />

    <!-- 3. Meta Pixel ID -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Meta Pixel (browser side)</h4>
        <p class="section-hint">
          ID usado no <code>fbq('init', ...)</code> do Meta Pixel browser.
          Em branco usa automaticamente o primeiro Pixel ID configurado.
        </p>
      </header>

      <BaseInput
        v-model="config.meta_init_pixel_id"
        label="Meta Init Pixel ID (opcional)"
        placeholder="Deixe vazio para usar o primeiro Pixel ID"
      />
    </section>

    <!-- 4. UTM origin -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Parâmetro de UTM</h4>
        <p class="section-hint">
          Nome do parâmetro de URL onde injetamos o <code>Indexador</code>.
          Padrão: <code>utm_source</code>. Para Hotmart, use <code>xcod</code>.
        </p>
      </header>

      <BaseInput
        v-model="config.utm_origin_param"
        label="Nome do parâmetro"
        placeholder="utm_source"
      />
    </section>

    <TrackerRulesSection v-model="config" />
  </div>
</template>

<script setup lang="ts">
import TrackerAutoCaptureSection from '~/components/pixels/TrackerAutoCaptureSection.vue'
import TrackerRulesSection from '~/components/pixels/TrackerRulesSection.vue'
import type { TrackerConfig } from '#shared/types'

interface Props {
  modelValue: TrackerConfig
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: TrackerConfig]
}>()

const config = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})
</script>

<style scoped>
.tracker-config {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.config-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  border-radius: 12px;
}

.section-header {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.section-title {
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}

.section-hint {
  font-size: var(--fs-base);
  color: var(--text-2);
  line-height: 1.5;
}

.section-hint code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
}
</style>
