<template>
  <div class="step-pixel">
    <div v-if="isMessagesOnly" class="mode-banner">
      <MessageCircle :size="16" class="banner-icon" />
      <span>
        <strong>Campanha de Mensagens:</strong> não precisa configurar domínio nem tracker de site — basta definir o Pixel abaixo.
      </span>
    </div>

    <div v-if="includesSite" class="sub-tabs">
      <button type="button" class="sub-tab" :class="{ active: subTab === 'pixel' }" @click="subTab = 'pixel'">
        1. Domínio &amp; Pixel
      </button>
      <button type="button" class="sub-tab" :class="{ active: subTab === 'tracker' }" @click="subTab = 'tracker'">
        2. Tracking do Site
      </button>
    </div>

    <div v-show="!includesSite || subTab === 'pixel'" class="pane-stack">
      <div class="grid-2">
        <BaseInput
          v-model="form.label"
          label="Nome de exibição do Pixel"
          placeholder="Ex: Pixel Principal"
          required
        />
        <BaseInput
          v-model="form.name"
          label="Identificador (slug)"
          placeholder="ex: clinica-prime"
          :disabled="Boolean(createdPixel)"
          :error="errors.name"
          required
        />
      </div>

      <div v-if="includesSite" class="domain-block">
        <BaseInput
          v-model="domainsInput"
          label="Domínio(s) do site"
          placeholder="seudominio.com.br, www.seudominio.com.br"
          hint="Sites onde o script será instalado. Separe múltiplos domínios por vírgula."
          :error="domainError"
          required
        />
      </div>

      <PixelMetaTabSection
        v-if="includesMeta"
        :form="form"
        :pixel-ids-error="errors.pixel_ids"
        :is-editing="Boolean(createdPixel)"
        :has-saved-manual-meta-token="hasSavedManualMetaToken"
        @clear-manual-meta-token="clearManualMetaToken"
      />

      <PixelGoogleAdsTabSection
        v-if="includesGoogle"
        :form="form"
      />

      <div v-if="isMessagesOnly && includesGoogle && !includesMeta" class="info-note">
        No Google Ads, para rastrear cliques até o WhatsApp sem site, você pode criar um <strong>Link Rastreável</strong> após finalizar ou usar o modo <strong>Site + Mensagens</strong>.
      </div>

      <div v-if="includesSite" class="extras-box">
        <button type="button" class="extras-toggle" @click="showExtras = !showExtras">
          <span>Canais adicionais opcionais (GA4 / TikTok)</span>
          <ChevronDown :size="15" :class="{ rotated: showExtras }" />
        </button>
        <div v-if="showExtras" class="extras-grid">
          <BaseInput v-model="form.ga4_measurement_id" label="GA4 Measurement ID" placeholder="G-XXXXXXXXXX" />
          <BaseInput v-model="form.ga4_api_secret" label="GA4 API Secret" type="password" placeholder="Opcional" />
        </div>
      </div>
    </div>

    <div v-if="includesSite" v-show="subTab === 'tracker'" class="pane-stack">
      <TrackerConfigEditor v-model="form.tracker_config" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ChevronDown, MessageCircle } from 'lucide-vue-next'
import PixelMetaTabSection from '~/components/pixels/form/PixelMetaTabSection.vue'
import PixelGoogleAdsTabSection from '~/components/pixels/form/PixelGoogleAdsTabSection.vue'
import TrackerConfigEditor from '~/components/pixels/TrackerConfigEditor.vue'
import { usePixelForm } from '~/composables/usePixelForm'
import { slugifyClientName, useClientOnboardingWizard } from '~/composables/useClientOnboardingWizard'
import { errorMessage } from '~/utils/errorMessage'

const emit = defineEmits<{ done: [] }>()

const pixelsApi = usePixels()
const toast = useToast()

const {
  clientName,
  createdAccount,
  metaAccount,
  googleAccount,
  createdPixel,
  skippedMetaOAuth,
  includesMeta,
  includesGoogle,
  includesSite,
  isMessagesOnly,
} = useClientOnboardingWizard()

const pixelProp = reactive({ pixel: createdPixel.value })
const {
  form,
  domainsInput,
  errors,
  hasSavedManualMetaToken,
  clearManualMetaToken,
  validate,
  buildPayload,
} = usePixelForm(pixelProp)

const subTab = ref<'pixel' | 'tracker'>('pixel')
const showExtras = ref(false)
const domainError = ref('')
const saving = ref(false)

onMounted(() => {
  if (createdAccount.value) {
    form.account_id = createdAccount.value.id
  }
  if (!createdPixel.value) {
    const baseName = clientName.value.trim() || createdAccount.value?.name || 'principal'
    form.label = baseName
    form.name = slugifyClientName(baseName) || 'principal'
  }
  if (includesMeta.value && metaAccount.value?.id && !form.meta_account_id) {
    form.meta_account_id = metaAccount.value.id
    form.meta_ad_account_id = metaAccount.value.ad_account_id || ''
  }
  if (includesMeta.value && (skippedMetaOAuth.value || !metaAccount.value) && !form.pixel_ids.length) {
    form.pixel_ids.push('')
    form.access_tokens.push('')
  }
  if (includesGoogle.value && googleAccount.value?.id && !form.google_ads_account_id) {
    form.google_ads_account_id = googleAccount.value.id
    form.google_ads_customer_id = googleAccount.value.customer_id || ''
    form.google_ads_customer_name = googleAccount.value.customer_name || ''
    form.google_ads_login_customer_id = googleAccount.value.login_customer_id || ''
  }
})

const submit = async () => {
  domainError.value = ''
  if (includesSite.value && subTab.value === 'pixel') {
    const cleanDomains = domainsInput.value.split(',').map((d) => d.trim()).filter(Boolean)
    if (!cleanDomains.length) {
      domainError.value = 'Informe pelo menos um domínio onde o script será instalado'
      return
    }
    if (!validate()) return
    subTab.value = 'tracker'
    return
  }

  if (includesSite.value) {
    const cleanDomains = domainsInput.value.split(',').map((d) => d.trim()).filter(Boolean)
    if (!cleanDomains.length) {
      subTab.value = 'pixel'
      domainError.value = 'Informe pelo menos um domínio onde o script será instalado'
      return
    }
  }
  if (!validate()) {
    subTab.value = 'pixel'
    return
  }

  saving.value = true
  try {
    const payload = buildPayload()
    if (createdAccount.value && !createdPixel.value) {
      payload.account_id = createdAccount.value.id
    }
    const saved = createdPixel.value
      ? await pixelsApi.update(createdPixel.value.id, payload)
      : await pixelsApi.create(payload)
    createdPixel.value = saved
    pixelProp.pixel = saved
    toast.success('Pixel salvo', `"${saved.label || saved.name}" configurado com sucesso.`)
    emit('done')
  } catch (err: unknown) {
    const msg = errorMessage(err, 'Não foi possível salvar o pixel')
    if (msg.toLowerCase().includes('slug') || msg.toLowerCase().includes('já existe')) {
      subTab.value = 'pixel'
      errors.name = msg
    }
    toast.error('Erro ao salvar Pixel', msg)
  } finally {
    saving.value = false
  }
}

const primaryLabel = computed(() => {
  if (includesSite.value && subTab.value === 'pixel') return 'Configurar Tracking'
  return createdPixel.value ? 'Atualizar Pixel e continuar' : 'Salvar Pixel e continuar'
})

defineExpose({ submit, saving, primaryLabel })
</script>

<style scoped>
.step-pixel { display: flex; flex-direction: column; gap: 1rem; }
.mode-banner {
  display: flex; align-items: center; gap: 0.6rem; padding: 0.75rem 0.95rem;
  border-radius: 10px; background: var(--brand-soft); border: 1px solid var(--brand-border);
  font-size: var(--fs-sm); color: var(--text-1); line-height: 1.45;
}
.mode-banner strong { color: var(--text-strong); font-weight: var(--fw-semibold); }
.banner-icon { color: var(--brand); flex-shrink: 0; }
.sub-tabs { display: flex; gap: 0.4rem; border-bottom: 1px solid var(--border-soft); padding-bottom: 0.5rem; }
.sub-tab {
  padding: 0.4rem 0.85rem; border-radius: 8px; border: 1px solid transparent;
  background: transparent; color: var(--text-3); font-size: var(--fs-sm); font-weight: var(--fw-medium); cursor: pointer;
}
.sub-tab.active { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); font-weight: var(--fw-semibold); }
.pane-stack { display: flex; flex-direction: column; gap: 1rem; }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
.domain-block { padding: 0.85rem; border-radius: 10px; background: var(--surface-deep); border: 1px solid var(--border-soft); }
.info-note { padding: 0.75rem 0.9rem; border-radius: 9px; background: var(--surface-soft); border: 1px solid var(--border-soft); font-size: var(--fs-xs); color: var(--text-2); line-height: 1.45; }
.info-note strong { color: var(--text-1); font-weight: var(--fw-semibold); }
.extras-box { border: 1px solid var(--border-soft); border-radius: 10px; padding: 0.7rem 0.9rem; background: var(--surface-soft); }
.extras-toggle { display: flex; align-items: center; justify-content: space-between; width: 100%; background: transparent; border: none; color: var(--text-2); font-size: var(--fs-sm); font-weight: var(--fw-medium); cursor: pointer; }
.extras-toggle .rotated { transform: rotate(180deg); transition: transform 0.15s; }
.extras-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; margin-top: 0.75rem; }
@media (max-width: 640px) { .grid-2, .extras-grid { grid-template-columns: 1fr; } }
</style>
