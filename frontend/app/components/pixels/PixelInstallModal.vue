<template>
  <BaseModal :model-value="modelValue" title="Instalar Pixel" :description="pixel ? `Configure e instale o pixel &quot;${pixel.name}&quot;` : ''" size="lg" @update:model-value="$emit('update:modelValue', $event)">
    <div v-if="pixel" class="install-content">
      <div class="tabs">
        <button class="tab" :class="{ active: tab === 'script' }" @click="tab = 'script'">Script</button>
        <button class="tab" :class="{ active: tab === 'events' }" @click="tab = 'events'">Eventos &amp; Teste</button>
      </div>

      <div v-show="tab === 'script'" class="tab-pane">
        <div v-if="currentToken" class="section">
          <div class="section-header">
            <h4 class="section-title">Token público</h4>
            <div style="display:flex;gap:.5rem;align-items:center">
              <span class="badge-pill">Opaco · Seguro</span>
              <BaseButton variant="ghost" size="sm" @click="showRegenConfirm = true">Regenerar</BaseButton>
            </div>
          </div>
          <p class="section-hint">Identificador usado no snippet. Não expõe o nome interno do pixel no código-fonte do site.</p>
          <div class="copy-row">
            <code class="copy-val token-val">{{ currentToken }}</code>
            <button class="copy-mini" @click="copy(currentToken, 'token')">{{ miniCopied === 'token' ? '✓' : '📋' }}</button>
          </div>
        </div>

        <div class="section">
          <div class="section-header">
            <h4 class="section-title">Cole no &lt;head&gt; do site</h4>
            <BaseButton variant="outline" size="sm" @click="copySnippet">{{ copied ? '✓ Copiado!' : 'Copiar' }}</BaseButton>
          </div>
          <pre class="code-block"><code>{{ snippet }}</code></pre>
          <p class="section-hint">Funciona em WordPress, Shopify, GTM. Cole antes de <code>&lt;/head&gt;</code>.</p>
        </div>

        <div class="section">
          <h4 class="section-title">Disparar eventos do código</h4>
          <pre class="code-block small"><code>{{ manualSnippet }}</code></pre>
        </div>

        <div class="section">
          <h4 class="section-title">Domínios autorizados</h4>
          <div class="domains-list">
            <span v-for="d in pixel.domains" :key="d" class="domain-pill">✓ {{ d }}</span>
            <span v-if="!pixel.domains?.length" class="hint-warn">Nenhum domínio cadastrado — edite o pixel.</span>
          </div>
        </div>
      </div>

      <div v-show="tab === 'events'" class="tab-pane">
        <div class="section">
          <div class="section-header">
            <h4 class="section-title">Eventos automáticos</h4>
            <BaseButton variant="ghost" size="sm" @click="$emit('edit-config', pixel)">Editar configuração</BaseButton>
          </div>
          <div class="events-grid">
            <div v-for="ev in autoEvents" :key="ev.key" class="event-item" :class="{ on: ev.enabled }">
              <span class="event-dot" />
              <div><p class="event-name">{{ ev.label }}</p><p class="event-tag">{{ ev.tag }}</p></div>
            </div>
          </div>
        </div>

        <div class="section">
          <h4 class="section-title">Enviar evento de teste</h4>
          <p v-if="!hasMeta && !hasTikTok" class="section-hint">
            Este pixel não possui Meta ou TikTok configurados. Para testar Google Ads, use <strong>Integrações &gt; Google Ads</strong>; para GA4, valide pelo <strong>DebugView</strong>.
          </p>
          <template v-else>
            <p class="section-hint">Usa o <code>test_event_code</code> do pixel — não conta como conversão real.</p>
            <div class="ch-tabs">
              <button class="ch-tab" :class="{ active: testForm.channel === 'meta' }" :disabled="!hasMeta" @click="testForm.channel = 'meta'">Meta</button>
              <button class="ch-tab" :class="{ active: testForm.channel === 'tiktok' }" :disabled="!hasTikTok" @click="testForm.channel = 'tiktok'">TikTok</button>
              <button class="ch-tab" :class="{ active: testForm.channel === 'all' }" :disabled="!hasMeta || !hasTikTok" @click="testForm.channel = 'all'">Todos</button>
            </div>
            <div class="test-grid">
              <BaseDropdown v-model="testForm.event_name" label="Evento" :options="testEventOptions" />
              <BaseInput v-model="testForm.email" label="E-mail (opcional)" placeholder="teste@exemplo.com" />
              <BaseInput v-model="testForm.phone" label="Telefone (opcional)" placeholder="+5511999999999" />
              <BaseInput v-model="testForm.value" label="Valor (opcional)" type="number" placeholder="99.90" />
            </div>
            <div class="test-footer"><BaseButton variant="primary" :loading="testing" @click="onSendTestEvent">Enviar evento de teste</BaseButton></div>
            <div v-if="testResults.length" class="test-results">
              <div v-for="(r, i) in testResults" :key="i" class="test-result" :class="{ ok: r.status_code === 200 && !r.error, fail: r.status_code !== 200 || r.error }">
                <div class="result-row">
                  <BrandIcon :channel="(r.channel || 'meta') as 'meta' | 'tiktok'" size="sm" />
                  <code class="result-pixel">{{ r.pixel_id }}</code>
                  <BaseBadge :variant="r.status_code === 200 && !r.error ? 'success' : 'danger'">HTTP {{ r.status_code || 'erro' }}</BaseBadge>
                </div>
                <p v-if="r.error" class="result-error">{{ r.error }}</p>
                <details v-if="r.response" class="result-detail"><summary>Resposta</summary><pre><code>{{ formatResponse(r.response) }}</code></pre></details>
              </div>
              <p v-if="lastEventID" class="result-meta">Event ID: <code>{{ lastEventID }}</code></p>
            </div>
          </template>
        </div>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">Fechar</BaseButton>
      <BaseButton v-if="tab === 'script'" variant="primary" @click="copySnippet">{{ copied ? '✓ Copiado!' : 'Copiar Script' }}</BaseButton>
    </template>
  </BaseModal>

  <ConfirmModal v-model="showRegenConfirm" title="Gerar novo token?" message="O token atual vai parar de funcionar imediatamente. Atualize o snippet no site após gerar o novo." confirm-label="Gerar novo token" variant="danger" @confirm="onRegenerateToken" />
</template>

<script setup lang="ts">
import BaseDropdown from '~/components/BaseDropdown.vue'
import ConfirmModal from '~/components/ConfirmModal.vue'
import BrandIcon from '~/components/events/BrandIcon.vue'
import { buildPixelSnippet } from '~/utils/apiOrigin'
import { errorMessage } from '~/utils/errorMessage'
import type { Pixel, TestEventInput, TestEventPixelResult } from '#shared/types'

const props = defineProps<{ modelValue: boolean; pixel: Pixel | null }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; 'edit-config': [pixel: Pixel]; 'token-regenerated': [pixel: Pixel] }>()
const { brandName } = useBranding()
const config = useRuntimeConfig()
const toast = useToast()
const pixelsApi = usePixels()

const tab = ref<'script' | 'events'>('script')
const currentToken = computed(() => props.pixel?.public_token || '')
const snippet = computed(() => props.pixel ? buildPixelSnippet(config.public.apiUrl as string, props.pixel.name, props.pixel.public_token || undefined, brandName.value) : '')
const manualSnippet = `// Lead no clique do WhatsApp\ntf('send', { eventType: 'Lead' });\n\n// Compra com valor\ntf('send', { eventType: 'Purchase', value: 99.90, currency: 'BRL' });`

const autoEvents = computed(() => {
  const c = props.pixel?.tracker_config
  if (!c) return []
  return [
    ['auto_page_view', 'PageView automático', 'PageView', c.auto_page_view],
    ['auto_checkout_buttons', 'Botões de checkout', 'InitiateCheckout', c.auto_checkout_buttons],
    ['auto_form_scraping', 'Captura de formulário', 'PII', c.auto_form_scraping],
    ['auto_form_lead', 'Lead em formulário', 'Lead', c.auto_form_lead],
    ['whatsapp_protocol', 'Protocolo WhatsApp', 'WhatsApp', c.whatsapp_protocol],
    ['whatsapp_lead', 'Lead no WhatsApp', 'Lead', c.whatsapp_lead],
    ['send_scrolls', 'Scroll tracking', 'Scroll_*', c.send_scrolls],
    ['send_timer_1min', 'Timer 1 minuto', 'Timer_1min', c.send_timer_1min],
    ['send_view_video', 'View Video', 'ViewVideo_*', c.send_view_video],
  ].map(([key, label, tag, en]) => ({ key: String(key), label: String(label), tag: String(tag), enabled: !!en }))
})

const miniCopied = ref('')
const showRegenConfirm = ref(false)
const regenerating = ref(false)

const onRegenerateToken = async () => {
  if (!props.pixel) return
  regenerating.value = true
  try {
    const r = await pixelsApi.regenerateToken(props.pixel.id)
    emit('token-regenerated', { ...props.pixel, public_token: r.public_token } as Pixel)
    toast.success('Novo token gerado', 'Atualize o snippet no site.')
  } catch (e: unknown) { toast.error('Erro', errorMessage(e, 'Falha ao gerar novo token')) }
  finally { regenerating.value = false; showRegenConfirm.value = false }
}

const hasMeta = computed(() => (props.pixel?.pixel_count ?? 0) > 0)
const hasTikTok = computed(() => (props.pixel?.tiktok_pixel_count ?? 0) > 0)
const testEventOptions = ['Lead', 'PageView', 'ViewContent', 'AddToCart', 'InitiateCheckout', 'Purchase', 'CompleteRegistration'].map(v => ({ value: v, label: v }))
const testForm = reactive({ event_name: 'Lead', email: '', phone: '', value: '', channel: 'meta' as 'meta' | 'tiktok' | 'all' })
const testing = ref(false)
const testResults = ref<TestEventPixelResult[]>([])
const lastEventID = ref('')

watch(() => props.pixel, () => {
  testResults.value = []
  lastEventID.value = ''
  testForm.channel = hasMeta.value ? 'meta' : (hasTikTok.value ? 'tiktok' : 'meta')
}, { immediate: true })

const onSendTestEvent = async () => {
  if (!props.pixel) return
  testing.value = true; testResults.value = []; lastEventID.value = ''
  try {
    const input: TestEventInput = { event_name: testForm.event_name, channel: testForm.channel }
    if (testForm.email.trim()) input.email = testForm.email.trim()
    if (testForm.phone.trim()) input.phone = testForm.phone.trim()
    const v = parseFloat(testForm.value.replace(',', '.'))
    if (!Number.isNaN(v)) input.value = v
    const res = await pixelsApi.sendTestEvent(props.pixel.id, input)
    testResults.value = res.results; lastEventID.value = res.event_id
    if (res.results.every(r => r.status_code === 200 && !r.error)) toast.success('Enviado!', 'Verifique em Events Manager → Test Events')
    else toast.warning('Evento com falhas', 'Veja os detalhes')
  } catch (e: unknown) { toast.error('Erro', errorMessage(e, 'Falha ao gerar novo token')) }
  finally { testing.value = false }
}
const formatResponse = (r: string) => { try { return JSON.stringify(JSON.parse(r), null, 2) } catch { return r } }

const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | null = null
const writeClipboard = async (t: string) => {
  if (navigator.clipboard) await navigator.clipboard.writeText(t)
  else { const ta = document.createElement('textarea'); ta.value = t; document.body.appendChild(ta); ta.select(); document.execCommand('copy'); document.body.removeChild(ta) }
}
const copy = async (text: string, tag: string) => {
  try { await writeClipboard(text); miniCopied.value = tag; setTimeout(() => { miniCopied.value = '' }, 1500) } catch { toast.error('Erro', 'Não foi possível copiar') }
}
const copySnippet = async () => {
  if (!snippet.value) return
  try {
    await writeClipboard(snippet.value); copied.value = true; toast.success('Copiado!', 'Script copiado para a área de transferência')
    if (copiedTimer) clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => { copied.value = false }, 2500)
  } catch { toast.error('Erro', 'Selecione e copie manualmente.') }
}

watch(() => props.modelValue, open => {
  if (!open) { copied.value = false; miniCopied.value = ''; tab.value = 'script' }
})
</script>

<style scoped>
.install-content { display: flex; flex-direction: column; }
.tabs { display: flex; border-bottom: 1px solid var(--border); margin-bottom: 1.5rem; overflow-x: auto; white-space: nowrap; }
.tab { padding: .625rem 1.125rem; background: transparent; border: none; color: var(--text-2); font-size: var(--fs-md); font-weight: var(--fw-medium); cursor: pointer; position: relative; display: inline-flex; align-items: center; gap: .375rem; flex-shrink: 0; }
.tab:hover { color: var(--text-1); } .tab.active { color: var(--brand); }
.tab.active::after { content: ''; position: absolute; bottom: -1px; left: 0; right: 0; height: 2px; background: var(--brand); }
.tab-dot { width: 7px; height: 7px; border-radius: 50%; } .tab-dot.ok { background: var(--success); } .tab-dot.pending { background: var(--warning); }
.tab-pane, .section { display: flex; flex-direction: column; gap: 1.25rem; } .section { gap: .625rem; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: .5rem; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); display: inline-flex; align-items: center; gap: .5rem; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); line-height: 1.5; }
.section-hint code, .result-meta code, .result-pixel, .copy-val, .code-block, .event-tag { font-family: ui-monospace, monospace; }
.section-hint code { font-size: var(--fs-base); background: var(--surface-soft); padding: .0625rem .3125rem; border-radius: 4px; }
.badge-pill { font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: .05em; padding: .125rem .5rem; border-radius: 9999px; background: var(--brand-soft); color: var(--brand); } .badge-pill.good { background: var(--success-soft); color: var(--success); }
.copy-row { display: flex; align-items: center; gap: .5rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 8px; padding: .5rem .75rem; }
.copy-val { font-size: var(--fs-base); color: var(--brand); flex: 1; word-break: break-all; letter-spacing: .04em; }
.copy-mini { background: transparent; border: 1px solid var(--border); color: var(--text-2); padding: .1875rem .5rem; border-radius: 5px; cursor: pointer; font-size: var(--fs-sm); }
.code-block { background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 10px; padding: 1rem; font-size: var(--fs-base); color: var(--text-1); overflow-x: auto; line-height: 1.6; white-space: pre; } .code-block.small { font-size: var(--fs-sm); }
.domains-list { display: flex; flex-wrap: wrap; gap: .375rem; } .domain-pill { font-size: var(--fs-sm); font-weight: var(--fw-medium); background: var(--success-soft); color: var(--success); padding: .1875rem .625rem; border-radius: 9999px; } .hint-warn { font-size: var(--fs-base); color: var(--warning); font-style: italic; }
.events-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(170px, 1fr)); gap: .375rem; }
.event-item { display: flex; align-items: center; gap: .5rem; padding: .5rem .75rem; border: 1px solid var(--border-soft); border-radius: 8px; background: var(--surface-soft-2); } .event-item.on { background: var(--success-soft); border-color: var(--success); }
.event-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-3); flex-shrink: 0; } .event-item.on .event-dot { background: var(--success); }
.event-name { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-1); } .event-item:not(.on) .event-name, .event-tag { color: var(--text-3); } .event-tag { font-size: var(--fs-xs); }
.ch-tabs { display: flex; gap: .25rem; background: var(--surface-soft); border: 1px solid var(--border-soft); border-radius: 8px; padding: .25rem; width: fit-content; }
.ch-tab { padding: .3125rem .875rem; background: transparent; border: none; border-radius: 6px; color: var(--text-2); font-size: var(--fs-base); font-weight: var(--fw-medium); cursor: pointer; } .ch-tab.active { background: var(--surface-2); color: var(--brand); } .ch-tab:disabled { opacity: .4; cursor: not-allowed; }
.test-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: .75rem; } .test-footer { display: flex; justify-content: flex-end; }
.test-results { display: flex; flex-direction: column; gap: .5rem; margin-top: .75rem; }
.test-result { padding: .75rem 1rem; border: 1px solid var(--border-soft); border-radius: 10px; background: var(--surface-2); } .test-result.ok { border-left: 4px solid var(--success); } .test-result.fail { border-left: 4px solid var(--danger); }
.result-row { display: flex; align-items: center; gap: .5rem; margin-bottom: .25rem; flex-wrap: wrap; } .result-pixel { font-size: var(--fs-base); flex: 1; min-width: 120px; } .result-error { font-size: var(--fs-base); color: var(--danger); }
.result-detail, .result-detail summary, .result-meta { font-size: var(--fs-sm); color: var(--text-3); margin-top: .5rem; }
</style>
