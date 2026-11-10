<template>
  <div class="auto-capture-sections">
    <!-- 1. Eventos automáticos -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Eventos automáticos</h4>
        <p class="section-hint">
          Eventos que o tracker dispara sem precisar de código no site.
        </p>
      </header>

      <div class="toggles-grid">
        <ToggleCard
          v-model="config.auto_page_view"
          title="PageView automático"
          description="Dispara PageView ao carregar a página (recomendado)"
          tag="PageView"
        />
        <ToggleCard
          v-model="config.auto_checkout_buttons"
          title="Botões de checkout"
          description="Detecta links com a palavra-chave abaixo, reescreve UTMs e dispara InitiateCheckout no clique"
          tag="InitiateCheckout"
        />
        <ToggleCard
          v-model="config.auto_form_scraping"
          title="Captura de formulário"
          description="Lê email, telefone e nome dos forms da página automaticamente"
          tag="PII"
        />
        <ToggleCard
          v-model="config.auto_form_lead"
          title="Lead em formulário"
          description="Dispara Lead automaticamente quando um form com email ou telefone é submetido"
          tag="Lead"
        />
        <ToggleCard
          v-model="config.whatsapp_protocol"
          title="Protocolo WhatsApp"
          description="Adiciona &quot;Protocolo: <id>&quot; no parâmetro text= dos links wa.me e api.whatsapp.com — correlaciona a conversa que entra no WhatsApp com o lead"
          tag="WhatsApp"
        />
        <div class="toggle-with-option">
          <ToggleCard
            v-model="config.whatsapp_lead"
            title="Evento no clique do WhatsApp"
            description="Dispara o evento escolhido abaixo (web + servidor com mesmo eventID, deduplicado pela Meta) quando o usuário clica em um link wa.me. Independente do toggle acima."
            :tag="whatsappLeadEvent"
          />
          <BaseDropdown
            v-if="config.whatsapp_lead"
            v-model="whatsappLeadEvent"
            label="Qual evento disparar"
            :options="whatsappLeadEventOptions"
          />
        </div>
        <ToggleCard
          v-model="config.send_scrolls"
          title="Scroll 25/50/75/90"
          description="Dispara eventos custom ao atingir percentuais de rolagem"
          tag="Scroll_*"
        />
        <ToggleCard
          v-model="config.send_timer_1min"
          title="Timer 1 minuto"
          description="Dispara Timer_1min após 60s na página"
          tag="Timer_1min"
        />
        <ToggleCard
          v-model="config.send_view_video"
          title="View Video"
          description="Dispara ViewVideo_25/50/75/90 ao reproduzir <video>"
          tag="ViewVideo_*"
        />
      </div>
    </section>

    <!-- 2. Produto -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Produto e checkout</h4>
        <p class="section-hint">
          Valor e moeda enviados nos eventos InitiateCheckout/Purchase.
        </p>
      </header>

      <div class="form-grid">
        <BaseInput
          v-model="config.product_checkout"
          label="Palavra-chave do checkout"
          placeholder="Ex: pay, /checkout, comprar"
          hint="Substring usada para identificar links de checkout no site"
        />
        <BaseInput
          v-model="config.product_currency"
          label="Moeda"
          placeholder="BRL"
          hint="Ex: BRL, USD, EUR"
        />
      </div>

      <div class="prices-block">
        <label class="prices-label">Valores dos produtos / ofertas</label>
        <p class="section-hint">
          Cada valor corresponde a um botão de checkout, em ordem de aparição na página.
        </p>
        <div
          v-for="(_, idx) in (config.product_value || [])"
          :key="idx"
          class="price-row"
        >
          <BaseInput
            :model-value="formatPrice(config.product_value[idx])"
            type="number"
            :placeholder="'Valor do botão ' + (idx + 1)"
            @update:model-value="(v) => updatePrice(idx, v)"
          />
          <button
            type="button"
            class="row-remove"
            aria-label="Remover preço"
            @click="removePrice(idx)"
          >
            ×
          </button>
        </div>
        <BaseButton type="button" variant="ghost" size="sm" @click="addPrice">
          + Adicionar valor
        </BaseButton>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import ToggleCard from '~/components/pixels/ToggleCard.vue'
import BaseDropdown from '~/components/BaseDropdown.vue'
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

const whatsappLeadEventOptions = [
  { value: 'Lead', label: 'Lead' },
  { value: 'Contact', label: 'Contact' },
]

const whatsappLeadEvent = computed({
  get: () => config.value.whatsapp_lead_event || 'Lead',
  set: (v) => {
    config.value = { ...config.value, whatsapp_lead_event: v }
  },
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

const updatePrice = (idx: number, raw: string) => {
  const arr = [...(config.value.product_value || [])]
  arr[idx] = parsePrice(raw)
  config.value = { ...config.value, product_value: arr }
}

const addPrice = () => {
  config.value = {
    ...config.value,
    product_value: [...(config.value.product_value || []), 0],
  }
}

const removePrice = (idx: number) => {
  const next = [...(config.value.product_value || [])]
  next.splice(idx, 1)
  config.value = { ...config.value, product_value: next }
}
</script>

<style scoped>
.auto-capture-sections { display: flex; flex-direction: column; gap: 1.5rem; }
.config-section { display: flex; flex-direction: column; gap: 0.75rem; padding: 1rem; background: var(--surface-soft-2); border: 1px solid var(--border-soft); border-radius: 12px; }
.section-header { display: flex; flex-direction: column; gap: 0.25rem; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); }
.section-hint { font-size: var(--fs-base); color: var(--text-2); line-height: 1.5; }
.toggles-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 0.625rem; align-items: start; }
.toggle-with-option { display: flex; flex-direction: column; gap: 0.5rem; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
@media (max-width: 640px) { .form-grid { grid-template-columns: 1fr; } }
.prices-block { display: flex; flex-direction: column; gap: 0.5rem; margin-top: 0.5rem; }
.prices-label { font-size: var(--fs-md); font-weight: var(--fw-medium); color: var(--text-1); }
.price-row { position: relative; display: flex; align-items: flex-end; gap: 0.5rem; }
.price-row > :first-child { flex: 1; }
.row-remove { background: transparent; border: 1px solid var(--border); color: var(--text-3); font-size: var(--fs-2xl); line-height: 1; cursor: pointer; padding: 0.375rem 0.625rem; border-radius: 6px; transition: all 0.15s; flex-shrink: 0; align-self: flex-end; }
.row-remove:hover { background: var(--danger-soft); border-color: var(--danger); color: var(--danger); }
</style>
