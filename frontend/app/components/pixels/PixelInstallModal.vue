<template>
  <BaseModal
    :model-value="modelValue"
    title="Instalar Pixel"
    :description="`Configure o DNS, copie o script e cole no site para começar a rastrear conversões com \&quot;${pixel?.name || ''}\&quot;.`"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="pixel" class="install-content">
      <!-- ETAPA 1: DNS first-party -->
      <section class="check-section">
        <div class="section-header">
          <h4 class="section-title">
            1. Configure o DNS no domínio do cliente
            <span v-if="hasTrackingHost" class="step-badge">opcional → recomendado</span>
            <span v-else class="step-badge step-badge-warn">não configurado</span>
          </h4>
        </div>

        <p class="section-hint">
          Para usar cookies first-party (essencial pra contornar adblockers e iOS), o cliente
          precisa criar um registro CNAME no DNS apontando um subdomínio dele para o servidor
          TrackFind.
        </p>

        <div v-if="!hasTrackingHost" class="dns-empty">
          <p>
            Este pixel ainda não tem um subdomínio configurado. Edite o pixel e preencha o
            campo <strong>"Subdomínio first-party"</strong> com algo como
            <code>tracking.{{ exampleDomain }}</code>.
          </p>
        </div>

        <template v-else>
          <div class="dns-record">
            <div class="dns-row">
              <span class="dns-label">Tipo</span>
              <code class="dns-value">CNAME</code>
            </div>
            <div class="dns-row">
              <span class="dns-label">Nome</span>
              <code class="dns-value">{{ pixel.tracking_host }}</code>
              <button class="copy-mini" @click="copy(pixel.tracking_host, 'host')" title="Copiar">
                {{ miniCopied === 'host' ? '✓' : '📋' }}
              </button>
            </div>
            <div class="dns-row">
              <span class="dns-label">Valor</span>
              <code class="dns-value">{{ dnsTarget || 'Defina TRACKING_TARGET_HOST no servidor' }}</code>
              <button v-if="dnsTarget" class="copy-mini" @click="copy(dnsTarget, 'target')" title="Copiar">
                {{ miniCopied === 'target' ? '✓' : '📋' }}
              </button>
            </div>
            <div class="dns-row">
              <span class="dns-label">TTL</span>
              <code class="dns-value">3600 (ou padrão)</code>
            </div>
          </div>

          <div class="dns-actions">
            <BaseButton variant="outline" size="sm" :loading="verifying" @click="onVerifyDNS">
              <template #iconLeft>
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
                </svg>
              </template>
              Verificar DNS
            </BaseButton>

            <span v-if="dnsResult" :class="['dns-result', `dns-result-${dnsResult.status}`]">
              <span class="dns-status-dot" />
              {{ dnsResult.hint }}
            </span>
          </div>

          <details v-if="dnsResult" class="dns-details">
            <summary>Detalhes técnicos</summary>
            <dl>
              <dt>Host consultado</dt>
              <dd><code>{{ dnsResult.host }}</code></dd>
              <dt>CNAME atual</dt>
              <dd><code>{{ dnsResult.cname || '—' }}</code></dd>
              <dt>IPs (A records)</dt>
              <dd><code>{{ dnsResult.a?.join(', ') || '—' }}</code></dd>
              <dt>Target esperado</dt>
              <dd><code>{{ dnsResult.target }}</code></dd>
            </dl>
          </details>
        </template>
      </section>

      <!-- ETAPA 2: Snippet -->
      <section class="snippet-section">
        <div class="section-header">
          <h4 class="section-title">
            2. Cole o script no &lt;head&gt; do site
            <span v-if="hasTrackingHost" class="step-badge step-badge-good">first-party</span>
          </h4>
          <BaseButton variant="outline" size="sm" @click="copySnippet">
            <template #iconLeft>
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                <path d="M8 2a1 1 0 000 2h2a1 1 0 100-2H8z" />
                <path d="M3 5a2 2 0 012-2 3 3 0 003 3h2a3 3 0 003-3 2 2 0 012 2v6h-4.586l1.293-1.293a1 1 0 00-1.414-1.414l-3 3a1 1 0 000 1.414l3 3a1 1 0 001.414-1.414L10.414 13H15v3a2 2 0 01-2 2H5a2 2 0 01-2-2V5z" />
                <path d="M15 11h2a1 1 0 110 2h-2v-2z" />
              </svg>
            </template>
            {{ copied ? 'Copiado!' : 'Copiar' }}
          </BaseButton>
        </div>

        <pre class="snippet-code"><code>{{ snippet }}</code></pre>

        <p class="section-hint">
          Funciona em qualquer plataforma (WordPress, Shopify, código próprio, GTM).
          Cole antes do fechamento da <code>&lt;/head&gt;</code> ou <code>&lt;/body&gt;</code>.
        </p>
      </section>

      <!-- ETAPA 3: Domínios -->
      <section class="check-section">
        <h4 class="section-title">3. Domínios autorizados</h4>
        <p class="section-hint">
          O backend só aceita eventos vindos dos domínios listados. Adicione subdomínios também
          (ex: <code>www.exemplo.com</code>).
        </p>
        <div class="domains-list">
          <span v-for="d in pixel.domains" :key="d" class="domain-pill">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
            </svg>
            {{ d }}
          </span>
          <span v-if="!pixel.domains?.length" class="muted">
            Nenhum domínio cadastrado — edite o pixel e adicione.
          </span>
        </div>
      </section>

      <!-- ETAPA 4: Eventos automáticos -->
      <section class="check-section">
        <div class="section-header">
          <h4 class="section-title">4. Eventos automáticos do tracker</h4>
          <BaseButton variant="outline" size="sm" @click="$emit('edit-config', pixel)">
            <template #iconLeft>
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
              </svg>
            </template>
            Editar configuração
          </BaseButton>
        </div>
        <p class="section-hint">
          Comportamento que o tracker dispara automaticamente. Edição visual na aba
          <strong>Tracker Config</strong>.
        </p>
        <div class="events-grid">
          <div
            v-for="event in autoEvents"
            :key="event.key"
            class="event-item"
            :class="{ enabled: event.enabled }"
          >
            <div class="event-icon">
              <svg
                v-if="event.enabled"
                xmlns="http://www.w3.org/2000/svg"
                class="h-4 w-4"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
              </svg>
              <svg
                v-else
                xmlns="http://www.w3.org/2000/svg"
                class="h-4 w-4"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
              </svg>
            </div>
            <div>
              <p class="event-name">{{ event.label }}</p>
              <p class="event-key">{{ event.tag }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- ETAPA 5: Enviar evento de teste -->
      <section class="check-section">
        <h4 class="section-title">5. Enviar evento de teste</h4>
        <p class="section-hint">
          Dispara um evento sincronamente para a Meta CAPI usando o
          <code>test_event_code</code> do pixel. Útil para validar a integração antes do
          tráfego real.
        </p>

        <div class="test-form">
          <BaseDropdown
            v-model="testForm.event_name"
            label="Evento"
            :options="testEventOptions"
          />
          <BaseInput
            v-model="testForm.email"
            label="E-mail (opcional)"
            placeholder="teste@exemplo.com"
          />
          <BaseInput
            v-model="testForm.phone"
            label="Telefone (opcional)"
            placeholder="+5511999999999"
          />
          <BaseInput
            v-model="testForm.value"
            label="Valor (opcional)"
            type="number"
            placeholder="99.90"
          />
        </div>

        <div class="test-actions">
          <BaseButton variant="primary" :loading="testing" @click="onSendTestEvent">
            <template #iconLeft>
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M10.894 2.553a1 1 0 00-1.788 0l-7 14a1 1 0 001.169 1.409l5-1.429A1 1 0 009 15.571V11a1 1 0 112 0v4.571a1 1 0 00.725.962l5 1.428a1 1 0 001.17-1.408l-7-14z" clip-rule="evenodd" />
              </svg>
            </template>
            Enviar evento de teste
          </BaseButton>
        </div>

        <div v-if="testResults.length" class="test-results">
          <div
            v-for="result in testResults"
            :key="result.pixel_id"
            class="test-result"
            :class="{ success: result.status_code === 200, error: result.status_code !== 200 || result.error }"
          >
            <div class="result-header">
              <code class="result-pixel">Pixel {{ result.pixel_id }}</code>
              <BaseBadge :variant="result.status_code === 200 ? 'success' : 'danger'">
                HTTP {{ result.status_code || 'erro' }}
              </BaseBadge>
            </div>
            <p v-if="result.error" class="result-error">{{ result.error }}</p>
            <details v-if="result.response" class="result-body">
              <summary>Resposta da Meta</summary>
              <pre><code>{{ formatResponse(result.response) }}</code></pre>
            </details>
          </div>
          <p v-if="lastEventID" class="result-meta">
            Event ID disparado: <code>{{ lastEventID }}</code> — verifique no
            <strong>Events Manager → Test Events</strong>.
          </p>
        </div>
      </section>

      <!-- ETAPA 6: Disparar eventos manualmente -->
      <section class="check-section">
        <h4 class="section-title">6. Disparar eventos do código do site</h4>
        <p class="section-hint">
          Use a função global <code>tf()</code> para disparar eventos customizados:
        </p>
        <pre class="snippet-code small"><code>{{ manualSnippet }}</code></pre>
      </section>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">
        Fechar
      </BaseButton>
      <BaseButton variant="primary" @click="copySnippet">
        {{ copied ? 'Copiado!' : 'Copiar Script' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import BaseDropdown from '~/components/BaseDropdown.vue'
import { buildPixelSnippet } from '~/utils/apiOrigin'
import type { DNSVerifyResponse, Pixel, TestEventInput, TestEventPixelResult } from '#shared/types'

interface Props {
  modelValue: boolean
  pixel: Pixel | null
}

const props = defineProps<Props>()

defineEmits<{
  'update:modelValue': [value: boolean]
  'edit-config': [pixel: Pixel]
}>()

const config = useRuntimeConfig()
const toast = useToast()
const pixelsApi = usePixels()

const hasTrackingHost = computed(() => !!props.pixel?.tracking_host)

const exampleDomain = computed(() => {
  const first = props.pixel?.domains?.[0]
  return first || 'seu-dominio.com'
})

const snippet = computed(() =>
  props.pixel
    ? buildPixelSnippet(
        config.public.apiUrl as string,
        props.pixel.name,
        props.pixel.tracking_host
      )
    : ''
)

const manualSnippet = computed(() => {
  if (!props.pixel) return ''
  return [
    "// Lead manual (ex: clique no botão de WhatsApp)",
    "tf('send', { eventType: 'Lead' });",
    '',
    "// Compra com valor",
    "tf('send', {",
    "  eventType: 'Purchase',",
    "  value: 99.90,",
    "  currency: 'BRL',",
    "  contents: [{ id: 'SKU-001', quantity: 1 }],",
    "});",
  ].join('\n')
})

const autoEvents = computed(() => {
  const cfg = props.pixel?.tracker_config
  if (!cfg) return []
  return [
    {
      key: 'auto_page_view',
      label: 'PageView automático',
      tag: 'PageView',
      enabled: !!cfg.auto_page_view,
    },
    {
      key: 'auto_checkout_buttons',
      label: 'Botões de checkout',
      tag: 'InitiateCheckout',
      enabled: !!cfg.auto_checkout_buttons,
    },
    {
      key: 'auto_form_scraping',
      label: 'Captura de formulário',
      tag: 'PII',
      enabled: !!cfg.auto_form_scraping,
    },
    {
      key: 'auto_form_lead',
      label: 'Lead em formulário',
      tag: 'Lead',
      enabled: !!cfg.auto_form_lead,
    },
    {
      key: 'whatsapp_protocol',
      label: 'Protocolo WhatsApp',
      tag: 'Lead',
      enabled: !!cfg.whatsapp_protocol,
    },
    {
      key: 'send_scrolls',
      label: 'Scroll tracking',
      tag: 'Scroll_*',
      enabled: !!cfg.send_scrolls,
    },
    {
      key: 'send_timer_1min',
      label: 'Timer 1 minuto',
      tag: 'Timer_1min',
      enabled: !!cfg.send_timer_1min,
    },
    {
      key: 'send_view_video',
      label: 'View Video',
      tag: 'ViewVideo_*',
      enabled: !!cfg.send_view_video,
    },
  ]
})

// DNS verification
const dnsResult = ref<DNSVerifyResponse | null>(null)
const dnsTarget = ref<string>('')
const verifying = ref(false)

const onVerifyDNS = async () => {
  if (!props.pixel) return
  verifying.value = true
  try {
    const res = await pixelsApi.verifyDNS(props.pixel.id)
    dnsResult.value = res
    if (res.target && !res.target.startsWith('(')) {
      dnsTarget.value = res.target
    }

    if (res.status === 'ok') {
      toast.success('DNS OK', 'O subdomínio está apontando corretamente')
    } else if (res.status === 'wrong_target') {
      toast.warning('DNS divergente', 'O domínio resolve mas não para o servidor TrackFind')
    } else if (res.status === 'not_found') {
      toast.warning('DNS não encontrado', 'Aguarde a propagação ou confira o registro')
    }
  } catch (error: any) {
    toast.error('Erro', error.message || 'Não foi possível verificar o DNS')
  } finally {
    verifying.value = false
  }
}

// === Test event ===
const testEventOptions = [
  { value: 'Lead', label: 'Lead' },
  { value: 'PageView', label: 'PageView' },
  { value: 'ViewContent', label: 'ViewContent' },
  { value: 'AddToCart', label: 'AddToCart' },
  { value: 'InitiateCheckout', label: 'InitiateCheckout' },
  { value: 'Purchase', label: 'Purchase' },
  { value: 'CompleteRegistration', label: 'CompleteRegistration' },
  { value: 'Contact', label: 'Contact' },
]

const testForm = reactive<{
  event_name: string
  email: string
  phone: string
  value: string
}>({
  event_name: 'Lead',
  email: '',
  phone: '',
  value: '',
})

const testing = ref(false)
const testResults = ref<TestEventPixelResult[]>([])
const lastEventID = ref<string>('')

const onSendTestEvent = async () => {
  if (!props.pixel) return
  testing.value = true
  testResults.value = []
  lastEventID.value = ''
  try {
    const input: TestEventInput = {
      event_name: testForm.event_name || 'Lead',
    }
    if (testForm.email.trim()) input.email = testForm.email.trim()
    if (testForm.phone.trim()) input.phone = testForm.phone.trim()
    if (testForm.value.trim()) {
      const v = parseFloat(testForm.value.replace(',', '.'))
      if (!Number.isNaN(v)) input.value = v
    }

    const response = await pixelsApi.sendTestEvent(props.pixel.id, input)
    testResults.value = response.results
    lastEventID.value = response.event_id

    const allOk = response.results.every((r) => r.status_code === 200 && !r.error)
    if (allOk) {
      toast.success('Evento enviado!', 'Verifique no Events Manager → Test Events')
    } else {
      toast.warning('Evento enviado com falhas', 'Veja os detalhes abaixo')
    }
  } catch (error: any) {
    toast.error('Erro ao enviar', error.message || 'Falha desconhecida')
  } finally {
    testing.value = false
  }
}

const formatResponse = (raw: string): string => {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

// Copy
const copied = ref(false)
const miniCopied = ref<string>('')
let copiedTimer: ReturnType<typeof setTimeout> | null = null

const writeClipboard = async (text: string) => {
  if (navigator.clipboard) {
    await navigator.clipboard.writeText(text)
  } else {
    const ta = document.createElement('textarea')
    ta.value = text
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }
}

const copy = async (text: string, tag: string) => {
  try {
    await writeClipboard(text)
    miniCopied.value = tag
    setTimeout(() => {
      miniCopied.value = ''
    }, 1500)
  } catch {
    toast.error('Erro', 'Não foi possível copiar')
  }
}

const copySnippet = async () => {
  if (!snippet.value) return
  try {
    await writeClipboard(snippet.value)
    copied.value = true
    toast.success('Copiado!', 'Script copiado para a área de transferência')
    if (copiedTimer) clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => {
      copied.value = false
    }, 2500)
  } catch {
    toast.error('Erro', 'Não foi possível copiar. Selecione e copie manualmente.')
  }
}

// Reset estado quando o modal abre/fecha
watch(
  () => props.modelValue,
  (open) => {
    if (!open) {
      copied.value = false
      miniCopied.value = ''
      dnsResult.value = null
      if (copiedTimer) {
        clearTimeout(copiedTimer)
        copiedTimer = null
      }
    } else {
      // Tenta pré-carregar o target esperado quando abre o modal
      if (props.pixel?.tracking_host && !dnsTarget.value) {
        // Faz um verify silencioso pra mostrar o target
        pixelsApi
          .verifyDNS(props.pixel.id)
          .then((res) => {
            if (res.target && !res.target.startsWith('(')) {
              dnsTarget.value = res.target
            }
          })
          .catch(() => {})
      }
    }
  }
)
</script>

<style scoped>
.install-content {
  display: flex;
  flex-direction: column;
  gap: 1.75rem;
}

.snippet-section,
.check-section {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.section-title {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text-strong);
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.step-badge {
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--brand-soft);
  color: var(--brand);
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
}

.step-badge-warn {
  background: var(--warning-soft);
  color: var(--warning);
}

.step-badge-good {
  background: var(--success-soft);
  color: var(--success);
}

.section-hint {
  font-size: 0.8125rem;
  color: var(--text-2);
  line-height: 1.5;
}

.section-hint code,
.dns-empty code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8125rem;
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
}

.snippet-code {
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  padding: 1rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8125rem;
  color: var(--text-1);
  overflow-x: auto;
  line-height: 1.6;
  white-space: pre;
  word-wrap: normal;
}

.snippet-code.small {
  font-size: 0.75rem;
}

.dns-empty {
  background: var(--warning-soft);
  border: 1px solid var(--warning);
  border-radius: 10px;
  padding: 0.875rem 1rem;
  font-size: 0.875rem;
  color: var(--text-1);
  line-height: 1.5;
}

.dns-record {
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  padding: 0.75rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.dns-row {
  display: grid;
  grid-template-columns: 80px 1fr auto;
  align-items: center;
  gap: 0.75rem;
  font-size: 0.875rem;
}

.dns-label {
  color: var(--text-3);
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 600;
}

.dns-value {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
  padding: 0.25rem 0.625rem;
  border-radius: 6px;
  color: var(--text-1);
  word-break: break-all;
}

.copy-mini {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-2);
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.75rem;
  transition: all 0.15s;
}

.copy-mini:hover {
  background: var(--surface-soft);
  color: var(--text-1);
}

.dns-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.dns-result {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  color: var(--text-2);
}

.dns-status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-3);
}

.dns-result-ok {
  color: var(--success);
}

.dns-result-ok .dns-status-dot {
  background: var(--success);
}

.dns-result-wrong_target {
  color: var(--warning);
}

.dns-result-wrong_target .dns-status-dot {
  background: var(--warning);
}

.dns-result-not_found,
.dns-result-no_host {
  color: var(--danger);
}

.dns-result-not_found .dns-status-dot,
.dns-result-no_host .dns-status-dot {
  background: var(--danger);
}

.dns-details {
  margin-top: 0.5rem;
  font-size: 0.8125rem;
  color: var(--text-2);
}

.dns-details summary {
  cursor: pointer;
  user-select: none;
  color: var(--text-3);
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 600;
  padding: 0.25rem 0;
}

.dns-details dl {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: 0.375rem 1rem;
  margin-top: 0.5rem;
  padding: 0.75rem;
  background: var(--surface-soft);
  border-radius: 8px;
}

.dns-details dt {
  color: var(--text-3);
  font-weight: 500;
}

.dns-details dd code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  word-break: break-all;
}

.domains-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.domain-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.75rem;
  background: var(--success-soft);
  color: var(--success);
  border-radius: 9999px;
  font-size: 0.8125rem;
  font-weight: 500;
}

.muted {
  font-size: 0.8125rem;
  color: var(--warning);
  font-style: italic;
}

.events-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 0.5rem;
}

.event-item {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.625rem 0.875rem;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  background: var(--surface-soft-2);
}

.event-item.enabled {
  background: var(--success-soft);
  border-color: var(--success);
}

.event-icon {
  flex-shrink: 0;
  color: var(--text-3);
}

.event-item.enabled .event-icon {
  color: var(--success);
}

.event-name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-1);
}

.event-item:not(.enabled) .event-name {
  color: var(--text-2);
}

.event-key {
  font-size: 0.6875rem;
  color: var(--text-3);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  margin-top: 0.125rem;
}

/* Test event section */
.test-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.test-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.5rem;
}

.test-results {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 1rem;
}

.test-result {
  padding: 0.875rem 1rem;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  background: var(--surface-2);
}

.test-result.success {
  border-left: 4px solid var(--success);
}

.test-result.error {
  border-left: 4px solid var(--danger);
}

.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.375rem;
}

.result-pixel {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8125rem;
  color: var(--text-1);
}

.result-error {
  font-size: 0.8125rem;
  color: var(--danger);
  margin-top: 0.25rem;
}

.result-body {
  margin-top: 0.5rem;
  font-size: 0.75rem;
}

.result-body summary {
  cursor: pointer;
  user-select: none;
  color: var(--text-3);
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 600;
}

.result-body pre {
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  border-radius: 6px;
  padding: 0.625rem;
  margin-top: 0.375rem;
  overflow-x: auto;
  max-height: 200px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.6875rem;
  color: var(--text-1);
  line-height: 1.5;
}

.result-meta {
  font-size: 0.8125rem;
  color: var(--text-2);
  margin-top: 0.5rem;
}

.result-meta code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
  font-size: 0.75rem;
}
</style>
