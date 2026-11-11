<template>
  <BaseModal
    :model-value="modelValue"
    :title="isWebhookOnly ? `Webhook ${providerLabel} — ${instance?.name || ''}` : `Conectar ${instance?.name || 'WhatsApp'}`"
    :description="isWebhookOnly
      ? `Copie a URL abaixo e cole na configuração de Webhook da sua instância na ${providerLabel}.`
      : 'Aponte a câmera do WhatsApp para o QR Code abaixo ou envie o link externo para seu cliente.'"
    size="sm"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="isWebhookOnly" class="qr-pane">
      <div class="wh-card">
        <span class="wh-label">URL exclusiva do Webhook:</span>
        <code class="wh-code">{{ instance?.webhook_url }}</code>
        <BaseButton variant="primary" size="sm" @click="copyWebhookUrl">
          <Check v-if="whCopied" :size="14" />
          <Link2 v-else :size="14" />
          <span>{{ whCopied ? 'URL copiada!' : 'Copiar URL do Webhook' }}</span>
        </BaseButton>
      </div>

      <div class="listening-pill">
        <span class="pulse-dot" />
        <span>Aguardando eventos da {{ providerLabel }}...</span>
      </div>

      <div class="steps-box">
        <span class="steps-title">Como configurar na {{ providerLabel }}:</span>
        <ol class="steps-list">
          <li><span class="step-num">1</span><span>Cole a <strong>URL do Webhook</strong> acima na sua instância</span></li>
          <li><span class="step-num">2</span><span>Em Eventos, marque <strong><code>messages</code></strong> e <strong><code>connection</code></strong></span></li>
          <li><span class="step-num">3</span><span>Em Excluir Mensagens, marque <strong><code>wasSentByApi</code></strong> (Enviadas por API) e <strong><code>isGroupYes</code></strong> (Grupos)</span></li>
        </ol>
      </div>
    </div>

    <div v-else class="qr-pane">
      <div v-if="loadingQr && !qrDataUrl" class="qr-placeholder">
        <QrCode :size="38" class="qr-icon-pulse" />
        <div class="spinner" />
        <span>Gerando QR Code...</span>
      </div>

      <div v-else-if="qrDataUrl" class="qr-frame">
        <img :src="qrDataUrl" alt="QR Code WhatsApp" class="qr-img">
      </div>

      <div v-else class="qr-placeholder">
        <QrCode :size="36" class="qr-icon-muted" />
        <span>Não foi possível exibir o QR Code agora.</span>
        <BaseButton variant="outline" size="sm" :loading="loadingQr" @click="loadQr">
          Tentar novamente
        </BaseButton>
      </div>

      <div class="listening-pill">
        <span class="pulse-dot" />
        <span>Aguardando leitura do QR Code pelo WhatsApp...</span>
      </div>

      <div class="steps-box">
        <span class="steps-title">Como conectar:</span>
        <ol class="steps-list">
          <li><span class="step-num">1</span><span>Abra o <strong>WhatsApp</strong> no celular</span></li>
          <li><span class="step-num">2</span><span>Toque em <strong>Mais opções</strong> ou <strong>Configurações</strong></span></li>
          <li><span class="step-num">3</span><span>Selecione <strong>Aparelhos conectados → Conectar aparelho</strong></span></li>
          <li><span class="step-num">4</span><span>Aponte a câmera para este QR Code</span></li>
        </ol>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="emit('update:modelValue', false)">
        Fechar
      </BaseButton>
      <BaseButton v-if="!isWebhookOnly && instance?.share_token" variant="outline" @click="copyShareLink">
        <Link2 :size="14" />
        <span>Link p/ Cliente</span>
      </BaseButton>
      <BaseButton v-if="!isWebhookOnly" variant="outline" :loading="loadingQr" @click="loadQr">
        <RefreshCw :size="14" />
        <span>Atualizar QR</span>
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { QrCode, RefreshCw, Link2, Check } from 'lucide-vue-next'
import type { WhatsAppInstance } from '#shared/types'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { useWhatsAppStore } from '~/stores/useWhatsAppStore'

const props = defineProps<{
  modelValue: boolean
  instance: WhatsAppInstance | null
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  connected: []
}>()

const wa = useWhatsApp()
const waStore = useWhatsAppStore()
const liveStore = useLiveEventsStore()
const toast = useToast()

const qrDataUrl = ref('')
const loadingQr = ref(false)
const whCopied = ref(false)

const isWebhookOnly = computed(() => {
  const p = props.instance?.provider
  return p === 'api_premium' || p === 'uazapi'
})

const providerLabel = computed(() => (props.instance?.provider === 'api_premium' ? 'API Premium' : 'Uazapi'))

const normalizeQrSrc = (raw?: string): string => {
  if (!raw) return ''
  return raw.startsWith('data:image') ? raw : `data:image/png;base64,${raw}`
}

const loadQr = async () => {
  if (!props.instance || isWebhookOnly.value) return
  loadingQr.value = true
  try {
    const res = await wa.getQrCode(props.instance.id)
    if (res.connected) {
      waStore.updateInstanceStatus(props.instance.id, 'OPEN')
      toast.success('WhatsApp conectado', `"${props.instance.name}" já está online.`)
      emit('connected')
      emit('update:modelValue', false)
      return
    }
    qrDataUrl.value = normalizeQrSrc(res.qrcode)
  } catch (e: any) {
    toast.error('Erro ao obter QR Code', e.message || 'Tente novamente em instantes.')
  } finally {
    loadingQr.value = false
  }
}

const copyShareLink = async () => {
  if (!props.instance?.share_token || typeof window === 'undefined') return
  const url = `${window.location.origin}/conectar-whatsapp/${props.instance.share_token}`
  try {
    await navigator.clipboard.writeText(url)
    toast.success('Link copiado!', 'Envie este link para seu cliente conectar o WhatsApp.')
  } catch {
    toast.error('Erro ao copiar', url)
  }
}

const copyWebhookUrl = async () => {
  if (!props.instance?.webhook_url || typeof window === 'undefined') return
  try {
    await navigator.clipboard.writeText(props.instance.webhook_url)
    whCopied.value = true
    setTimeout(() => { whCopied.value = false }, 2200)
    toast.success('Webhook copiado!', `Cole no painel da ${providerLabel.value}.`)
  } catch {
    toast.error('Erro ao copiar', props.instance.webhook_url)
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      qrDataUrl.value = ''
      whCopied.value = false
      if (!isWebhookOnly.value) {
        void loadQr()
      }
    }
  },
)

let offQr: (() => void) | null = null
let offStatus: (() => void) | null = null

onMounted(() => {
  offQr = liveStore.onCustomEvent('whatsapp_qr', (data: any) => {
    if (props.modelValue && props.instance && data?.instance_id === props.instance.id && data.qrcode) {
      qrDataUrl.value = normalizeQrSrc(data.qrcode)
    }
  })
  offStatus = liveStore.onCustomEvent('whatsapp_status', (data: any) => {
    if (!props.instance || data?.instance_id !== props.instance.id) return
    waStore.updateInstanceStatus(props.instance.id, data.status, data.owner_jid, data.profile_name, data.profile_pic_url)
    if (props.modelValue && data.status === 'OPEN') {
      toast.success('WhatsApp conectado!', 'Aparelho sincronizado com sucesso.')
      emit('connected')
      emit('update:modelValue', false)
    }
  })
})

onUnmounted(() => {
  offQr?.()
  offStatus?.()
})
</script>

<style scoped>
.qr-pane { display: flex; flex-direction: column; align-items: center; gap: 1rem; text-align: center; }
.qr-frame { padding: 0.875rem; background: var(--surface-qr); border: 2px solid var(--brand-border); border-radius: 16px; display: inline-flex; }
.qr-img { width: 224px; height: 224px; display: block; object-fit: contain; }
.qr-placeholder { width: 252px; height: 252px; border-radius: 16px; background: var(--surface-deep); border: 1px dashed var(--border); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.75rem; color: var(--text-3); font-size: var(--fs-sm); padding: 1.25rem; }
.wh-card { width: 100%; padding: 1rem; border-radius: 12px; background: var(--surface-deep); border: 1px solid var(--border); display: flex; flex-direction: column; align-items: stretch; gap: 0.65rem; text-align: left; }
.wh-label { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-2); }
.wh-code { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--brand); background: var(--surface-2); border: 1px solid var(--border-soft); border-radius: 8px; padding: 0.6rem 0.75rem; word-break: break-all; line-height: 1.4; }
.qr-icon-pulse { color: var(--brand); opacity: 0.65; }
.qr-icon-muted { color: var(--text-3); opacity: 0.5; }
.listening-pill { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.35rem 0.85rem; border-radius: 999px; background: var(--surface-deep); border: 1px solid var(--border-soft); color: var(--warning); font-size: var(--fs-xs); font-weight: var(--fw-medium); }
.pulse-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--warning); flex-shrink: 0; animation: pulse 1.8s ease-in-out infinite; }
.steps-box { width: 100%; padding: 0.95rem 1.1rem; border-radius: 12px; background: var(--surface-deep); border: 1px solid var(--border-soft); text-align: left; display: flex; flex-direction: column; gap: 0.6rem; }
.steps-title { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-1); text-transform: uppercase; letter-spacing: 0.04em; }
.steps-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 0.5rem; font-size: var(--fs-xs); color: var(--text-2); line-height: 1.4; }
.steps-list li { display: flex; align-items: center; gap: 0.6rem; }
.steps-list code { font-family: var(--font-mono); font-size: var(--fs-2xs); color: var(--text-1); background: var(--surface-2); padding: 0.1rem 0.3rem; border-radius: 4px; }
.step-num { width: 20px; height: 20px; border-radius: 50%; background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand); font-size: var(--fs-2xs); font-weight: var(--fw-bold); display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; }
.spinner { width: 1.5rem; height: 1.5rem; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@keyframes pulse { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.45; transform: scale(0.85); } }
</style>
