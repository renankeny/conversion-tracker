<template>
  <div class="connect-pane">
    <div v-if="isConnected" class="connected-box">
      <CheckCircle2 :size="28" class="ok-icon" />
      <h4 class="ok-title">WhatsApp conectado com sucesso!</h4>
      <p class="ok-desc">O aparelho <strong>{{ instance.name }}</strong> está sincronizado e pronto para rastrear conversas e gatilhos.</p>
    </div>

    <template v-else-if="isWebhookOnly">
      <div class="wh-card">
        <span class="wh-label">URL exclusiva do Webhook ({{ providerLabel }}):</span>
        <code class="wh-code">{{ instance.webhook_url }}</code>
        <BaseButton variant="primary" size="sm" @click="copyWebhook">
          <Check v-if="whCopied" :size="14" />
          <Link2 v-else :size="14" />
          <span>{{ whCopied ? 'URL copiada!' : 'Copiar URL do Webhook' }}</span>
        </BaseButton>
      </div>
      <p class="connect-hint">
        Cole esta URL na sua instância da {{ providerLabel }}, marque os eventos <code>messages</code> e <code>connection</code> e exclua <code>wasSentByApi</code> e <code>isGroupYes</code>.
      </p>
    </template>

    <template v-else>
      <div v-if="loadingQr && !qrDataUrl" class="qr-box">
        <div class="spinner" />
        <span>Gerando QR Code...</span>
      </div>
      <div v-else-if="qrDataUrl" class="qr-frame">
        <img :src="qrDataUrl" alt="QR Code WhatsApp" class="qr-img">
      </div>
      <div v-else class="qr-box">
        <QrCode :size="30" class="muted-icon" />
        <span>Não foi possível gerar o QR Code agora.</span>
        <BaseButton variant="outline" size="sm" :loading="loadingQr" @click="loadQr">
          Tentar novamente
        </BaseButton>
      </div>

      <div class="qr-actions">
        <BaseButton variant="outline" size="sm" :loading="loadingQr" @click="loadQr">
          <RefreshCw :size="14" />
          <span>Atualizar QR</span>
        </BaseButton>
        <BaseButton v-if="instance.share_token" variant="outline" size="sm" @click="copyShareLink">
          <Check v-if="linkCopied" :size="14" />
          <Link2 v-else :size="14" />
          <span>{{ linkCopied ? 'Link copiado!' : 'Copiar link p/ cliente conectar' }}</span>
        </BaseButton>
      </div>

      <p class="connect-hint">
        Não está com o celular do cliente agora? Copie o link acima para ele escanear no celular dele ou clique em concluir para conectar depois.
      </p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Check, CheckCircle2, Link2, QrCode, RefreshCw } from 'lucide-vue-next'
import type { WhatsAppInstance } from '#shared/types'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { useWhatsAppStore } from '~/stores/useWhatsAppStore'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{ instance: WhatsAppInstance }>()
const emit = defineEmits<{ connected: [] }>()

const wa = useWhatsApp()
const waStore = useWhatsAppStore()
const liveStore = useLiveEventsStore()
const toast = useToast()

const qrDataUrl = ref('')
const loadingQr = ref(false)
const whCopied = ref(false)
const linkCopied = ref(false)
const isConnected = ref(props.instance.connection_status === 'OPEN')

const isWebhookOnly = computed(() => props.instance.provider === 'api_premium' || props.instance.provider === 'uazapi')
const providerLabel = computed(() => (props.instance.provider === 'api_premium' ? 'API Premium' : 'Uazapi'))

const normalizeQr = (raw?: string): string => {
  if (!raw) return ''
  return raw.startsWith('data:image') ? raw : `data:image/png;base64,${raw}`
}

const loadQr = async () => {
  if (isWebhookOnly.value || isConnected.value) return
  loadingQr.value = true
  try {
    const res = await wa.getQrCode(props.instance.id)
    if (res.connected) {
      isConnected.value = true
      waStore.updateInstanceStatus(props.instance.id, 'OPEN')
      emit('connected')
      return
    }
    qrDataUrl.value = normalizeQr(res.qrcode)
  } catch (e: unknown) {
    toast.error('Erro ao obter QR Code', errorMessage(e, 'Tente novamente em instantes.'))
  } finally {
    loadingQr.value = false
  }
}

const copyWebhook = async () => {
  if (!props.instance.webhook_url) return
  await navigator.clipboard.writeText(props.instance.webhook_url)
  whCopied.value = true
  setTimeout(() => { whCopied.value = false }, 2200)
  toast.success('Webhook copiado!', `Cole na configuração da ${providerLabel.value}.`)
}

const copyShareLink = async () => {
  if (!props.instance.share_token || typeof window === 'undefined') return
  const url = `${window.location.origin}/conectar-whatsapp/${props.instance.share_token}`
  await navigator.clipboard.writeText(url)
  linkCopied.value = true
  setTimeout(() => { linkCopied.value = false }, 2200)
  toast.success('Link copiado!', 'Envie este link para o cliente escanear o QR Code.')
}

let offQr: (() => void) | null = null
let offStatus: (() => void) | null = null

onMounted(() => {
  if (!isWebhookOnly.value && !isConnected.value) void loadQr()
  offQr = liveStore.onCustomEvent('whatsapp_qr', (raw: unknown) => {
    const d = raw as { instance_id?: number; qrcode?: string } | null
    if (d?.instance_id === props.instance.id && d.qrcode) qrDataUrl.value = normalizeQr(d.qrcode)
  })
  offStatus = liveStore.onCustomEvent('whatsapp_status', (raw: unknown) => {
    const d = raw as { instance_id?: number; status?: string; owner_jid?: string; profile_name?: string; profile_pic_url?: string } | null
    if (d?.instance_id !== props.instance.id || !d.status) return
    waStore.updateInstanceStatus(props.instance.id, d.status, d.owner_jid, d.profile_name, d.profile_pic_url)
    if (d.status === 'OPEN') {
      isConnected.value = true
      toast.success('WhatsApp conectado!', 'Aparelho sincronizado com sucesso.')
      emit('connected')
    }
  })
})

onUnmounted(() => {
  offQr?.()
  offStatus?.()
})
</script>

<style scoped>
.connect-pane { display: flex; flex-direction: column; align-items: center; gap: 0.9rem; text-align: center; }
.connected-box {
  width: 100%; padding: 1.5rem; border-radius: 12px; background: var(--success-soft);
  border: 1px solid var(--success); display: flex; flex-direction: column; align-items: center; gap: 0.45rem;
}
.ok-icon { color: var(--success); }
.ok-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.ok-desc { font-size: var(--fs-sm); color: var(--text-2); margin: 0; }
.ok-desc strong { color: var(--text-strong); font-weight: var(--fw-semibold); }
.wh-card { width: 100%; padding: 1rem; border-radius: 11px; background: var(--surface-deep); border: 1px solid var(--border); display: flex; flex-direction: column; gap: 0.65rem; text-align: left; }
.wh-label { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-2); }
.wh-code { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--brand); background: var(--surface-2); padding: 0.6rem 0.75rem; border-radius: 8px; word-break: break-all; }
.qr-frame { padding: 0.75rem; background: var(--surface-qr); border: 2px solid var(--brand-border); border-radius: 14px; display: inline-flex; }
.qr-img { width: 210px; height: 210px; object-fit: contain; display: block; }
.qr-box { width: 226px; height: 226px; border-radius: 14px; background: var(--surface-deep); border: 1px dashed var(--border); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.65rem; padding: 1rem; font-size: var(--fs-xs); color: var(--text-3); }
.muted-icon { color: var(--text-3); opacity: 0.6; }
.qr-actions { display: flex; align-items: center; justify-content: center; gap: 0.5rem; flex-wrap: wrap; }
.connect-hint { font-size: var(--fs-xs); color: var(--text-3); max-width: 480px; line-height: 1.5; margin: 0; }
.connect-hint code { font-family: var(--font-mono); color: var(--text-2); background: var(--surface-soft); padding: 0.08rem 0.3rem; border-radius: 4px; }
.spinner { width: 1.5rem; height: 1.5rem; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
