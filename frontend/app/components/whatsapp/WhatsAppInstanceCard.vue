<template>
  <article class="inst-card">
    <div class="inst-top">
      <header class="inst-header">
        <div class="inst-avatar-wrap">
          <img
            v-if="hasValidPhoto"
            :src="instance.profile_pic_url"
            :alt="instance.profile_name || instance.name"
            class="inst-avatar-img"
            @error="imgError = true"
          >
          <div v-else class="inst-avatar">
            <Smartphone :size="20" />
          </div>
          <span class="avatar-dot" :class="dotClass" />
        </div>

        <div class="inst-titles">
          <h3 class="inst-name">{{ instance.name }}</h3>
          <p class="inst-sub">{{ headerSubtitle }}</p>
        </div>

        <span class="status-pill" :class="pillClass">
          <span class="status-dot" :class="dotClass" />
          {{ statusLabel }}
        </span>
      </header>

      <div class="inst-body">
        <div class="meta-row">
          <span class="meta-label">Provedor</span>
          <span class="meta-val">{{ providerLabel }}</span>
        </div>

        <div class="meta-row">
          <span class="meta-label">WhatsApp / Número</span>
          <span class="meta-val meta-mono" :class="{ muted: !formattedPhone }">
            {{ formattedPhone || (isExternalProvider ? 'Aguardando Webhook' : 'Não conectado') }}
          </span>
        </div>

        <div class="meta-row">
          <span class="meta-label">Pixel & Moeda</span>
          <span class="meta-val">{{ instance.pixel_name || `#${instance.pixel_id}` }} · {{ instance.currency || 'BRL' }}</span>
        </div>

        <div class="meta-row">
          <span class="meta-label">CRM Automático</span>
          <span class="meta-val" :class="instance.auto_create_crm_lead ? 'val-ok' : 'muted'">
            {{ instance.auto_create_crm_lead ? 'Ativo' : 'Desativado' }}
          </span>
        </div>

        <div v-if="isExternalProvider && instance.webhook_url" class="meta-row">
          <span class="meta-label">URL do Webhook</span>
          <button type="button" class="copy-link-badge" :class="{ copied: whCopied }" title="Copiar URL do Webhook" @click="copyWebhookUrl">
            <Check v-if="whCopied" :size="13" />
            <Link2 v-else :size="13" />
            <span>{{ whCopied ? 'Webhook copiado!' : 'Copiar Webhook' }}</span>
          </button>
        </div>

        <div v-else-if="instance.share_token" class="meta-row">
          <span class="meta-label">Link p/ Cliente</span>
          <button type="button" class="copy-link-badge" :class="{ copied: linkCopied }" title="Copiar link externo para o cliente conectar o WhatsApp" @click="copyClientLink">
            <Check v-if="linkCopied" :size="13" />
            <Link2 v-else :size="13" />
            <span>{{ linkCopied ? 'Link copiado!' : 'Copiar link' }}</span>
          </button>
        </div>
      </div>
    </div>

    <footer class="inst-actions">
      <button v-if="isExternalProvider" type="button" class="action-main action-connect" @click="$emit('connect', instance)">
        <Link2 :size="16" />
        <span>Ver Webhook</span>
      </button>
      <button v-else-if="!isConnected" type="button" class="action-main action-connect" @click="$emit('connect', instance)">
        <QrCode :size="16" />
        <span>Conectar</span>
      </button>
      <button v-else type="button" class="action-main action-disconnect" :disabled="disconnecting" @click="$emit('disconnect', instance)">
        <Power :size="16" />
        <span>{{ disconnecting ? 'Desconectando...' : 'Desconectar' }}</span>
      </button>

      <template v-if="!isExternalProvider">
        <div class="action-divider" />
        <button type="button" class="action-icon" :disabled="refreshing" title="Sincronizar status e foto" @click="$emit('refresh', instance)">
          <RefreshCw :size="16" :class="{ spin: refreshing }" />
        </button>
      </template>

      <div class="action-divider" />

      <button type="button" class="action-icon" title="Configurações e gatilhos" @click="$emit('edit', instance)">
        <Settings :size="16" />
      </button>

      <div class="action-divider" />

      <button type="button" class="action-icon action-delete" title="Excluir conexão" @click="$emit('delete', instance)">
        <Trash2 :size="16" />
      </button>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { Smartphone, QrCode, Power, RefreshCw, Settings, Trash2, Link2, Check } from 'lucide-vue-next'
import type { WhatsAppInstance } from '#shared/types'

const props = defineProps<{ instance: WhatsAppInstance; refreshing?: boolean; disconnecting?: boolean }>()
defineEmits<{ connect: [inst: WhatsAppInstance]; disconnect: [inst: WhatsAppInstance]; refresh: [inst: WhatsAppInstance]; edit: [inst: WhatsAppInstance]; delete: [inst: WhatsAppInstance] }>()

const toast = useToast()
const imgError = ref(false)
const linkCopied = ref(false)
const whCopied = ref(false)

watch(() => props.instance.profile_pic_url, () => { imgError.value = false })

const isExternalProvider = computed(() => props.instance.provider === 'api_premium' || props.instance.provider === 'uazapi')
const providerLabel = computed(() => {
  if (props.instance.provider === 'api_premium') return 'API Premium'
  if (props.instance.provider === 'uazapi') return 'Uazapi'
  return 'Conexão Direta (QR Code)'
})

const isConnected = computed(() => props.instance.connection_status === 'OPEN')
const hasValidPhoto = computed(() => Boolean(props.instance.profile_pic_url && !imgError.value))

const statusState = computed<'connected' | 'connecting' | 'disconnected'>(() => {
  if (!props.instance.enabled) return 'disconnected'
  if (props.instance.connection_status === 'OPEN') return 'connected'
  if (props.instance.connection_status === 'CONNECTING') return 'connecting'
  return 'disconnected'
})

const statusLabel = computed(() => {
  if (!props.instance.enabled) return 'Pausada'
  if (statusState.value === 'connected') return 'Conectada'
  if (statusState.value === 'connecting') return isExternalProvider.value ? 'Aguardando' : 'Pareando'
  return 'Desconectada'
})

const pillClass = computed(() => `pill-${statusState.value}`)
const dotClass = computed(() => `dot-${statusState.value}`)

const formattedPhone = computed(() => {
  const jid = props.instance.owner_jid || ''
  const digits = jid.split('@')[0]?.replace(/\D/g, '') || ''
  if (!digits) return ''
  if (digits.startsWith('55') && (digits.length === 12 || digits.length === 13)) {
    const ddd = digits.slice(2, 4)
    const rest = digits.slice(4)
    if (rest.length === 9) return `+55 (${ddd}) ${rest.slice(0, 5)}-${rest.slice(5)}`
    return `+55 (${ddd}) ${rest.slice(0, 4)}-${rest.slice(4)}`
  }
  return `+${digits}`
})

const headerSubtitle = computed(() => {
  if (isConnected.value) {
    if (props.instance.profile_name) return props.instance.profile_name
    if (formattedPhone.value) return formattedPhone.value
  }
  if (isExternalProvider.value) return `Webhook ${providerLabel.value} ativo`
  if (statusState.value === 'connecting') return 'Aguardando leitura do QR Code'
  return 'Aparelho desconectado'
})

const copyClientLink = async () => {
  if (!props.instance.share_token || typeof window === 'undefined') return
  const url = `${window.location.origin}/conectar-whatsapp/${props.instance.share_token}`
  try {
    await navigator.clipboard.writeText(url)
    linkCopied.value = true
    setTimeout(() => { linkCopied.value = false }, 2200)
    toast.success('Link copiado!', 'Envie este link para seu cliente conectar o WhatsApp.')
  } catch {
    toast.error('Erro ao copiar', url)
  }
}

const copyWebhookUrl = async () => {
  if (!props.instance.webhook_url || typeof window === 'undefined') return
  try {
    await navigator.clipboard.writeText(props.instance.webhook_url)
    whCopied.value = true
    setTimeout(() => { whCopied.value = false }, 2200)
    toast.success('Webhook copiado!', `Cole no painel da ${providerLabel.value}.`)
  } catch {
    toast.error('Erro ao copiar', props.instance.webhook_url)
  }
}
</script>

<style scoped>
.inst-card { display: flex; flex-direction: column; justify-content: space-between; background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px; overflow: hidden; transition: border-color 0.2s ease, transform 0.2s ease; }
.inst-card:hover { border-color: var(--brand-border); transform: translateY(-1px); }
.inst-header { display: flex; align-items: center; gap: 0.875rem; padding: 1.125rem 1.25rem; border-bottom: 1px solid var(--border-soft); }
.inst-avatar-wrap { position: relative; width: 46px; height: 46px; flex-shrink: 0; }
.inst-avatar, .inst-avatar-img { width: 46px; height: 46px; border-radius: 12px; object-fit: cover; display: flex; align-items: center; justify-content: center; }
.inst-avatar-img { border: 1px solid var(--border); background: var(--surface-deep); }
.inst-avatar { background: var(--brand-soft); border: 1px solid var(--brand-border); color: var(--brand); }
.avatar-dot { position: absolute; right: -2px; bottom: -2px; width: 11px; height: 11px; border-radius: 50%; border: 2px solid var(--surface-2); }
.inst-titles { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 0.15rem; }
.inst-name { font-family: var(--font-heading); font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.inst-sub { font-size: var(--fs-xs); color: var(--text-3); margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.status-pill { display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.3rem 0.7rem; border-radius: 999px; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.04em; white-space: nowrap; flex-shrink: 0; }
.pill-connected { background: var(--surface-soft); color: var(--success); border: 1px solid var(--border-soft); }
.pill-connecting { background: var(--surface-soft); color: var(--warning); border: 1px solid var(--border-soft); }
.pill-disconnected { background: var(--surface-deep); color: var(--text-3); border: 1px solid var(--border-soft); }
.status-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.dot-connected { background: var(--success); }
.dot-connecting { background: var(--warning); }
.dot-disconnected { background: var(--text-3); }
.inst-body { padding: 0.95rem 1.25rem; display: flex; flex-direction: column; gap: 0.6rem; background: var(--surface-deep); border-bottom: 1px solid var(--border-soft); }
.meta-row { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; font-size: var(--fs-xs); }
.meta-label { color: var(--text-3); font-weight: var(--fw-medium); white-space: nowrap; }
.meta-val { color: var(--text-1); font-weight: var(--fw-semibold); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; text-align: right; }
.meta-mono { font-family: var(--font-mono); }
.val-ok { color: var(--success); }
.muted { color: var(--text-3); font-weight: var(--fw-normal); }
.copy-link-badge { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.22rem 0.6rem; border-radius: 6px; background: var(--surface-2); border: 1px solid var(--border); color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-semibold); cursor: pointer; transition: all 0.15s ease; }
.copy-link-badge:hover { border-color: var(--brand); background: var(--brand-soft); }
.copy-link-badge.copied { color: var(--success); border-color: var(--success); }
.inst-actions { display: flex; align-items: stretch; height: 46px; background: var(--surface-2); }
.action-main { flex: 1; display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem; padding: 0 1rem; border: none; background: transparent; font-size: var(--fs-sm); font-weight: var(--fw-semibold); cursor: pointer; white-space: nowrap; transition: background 0.15s ease, color 0.15s ease; }
.action-connect { color: var(--brand); }
.action-connect:hover { background: var(--brand-soft); }
.action-disconnect { color: var(--warning); }
.action-disconnect:hover { background: var(--surface-soft); }
.action-divider { width: 1px; background: var(--border-soft); flex-shrink: 0; }
.action-icon { width: 46px; display: inline-flex; align-items: center; justify-content: center; border: none; background: transparent; color: var(--text-2); cursor: pointer; flex-shrink: 0; transition: background 0.15s ease, color 0.15s ease; }
.action-icon:hover { background: var(--surface-soft); color: var(--text-strong); }
.action-delete { color: var(--danger); }
.action-delete:hover { background: var(--surface-soft); color: var(--danger); }
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
