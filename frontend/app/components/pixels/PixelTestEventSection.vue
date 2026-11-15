<template>
  <div class="tab-pane">
    <div class="section">
      <div class="section-header">
        <h4 class="section-title">Eventos automáticos</h4>
        <BaseButton v-if="canEdit" variant="ghost" size="sm" @click="$emit('edit-config', pixel)">
          Editar configuração
        </BaseButton>
      </div>
      <div class="events-grid">
        <div v-for="ev in autoEvents" :key="ev.key" class="event-item" :class="{ on: ev.enabled }">
          <span class="event-dot" />
          <div>
            <p class="event-name">{{ ev.label }}</p>
            <p class="event-tag">{{ ev.tag }}</p>
          </div>
        </div>
      </div>
    </div>

    <div v-if="canEdit" class="section">
      <h4 class="section-title">Enviar evento de teste</h4>
      <p v-if="!hasMeta && !hasTikTok" class="section-hint">
        Este pixel não possui Meta ou TikTok configurados para envio de teste. Para testar Google Ads, use <strong>Integrações &gt; Google Ads</strong>; para GA4, valide pelo <strong>DebugView</strong>.
      </p>
      <template v-else>
        <div class="ch-tabs">
          <button type="button" class="ch-tab" :class="{ active: testForm.channel === 'meta' }" :disabled="!hasMeta" @click="testForm.channel = 'meta'">Meta</button>
          <button type="button" class="ch-tab" :class="{ active: testForm.channel === 'tiktok' }" :disabled="!hasTikTok" @click="testForm.channel = 'tiktok'">TikTok</button>
          <button type="button" class="ch-tab" :class="{ active: testForm.channel === 'all' }" :disabled="!hasMeta || !hasTikTok" @click="testForm.channel = 'all'">Todos</button>
        </div>

        <div class="test-grid">
          <BaseDropdown v-model="testForm.event_name" label="Evento" :options="testEventOptions" />
          <BaseInput
            v-model="testForm.test_event_code"
            label="Test Event Code (opcional)"
            placeholder="Ex: TEST12345"
            hint="Use apenas se quiser visualizar este disparo na aba Eventos de Teste."
          />
          <BaseInput v-model="testForm.email" label="E-mail (opcional)" placeholder="teste@exemplo.com" />
          <BaseInput v-model="testForm.phone" label="Telefone (opcional)" placeholder="+5511999999999" />
          <BaseInput v-model="testForm.value" label="Valor (opcional)" inputmode="decimal" placeholder="99.90" />
        </div>

        <p v-if="testForm.test_event_code.trim()" class="test-notice safe">
          Apenas este disparo manual usará o código <code>{{ testForm.test_event_code.trim() }}</code>. O seu pixel permanece operando normalmente em produção.
        </p>
        <p v-else class="test-notice neutral">
          Seu pixel opera em produção. Se deixar o Test Event Code vazio, este disparo manual será enviado como evento normal para a plataforma.
        </p>

        <div class="test-footer">
          <BaseButton variant="primary" :loading="testing" @click="onSendTestEvent">
            Enviar evento de teste
          </BaseButton>
        </div>

        <div v-if="testResults.length" class="test-results">
          <div
            v-for="(r, i) in testResults"
            :key="i"
            class="test-result"
            :class="{ ok: r.status_code === 200 && !r.error, fail: r.status_code !== 200 || r.error }"
          >
            <div class="result-row">
              <BrandIcon :channel="(r.channel || 'meta') as 'meta' | 'tiktok'" size="sm" />
              <code class="result-pixel">{{ r.pixel_id }}</code>
              <BaseBadge v-if="r.used_test_code" variant="default">Teste</BaseBadge>
              <BaseBadge v-else variant="success">Produção</BaseBadge>
              <BaseBadge :variant="r.status_code === 200 && !r.error ? 'success' : 'danger'">
                HTTP {{ r.status_code || 'erro' }}
              </BaseBadge>
            </div>
            <p v-if="r.error" class="result-error">{{ r.error }}</p>
            <details v-if="r.response" class="result-detail">
              <summary>Resposta</summary>
              <pre><code>{{ formatResponse(r.response) }}</code></pre>
            </details>
          </div>
          <p v-if="lastEventID" class="result-meta">Event ID: <code>{{ lastEventID }}</code></p>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import BaseDropdown from '~/components/BaseDropdown.vue'
import BrandIcon from '~/components/events/BrandIcon.vue'
import { errorMessage } from '~/utils/errorMessage'
import type { Pixel, TestEventInput, TestEventPixelResult } from '#shared/types'

const props = defineProps<{
  pixel: Pixel
  canEdit?: boolean
}>()

defineEmits<{
  'edit-config': [pixel: Pixel]
}>()

const toast = useToast()
const pixelsApi = usePixels()

const autoEvents = computed(() => {
  const c = props.pixel?.tracker_config
  if (!c) return []
  const wppTag = c.whatsapp_lead_event || 'Lead'
  return [
    ['auto_page_view', 'PageView automático', 'PageView', c.auto_page_view],
    ['auto_checkout_buttons', 'Botões de checkout', 'InitiateCheckout', c.auto_checkout_buttons],
    ['auto_form_scraping', 'Captura de formulário', 'PII', c.auto_form_scraping],
    ['auto_form_lead', 'Lead em formulário', 'Lead', c.auto_form_lead],
    ['whatsapp_protocol', 'Protocolo WhatsApp', 'WhatsApp', c.whatsapp_protocol],
    ['whatsapp_lead', 'Clique no WhatsApp', wppTag, c.whatsapp_lead],
    ['send_scrolls', 'Scroll tracking', 'Scroll_*', c.send_scrolls],
    ['send_timer_1min', 'Timer 1 minuto', 'Timer_1min', c.send_timer_1min],
    ['send_view_video', 'View Video', 'ViewVideo_*', c.send_view_video],
  ].map(([key, label, tag, en]) => ({ key: String(key), label: String(label), tag: String(tag), enabled: Boolean(en) }))
})

const hasMeta = computed(() => (props.pixel?.pixel_count ?? 0) > 0)
const hasTikTok = computed(() => (props.pixel?.tiktok_pixel_count ?? 0) > 0)
const testEventOptions = ['Lead', 'PageView', 'ViewContent', 'AddToCart', 'InitiateCheckout', 'Purchase', 'CompleteRegistration'].map((v) => ({ value: v, label: v }))

const testForm = reactive({
  event_name: 'Lead',
  test_event_code: '',
  email: '',
  phone: '',
  value: '',
  channel: 'meta' as 'meta' | 'tiktok' | 'all',
})
const testing = ref(false)
const testResults = ref<TestEventPixelResult[]>([])
const lastEventID = ref('')

watch(() => props.pixel, () => {
  testResults.value = []
  lastEventID.value = ''
  testForm.test_event_code = ''
  testForm.channel = hasMeta.value ? 'meta' : (hasTikTok.value ? 'tiktok' : 'meta')
}, { immediate: true })

const onSendTestEvent = async () => {
  testing.value = true
  testResults.value = []
  lastEventID.value = ''
  try {
    const input: TestEventInput = { event_name: testForm.event_name, channel: testForm.channel }
    if (testForm.test_event_code.trim()) input.test_event_code = testForm.test_event_code.trim()
    if (testForm.email.trim()) input.email = testForm.email.trim()
    if (testForm.phone.trim()) input.phone = testForm.phone.trim()
    const rawVal = String(testForm.value ?? '').replace(',', '.').trim()
    if (rawVal) {
      const v = parseFloat(rawVal)
      if (!Number.isNaN(v) && v > 0) input.value = v
    }
    const res = await pixelsApi.sendTestEvent(props.pixel.id, input)
    testResults.value = res.results
    lastEventID.value = res.event_id
    if (res.results.every((r) => r.status_code === 200 && !r.error)) {
      toast.success('Evento enviado', 'Confira o recebimento no Gerenciador de Eventos.')
    } else {
      toast.warning('Evento com falhas', 'Veja os detalhes do retorno abaixo.')
    }
  } catch (e: unknown) {
    toast.error('Erro no envio', errorMessage(e, 'Falha ao enviar evento de teste'))
  } finally {
    testing.value = false
  }
}

const formatResponse = (r: string) => {
  try { return JSON.stringify(JSON.parse(r), null, 2) } catch { return r }
}
</script>

<style scoped>
.tab-pane { display: flex; flex-direction: column; gap: 1.25rem; }
.section { display: flex; flex-direction: column; gap: 0.625rem; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); }
.section-hint { font-size: var(--fs-base); color: var(--text-2); line-height: 1.5; }
.events-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(170px, 1fr)); gap: 0.375rem; }
.event-item { display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem 0.75rem; border: 1px solid var(--border-soft); border-radius: 8px; background: var(--surface-soft-2); }
.event-item.on { background: var(--success-soft); border-color: var(--success); }
.event-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-3); flex-shrink: 0; }
.event-item.on .event-dot { background: var(--success); }
.event-name { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-1); }
.event-item:not(.on) .event-name, .event-tag { color: var(--text-3); }
.event-tag { font-size: var(--fs-xs); font-family: var(--font-mono); }
.ch-tabs { display: flex; gap: 0.25rem; background: var(--surface-soft); border: 1px solid var(--border-soft); border-radius: 8px; padding: 0.25rem; width: fit-content; }
.ch-tab { padding: 0.3125rem 0.875rem; background: transparent; border: none; border-radius: 6px; color: var(--text-2); font-size: var(--fs-base); font-weight: var(--fw-medium); cursor: pointer; }
.ch-tab.active { background: var(--surface-2); color: var(--brand); }
.ch-tab:disabled { opacity: 0.4; cursor: not-allowed; }
.test-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 0.75rem; }
.test-notice { font-size: var(--fs-sm); padding: 0.5rem 0.75rem; border-radius: 8px; line-height: 1.45; }
.test-notice.safe { background: var(--brand-soft-2); border: 1px solid var(--brand-border); color: var(--text-1); }
.test-notice.neutral { background: var(--surface-soft); border: 1px solid var(--border-soft); color: var(--text-2); }
.test-notice code, .result-meta code, .result-pixel { font-family: var(--font-mono); }
.test-footer { display: flex; justify-content: flex-end; }
.test-results { display: flex; flex-direction: column; gap: 0.5rem; margin-top: 0.5rem; }
.test-result { padding: 0.75rem 1rem; border: 1px solid var(--border-soft); border-radius: 10px; background: var(--surface-2); }
.test-result.ok { border-left: 4px solid var(--success); }
.test-result.fail { border-left: 4px solid var(--danger); }
.result-row { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.25rem; flex-wrap: wrap; }
.result-pixel { font-size: var(--fs-base); flex: 1; min-width: 120px; }
.result-error { font-size: var(--fs-base); color: var(--danger); }
.result-detail, .result-detail summary, .result-meta { font-size: var(--fs-sm); color: var(--text-3); margin-top: 0.5rem; }
</style>
