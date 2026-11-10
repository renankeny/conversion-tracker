<template>
  <BaseModal
    :model-value="modelValue"
    :title="`Conectar ${instance?.name || 'WhatsApp'}`"
    description="Escaneie o QR Code no seu WhatsApp ou gere um código numérico de 8 dígitos."
    size="sm"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="connect-wrap">
      <div class="mode-tabs">
        <button
          type="button"
          class="mode-btn"
          :class="{ active: mode === 'qr' }"
          @click="switchMode('qr')"
        >
          QR Code
        </button>
        <button
          type="button"
          class="mode-btn"
          :class="{ active: mode === 'pair' }"
          @click="switchMode('pair')"
        >
          Código de 8 dígitos
        </button>
      </div>

      <div v-if="mode === 'qr'" class="qr-pane">
        <div v-if="loadingQr && !qrDataUrl" class="qr-placeholder">
          <div class="spinner" />
          <span>Gerando QR Code...</span>
        </div>

        <div v-else-if="qrDataUrl" class="qr-frame">
          <img :src="qrDataUrl" alt="QR Code WhatsApp" class="qr-img">
        </div>

        <div v-else class="qr-placeholder">
          <span>Não foi possível exibir o QR Code agora.</span>
          <BaseButton variant="outline" size="sm" :loading="loadingQr" @click="loadQr">
            Tentar novamente
          </BaseButton>
        </div>

        <p class="qr-steps">
          No celular: <strong>WhatsApp → Aparelhos conectados → Conectar aparelho</strong>.
          O QR Code atualiza sozinho em tempo real.
        </p>
      </div>

      <div v-else class="pair-pane">
        <BaseInput
          v-model="phone"
          label="Número do WhatsApp (com DDI 55 + DDD)"
          placeholder="5511999999999"
        />

        <BaseButton
          variant="primary"
          :loading="loadingPair"
          @click="requestPairCode"
        >
          Gerar código de pareamento
        </BaseButton>

        <div v-if="pairingCode" class="pair-code-box">
          <span class="pair-label">Digite este código no seu WhatsApp:</span>
          <strong class="pair-code">{{ pairingCode }}</strong>
        </div>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="emit('update:modelValue', false)">
        Fechar
      </BaseButton>
      <BaseButton v-if="mode === 'qr'" variant="outline" :loading="loadingQr" @click="loadQr">
        Atualizar QR
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
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

const mode = ref<'qr' | 'pair'>('qr')
const qrDataUrl = ref('')
const loadingQr = ref(false)
const phone = ref('')
const pairingCode = ref('')
const loadingPair = ref(false)

const normalizeQrSrc = (raw?: string): string => {
  if (!raw) return ''
  return raw.startsWith('data:image') ? raw : `data:image/png;base64,${raw}`
}

const loadQr = async () => {
  if (!props.instance) return
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

const requestPairCode = async () => {
  if (!props.instance) return
  if (phone.value.replace(/\D/g, '').length < 10) {
    toast.error('Número inválido', 'Digite o DDI + DDD + número (ex: 5511999999999).')
    return
  }
  loadingPair.value = true
  try {
    const res = await wa.pairPhone(props.instance.id, phone.value)
    pairingCode.value = res.pairing_code
  } catch (e: any) {
    toast.error('Erro no pareamento', e.message || 'Falha ao gerar código.')
  } finally {
    loadingPair.value = false
  }
}

const switchMode = (next: 'qr' | 'pair') => {
  mode.value = next
  if (next === 'qr' && !qrDataUrl.value) void loadQr()
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      mode.value = 'qr'
      qrDataUrl.value = ''
      pairingCode.value = ''
      void loadQr()
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
    waStore.updateInstanceStatus(props.instance.id, data.status, data.owner_jid, data.profile_name)
    if (props.modelValue && data.status === 'OPEN') {
      toast.success('WhatsApp conectado!', 'Aparelho pareado com sucesso.')
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
.connect-wrap { display: flex; flex-direction: column; gap: 1rem; }
.mode-tabs {
  display: flex; gap: 0.25rem; padding: 0.25rem;
  background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 9px;
}
.mode-btn {
  flex: 1; padding: 0.45rem 0.75rem; border-radius: 6px; border: none;
  background: transparent; color: var(--text-3); font-size: var(--fs-sm);
  font-weight: var(--fw-semibold); cursor: pointer;
}
.mode-btn.active { background: var(--surface-soft); color: var(--brand); }
.qr-pane { display: flex; flex-direction: column; align-items: center; gap: 0.875rem; text-align: center; }
.qr-frame { padding: 0.75rem; background: var(--surface-qr); border-radius: 12px; display: inline-flex; }
.qr-img { width: 220px; height: 220px; display: block; }
.qr-placeholder {
  width: 236px; height: 236px; border-radius: 12px;
  background: var(--surface-deep); border: 1px dashed var(--border);
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 0.75rem; color: var(--text-3); font-size: var(--fs-sm); padding: 1rem;
}
.qr-steps { font-size: var(--fs-sm); color: var(--text-2); line-height: 1.5; margin: 0; }
.pair-pane { display: flex; flex-direction: column; gap: 0.875rem; }
.pair-code-box {
  padding: 1rem; border-radius: 10px; background: var(--surface-deep);
  border: 1px solid var(--brand-border); text-align: center;
  display: flex; flex-direction: column; gap: 0.35rem;
}
.pair-label { font-size: var(--fs-sm); color: var(--text-2); }
.pair-code { font-family: var(--font-mono); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--brand); letter-spacing: 0.12em; }
.spinner { width: 1.75rem; height: 1.75rem; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
