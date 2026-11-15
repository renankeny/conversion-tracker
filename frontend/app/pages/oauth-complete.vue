<template>
  <div class="oauth-complete-shell">
    <div class="oauth-complete-card">
      <div class="status-icon" :class="isSuccess ? 'is-ok' : 'is-err'">
        <Check v-if="isSuccess" :size="24" />
        <AlertTriangle v-else :size="24" />
      </div>
      <h1 class="status-title">
        {{ isSuccess ? 'Conexão concluída!' : 'Não foi possível concluir o login' }}
      </h1>
      <p class="status-desc">
        {{ statusMessage }}
      </p>
      <BaseButton variant="primary" size="sm" @click="closeOrReturn">
        Fechar janela
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Check, AlertTriangle } from 'lucide-vue-next'

definePageMeta({
  layout: false,
})

const route = useRoute()

const channel = computed(() => String(route.query.channel || 'meta'))
const status = computed(() => (route.query.status === 'success' ? 'success' : 'error'))
const accountId = computed(() => Number(route.query.account_id || 0) || 0)
const rawMessage = computed(() => String(route.query.message || '').trim())
const isSuccess = computed(() => status.value === 'success')

const statusMessage = computed(() => {
  if (isSuccess.value) {
    return rawMessage.value
      ? `Conectado como ${rawMessage.value}. Esta janela fechará automaticamente.`
      : 'Conta autenticada com sucesso. Esta janela fechará automaticamente.'
  }
  return rawMessage.value || 'A autenticação foi cancelada ou expirou. Você pode fechar esta janela e tentar novamente.'
})

const notifyOpener = () => {
  if (!import.meta.client) return
  const payload = {
    type: 'trackfind:oauth-complete',
    channel: channel.value,
    status: status.value,
    accountId: accountId.value,
    message: rawMessage.value,
    ts: Date.now(),
  }

  try {
    if (typeof BroadcastChannel !== 'undefined') {
      const bc = new BroadcastChannel('trackfind_oauth')
      bc.postMessage(payload)
      bc.close()
    }
  } catch {}

  try {
    localStorage.setItem('trackfind_oauth_event', JSON.stringify(payload))
  } catch {}

  try {
    if (window.opener && !window.opener.closed) {
      window.opener.postMessage(payload, window.location.origin)
    }
  } catch {}
}

const closeOrReturn = () => {
  if (!import.meta.client) return
  window.close()
  setTimeout(() => {
    const target = channel.value === 'google_ads' ? '/integracoes/google-ads' : '/integracoes/meta-ads'
    const qs = accountId.value > 0 ? `?account_id=${accountId.value}` : ''
    navigateTo(`${target}${qs}`, { replace: true })
  }, 250)
}

onMounted(() => {
  notifyOpener()
  setTimeout(() => {
    window.close()
  }, 120)
})
</script>

<style scoped>
.oauth-complete-shell {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  background: var(--surface-deep);
  color: var(--text-1);
}
.oauth-complete-card {
  width: 100%;
  max-width: 380px;
  padding: 2rem 1.5rem;
  border-radius: 14px;
  background: var(--surface-1);
  border: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 0.75rem;
}
.status-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}
.status-icon.is-ok {
  background: var(--success-soft);
  color: var(--success);
}
.status-icon.is-err {
  background: var(--danger-soft);
  color: var(--danger);
}
.status-title {
  margin: 0;
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
}
.status-desc {
  margin: 0 0 0.35rem;
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.45;
}
</style>
