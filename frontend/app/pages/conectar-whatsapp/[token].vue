<template>
  <div class="pub-wa-page">
    <div class="pub-wa-card">
      <div class="brand-header">
        <BrandLogo size="md" />
      </div>

      <div v-if="notFound" class="state-pane">
        <div class="danger-badge">
          <AlertCircle :size="32" />
        </div>
        <h1 class="card-title">Link de conexão inválido</h1>
        <p class="card-sub">
          Este link não foi encontrado ou foi desativado. Solicite um novo link ao responsável pela conta.
        </p>
      </div>

      <div v-else-if="isConnected" class="state-pane">
        <div class="connected-avatar-wrap">
          <img
            v-if="profilePicUrl && !avatarError"
            :src="profilePicUrl"
            :alt="profileName || instanceName"
            class="connected-avatar"
            @error="avatarError = true"
          >
          <div v-else class="success-badge">
            <CheckCircle2 :size="36" />
          </div>
          <span class="connected-check">
            <CheckCircle2 :size="16" />
          </span>
        </div>

        <h1 class="card-title">WhatsApp Conectado!</h1>
        <p class="card-sub">
          A conexão <strong>{{ instanceName }}</strong> está online e sincronizada com sucesso.
        </p>

        <div v-if="formattedPhone || profileName" class="connected-info">
          <span v-if="profileName" class="info-name">{{ profileName }}</span>
          <span v-if="formattedPhone" class="info-phone">{{ formattedPhone }}</span>
        </div>
      </div>

      <template v-else>
        <div class="header-text">
          <span v-if="instanceName" class="inst-tag">
            <Smartphone :size="13" />
            <span>{{ instanceName }}</span>
          </span>
          <h1 class="card-title">Conectar WhatsApp</h1>
          <p class="card-sub">
            Aponte a câmera do seu WhatsApp para o QR Code abaixo para sincronizar o aparelho.
          </p>
        </div>

        <div class="qr-pane">
          <div v-if="loading && !qrDataUrl" class="qr-placeholder">
            <QrCode :size="40" class="qr-icon-pulse" />
            <div class="spinner" />
            <span>Gerando QR Code...</span>
          </div>

          <div v-else-if="qrDataUrl" class="qr-frame">
            <img :src="qrDataUrl" :alt="`QR Code ${instanceName}`" class="qr-img">
          </div>

          <div v-else class="qr-placeholder">
            <QrCode :size="36" class="qr-icon-muted" />
            <span>Aguardando geração do QR Code...</span>
            <BaseButton variant="outline" size="sm" :loading="loading" @click="initAndFetch">
              Gerar QR Code
            </BaseButton>
          </div>

          <div class="listening-pill">
            <span class="pulse-dot" />
            <span>Aguardando leitura do QR Code pelo WhatsApp...</span>
          </div>

          <div class="steps-box">
            <span class="steps-title">Como conectar seu aparelho:</span>
            <ol class="steps-list">
              <li><span class="step-num">1</span><span>Abra o <strong>WhatsApp</strong> no seu celular</span></li>
              <li><span class="step-num">2</span><span>Toque em <strong>Mais opções</strong> ou <strong>Configurações</strong></span></li>
              <li><span class="step-num">3</span><span>Toque em <strong>Aparelhos conectados → Conectar aparelho</strong></span></li>
              <li><span class="step-num">4</span><span>Aponte a câmera do celular para este QR Code</span></li>
            </ol>
          </div>

          <BaseButton variant="outline" size="md" :loading="loading" @click="fetchStatus(false)">
            <RefreshCw :size="14" />
            <span>Atualizar QR Code</span>
          </BaseButton>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CheckCircle2, AlertCircle, RefreshCw, QrCode, Smartphone } from 'lucide-vue-next'
import { errorStatusCode } from '~/utils/errorMessage'

definePageMeta({ layout: 'default' })

const route = useRoute()
const token = computed(() => String(route.params.token || '').trim())
const api = useAPI()
const { formatTitle, fetchBranding } = useBranding()

const instanceName = ref('')
const profileName = ref('')
const profilePicUrl = ref('')
const avatarError = ref(false)
const ownerJid = ref('')
const status = ref('CONNECTING')
const qrDataUrl = ref('')
const loading = ref(true)
const notFound = ref(false)

useHead({ title: computed(() => formatTitle(instanceName.value ? `Conectar ${instanceName.value}` : 'Conectar WhatsApp')) })

const isConnected = computed(() => status.value === 'OPEN')

const formattedPhone = computed(() => {
  const digits = (ownerJid.value.split('@')[0] || '').replace(/\D/g, '')
  if (!digits) return ''
  if (digits.startsWith('55') && (digits.length === 12 || digits.length === 13)) {
    const ddd = digits.slice(2, 4)
    const rest = digits.slice(4)
    if (rest.length === 9) return `+55 (${ddd}) ${rest.slice(0, 5)}-${rest.slice(5)}`
    return `+55 (${ddd}) ${rest.slice(0, 4)}-${rest.slice(4)}`
  }
  return `+${digits}`
})

const normalizeQrSrc = (raw?: string): string => {
  if (!raw) return ''
  return raw.startsWith('data:image') ? raw : `data:image/png;base64,${raw}`
}

interface PublicStatusResp {
  success: boolean
  instance_name: string
  profile_name?: string
  profile_pic_url?: string
  owner_jid?: string
  status: string
  qrcode?: string
}

const fetchStatus = async (silent = false) => {
  if (!token.value) {
    notFound.value = true
    return
  }
  if (!silent) loading.value = true
  try {
    const res = await api.publicGet<PublicStatusResp>(`/public/whatsapp/${encodeURIComponent(token.value)}/status`)
    instanceName.value = res.instance_name || 'WhatsApp'
    profileName.value = res.profile_name || ''
    if (res.profile_pic_url && res.profile_pic_url !== profilePicUrl.value) {
      profilePicUrl.value = res.profile_pic_url
      avatarError.value = false
    }
    ownerJid.value = res.owner_jid || ''
    status.value = res.status || 'CONNECTING'
    if (res.qrcode) qrDataUrl.value = normalizeQrSrc(res.qrcode)
  } catch (e: unknown) {
    const code = errorStatusCode(e)
    if (code === 404 || code === 400) notFound.value = true
  } finally {
    if (!silent) loading.value = false
  }
}

const initAndFetch = async () => {
  if (!token.value) return
  loading.value = true
  try {
    await api.publicRequest(`/public/whatsapp/${encodeURIComponent(token.value)}/connect`, { method: 'POST' })
  } catch {}
  await fetchStatus(false)
}

let pollTimer: ReturnType<typeof setInterval> | null = null
let unmounted = false

onMounted(async () => {
  unmounted = false
  pollTimer = setInterval(() => {
    if (!unmounted && !notFound.value && !isConnected.value) void fetchStatus(true)
  }, 4500)
  void fetchBranding()
  await fetchStatus(false)
})

onUnmounted(() => {
  unmounted = true
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = null
})
</script>

<style scoped>
.pub-wa-page { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 1.5rem; background: var(--surface-1); }
.pub-wa-card { width: 100%; max-width: 440px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 18px; padding: 1.75rem; display: flex; flex-direction: column; gap: 1.25rem; }
.brand-header { display: flex; justify-content: center; padding-bottom: 1rem; border-bottom: 1px solid var(--border-soft); }
.header-text, .state-pane { display: flex; flex-direction: column; align-items: center; text-align: center; gap: 0.45rem; }
.inst-tag { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.2rem 0.65rem; border-radius: 999px; background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-semibold); margin-bottom: 0.15rem; }
.card-title { font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); margin: 0; }
.card-sub { font-size: var(--fs-sm); color: var(--text-2); line-height: 1.5; margin: 0; max-width: 340px; }
.danger-badge { width: 64px; height: 64px; border-radius: 50%; background: var(--surface-soft); color: var(--danger); display: flex; align-items: center; justify-content: center; margin-bottom: 0.25rem; }
.connected-avatar-wrap { position: relative; margin-bottom: 0.35rem; }
.connected-avatar { width: 76px; height: 76px; border-radius: 50%; object-fit: cover; border: 2px solid var(--success); display: block; }
.success-badge { width: 72px; height: 72px; border-radius: 50%; background: var(--surface-soft); color: var(--success); border: 1px solid var(--border-soft); display: flex; align-items: center; justify-content: center; }
.connected-check { position: absolute; right: -2px; bottom: -2px; width: 24px; height: 24px; border-radius: 50%; background: var(--surface-2); color: var(--success); display: flex; align-items: center; justify-content: center; }
.connected-info { width: 100%; margin-top: 0.5rem; padding: 0.75rem 1rem; border-radius: 12px; background: var(--surface-deep); border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 0.2rem; font-size: var(--fs-sm); }
.info-name { font-weight: var(--fw-semibold); color: var(--text-strong); }
.info-phone { font-family: var(--font-mono); color: var(--text-2); font-size: var(--fs-xs); }
.qr-pane { display: flex; flex-direction: column; align-items: center; gap: 1rem; text-align: center; }
.qr-frame { padding: 0.875rem; background: var(--surface-qr); border: 2px solid var(--brand-border); border-radius: 16px; display: inline-flex; }
.qr-img { width: 224px; height: 224px; display: block; object-fit: contain; }
.qr-placeholder { width: 252px; height: 252px; border-radius: 16px; background: var(--surface-deep); border: 1px dashed var(--border); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.75rem; color: var(--text-3); font-size: var(--fs-sm); padding: 1.25rem; }
.qr-icon-pulse { color: var(--brand); opacity: 0.65; }
.qr-icon-muted { color: var(--text-3); opacity: 0.5; }
.listening-pill { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.35rem 0.85rem; border-radius: 999px; background: var(--surface-deep); border: 1px solid var(--border-soft); color: var(--warning); font-size: var(--fs-xs); font-weight: var(--fw-medium); }
.pulse-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--warning); flex-shrink: 0; animation: pulse 1.8s ease-in-out infinite; }
.steps-box { width: 100%; padding: 0.95rem 1.1rem; border-radius: 12px; background: var(--surface-deep); border: 1px solid var(--border-soft); text-align: left; display: flex; flex-direction: column; gap: 0.6rem; }
.steps-title { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--text-1); text-transform: uppercase; letter-spacing: 0.04em; }
.steps-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 0.5rem; font-size: var(--fs-xs); color: var(--text-2); line-height: 1.4; }
.steps-list li { display: flex; align-items: center; gap: 0.6rem; }
.step-num { width: 20px; height: 20px; border-radius: 50%; background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand); font-size: var(--fs-2xs); font-weight: var(--fw-bold); display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; }
.spinner { width: 1.5rem; height: 1.5rem; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@keyframes pulse { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.45; transform: scale(0.85); } }
</style>
