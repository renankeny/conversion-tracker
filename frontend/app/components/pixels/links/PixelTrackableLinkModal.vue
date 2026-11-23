<template>
  <BaseModal
    :model-value="modelValue"
    :title="editing ? 'Editar Link Rastreável' : 'Novo Link Rastreável de WhatsApp'"
    description="Redireciona automaticamente para o WhatsApp conectado, atribuindo cliques, conversas e vendas."
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <form class="link-form" @submit.prevent="onSubmit">
      <BaseInput
        v-model="form.name"
        label="Nome de identificação"
        placeholder="Ex: Bio do Instagram"
        :error="errors.name"
        required
      />

      <div class="preview-strip">
        <span class="preview-label">URL do link:</span>
        <code class="preview-code">{{ previewUrl }}</code>
        <span v-if="!editing" class="preview-auto">Gerado automaticamente (10 dígitos)</span>
      </div>

      <PixelTrackableLinkDestinationBox
        v-model:routing-mode="form.routing_mode"
        v-model:whatsapp-instance-id="form.whatsapp_instance_id"
        v-model:fallback-phone="form.fallback_phone"
        v-model:prefill-message="form.prefill_message"
        :destination-error="errors.destination"
        :instances="instances"
      />

      <div class="section-box">
        <h4 class="box-title">UTMs padrão embutidas (opcional)</h4>
        <p class="box-hint">Mantém o link curto na Bio sem parâmetros longos, atribuindo a origem automaticamente.</p>
        <div class="grid-3">
          <BaseInput v-model="form.utm_source" label="utm_source" placeholder="instagram" />
          <BaseInput v-model="form.utm_medium" label="utm_medium" placeholder="bio" />
          <BaseInput v-model="form.utm_campaign" label="utm_campaign" placeholder="perfil" />
        </div>
      </div>

      <div class="section-box">
        <BaseCheckbox
          v-model="form.send_events_enabled"
          label="Disparar evento de conversão nas plataformas de anúncios"
          hint="Desmarque caso queira apenas contabilizar cliques, conversas e vendas internamente no card."
        />

        <template v-if="form.send_events_enabled">
          <div class="grid-2">
            <BaseDropdown
              v-model="form.event_name"
              label="Evento a disparar"
              :options="eventOptions"
            />
            <BaseDropdown
              v-model="form.event_trigger_mode"
              label="Momento do disparo"
              :options="triggerOptions"
            />
          </div>

          <div class="channels-block">
            <span class="channels-label">Enviar este evento para:</span>
            <div class="channels-grid">
              <BaseCheckbox v-model="form.send_meta" label="Meta Ads (CAPI)" />
              <BaseCheckbox v-model="form.send_tiktok" label="TikTok Ads" />
              <BaseCheckbox v-model="form.send_google_ads" label="Google Ads" />
              <BaseCheckbox v-model="form.send_ga4" label="Google Analytics 4 (GA4)" />
            </div>
          </div>
        </template>
      </div>

      <BaseCheckbox
        v-model="form.enabled"
        label="Link ativo"
        hint="Quando desativado, o link deixa de redirecionar novos cliques."
      />
    </form>

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">Cancelar</BaseButton>
      <BaseButton variant="primary" :loading="saving" @click="onSubmit">
        {{ editing ? 'Salvar alterações' : 'Criar link rastreável' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import PixelTrackableLinkDestinationBox from '~/components/pixels/links/PixelTrackableLinkDestinationBox.vue'
import type { Pixel, TrackableLink, TrackableLinkInput, WhatsAppInstance } from '#shared/types'
import { useTrackableLinkForm } from '~/composables/useTrackableLinkForm'

const props = withDefaults(defineProps<{
  modelValue: boolean
  pixel: Pixel
  editing?: TrackableLink | null
  instances?: WhatsAppInstance[]
  saving?: boolean
}>(), {
  editing: null,
  instances: () => [],
  saving: false,
})

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  submit: [payload: TrackableLinkInput]
}>()

const { form, errors, previewUrl, fillForm, validate } = useTrackableLinkForm(
  reactive({
    pixel: toRef(props, 'pixel'),
    editing: toRef(props, 'editing'),
    instances: toRef(props, 'instances'),
  }),
)

watch(() => props.modelValue, (open) => {
  if (open) fillForm(props.editing)
})

const eventOptions = [
  { value: 'Lead', label: 'Lead (Recomendado)' },
  { value: 'Contact', label: 'Contact (Contato)' },
  { value: 'PageView', label: 'PageView' },
  { value: 'ViewContent', label: 'ViewContent' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout' },
]

const triggerOptions = [
  { value: 'on_first_message', label: 'Quando enviar a 1ª mensagem no WhatsApp (Recomendado)' },
  { value: 'on_click', label: 'Imediatamente no clique no link' },
]

const onSubmit = () => {
  const payload = validate()
  if (payload) emit('submit', payload)
}
</script>

<style scoped>
.link-form { display: flex; flex-direction: column; gap: 1rem; }
.grid-2 { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.75rem; }
.grid-3 { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.625rem; }
.preview-strip { display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem 0.75rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 8px; flex-wrap: wrap; }
.preview-label { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em; }
.preview-code { font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--brand); word-break: break-all; }
.preview-auto { font-size: var(--fs-xs); color: var(--text-3); margin-left: auto; }
.section-box { display: flex; flex-direction: column; gap: 0.75rem; padding: 0.875rem; background: var(--surface-soft-2); border: 1px solid var(--border-soft); border-radius: 10px; }
.box-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.box-hint { font-size: var(--fs-sm); color: var(--text-3); margin: -0.35rem 0 0; line-height: 1.4; }
.channels-block { display: flex; flex-direction: column; gap: 0.5rem; padding-top: 0.25rem; }
.channels-label { font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-2); }
.channels-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.5rem; }
@media (max-width: 640px) { .grid-2, .grid-3, .channels-grid { grid-template-columns: 1fr; } }
</style>
