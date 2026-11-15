<template>
  <div>
    <PixelGoogleAdsTabSection
      v-show="activeTab === 'gads'"
      :form="form"
    />

    <div v-show="activeTab === 'ga4'" class="tab-content">
      <div class="section-header">
        <h4 class="section-title">
          <BrandIcon channel="ga4" size="sm" :colored="true" />
          Google Analytics 4 (GA4)
        </h4>
        <span class="badge-optional">opcional</span>
      </div>
      <p class="section-hint">
        Integração Server-Side via Measurement Protocol. O Measurement ID começa com "G-" e o API Secret é gerado no GA4 em Fluxos de Dados.
      </p>

      <div class="form-section">
        <BaseInput
          v-model="form.ga4_measurement_id"
          label="Measurement ID"
          placeholder="G-XXXXXXXXXX"
          hint="Deixe vazio para desativar o envio ao GA4 e remover o API Secret salvo."
        />
        <BaseInput
          v-model="form.ga4_api_secret"
          type="password"
          label="API Secret"
          :placeholder="hasSavedGa4Secret && form.ga4_measurement_id.trim() ? 'Secret salvo (deixe vazio para manter)' : 'Cole o API Secret do fluxo GA4'"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import BrandIcon from '~/components/events/BrandIcon.vue'
import PixelGoogleAdsTabSection from '~/components/pixels/form/PixelGoogleAdsTabSection.vue'
import type { GoogleAdsTabFormState } from '~/composables/usePixelGoogleAdsTab'

interface AdvancedFormState extends GoogleAdsTabFormState {
  ga4_measurement_id: string
  ga4_api_secret: string
}

defineProps<{
  activeTab: string
  form: AdvancedFormState
  hasSavedGa4Secret?: boolean
}>()
</script>

<style scoped>
.tab-content { display: flex; flex-direction: column; gap: 1rem; }
.form-section { display: flex; flex-direction: column; gap: 0.75rem; padding-top: 0.5rem; border-top: 1px solid var(--border-soft); }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); display: inline-flex; align-items: center; gap: 0.5rem; margin: 0; }
.badge-optional { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); background: var(--surface-soft); border: 1px solid var(--border-soft); padding: 0.125rem 0.4rem; border-radius: 4px; }
.section-hint { font-size: var(--fs-base); color: var(--text-3); margin: 0; }
</style>
