<template>
  <section class="device-card">
    <div class="card-top">
      <div class="title-row">
        <div class="icon-badge">
          <Smartphone :size="18" />
        </div>
        <div>
          <h2 class="card-title">Aplicativo PWA & Este Dispositivo</h2>
          <p class="card-desc">Receba alertas nativos no PC ou celular mesmo com o painel fechado.</p>
        </div>
      </div>
      <BaseBadge :variant="isThisDeviceSubscribed ? 'success' : 'default'">
        {{ isThisDeviceSubscribed ? 'Ativo neste aparelho' : 'Não ativado aqui' }}
      </BaseBadge>
    </div>

    <div v-if="isIOS && !isStandalonePWA" class="ios-guide">
      <p class="ios-title">Como ativar no iPhone / iPad (iOS)</p>
      <p class="ios-text">
        Toque em <strong>Compartilhar</strong> na barra do Safari e escolha
        <strong>Adicionar à Tela de Início</strong>. Depois abra o app pela tela inicial e clique em ativar abaixo.
      </p>
    </div>

    <div v-else-if="permissionState === 'denied'" class="denied-box">
      As notificações foram bloqueadas nas permissões deste navegador. Libere o cadeado ao lado da URL e recarregue a página.
    </div>

    <div class="action-row">
      <BaseButton
        v-if="canInstallPWA"
        variant="outline"
        @click="$emit('install')"
      >
        <Download :size="16" />
        <span>Instalar aplicativo</span>
      </BaseButton>

      <BaseButton
        v-if="!isThisDeviceSubscribed"
        variant="primary"
        :loading="subscribing"
        :disabled="!pushSupported || (isIOS && !isStandalonePWA)"
        @click="$emit('subscribe')"
      >
        <BellRing :size="16" />
        <span>Ativar notificações neste aparelho</span>
      </BaseButton>

      <span v-else class="active-hint">
        <CheckCircle2 :size="16" />
        <span>Este navegador está pronto para receber alertas em segundo plano.</span>
      </span>
    </div>

    <div v-if="subscriptions.length > 0" class="subs-section">
      <span class="subs-heading">Dispositivos conectados na sua conta ({{ subscriptions.length }})</span>
      <div class="subs-list">
        <div v-for="sub in subscriptions" :key="sub.id" class="sub-row">
          <Monitor v-if="!isMobileLabel(sub.device_label)" :size="16" class="sub-icon" />
          <Smartphone v-else :size="16" class="sub-icon" />
          <div class="sub-info">
            <span class="sub-name">
              {{ sub.device_label || 'Navegador conectado' }}
              <em v-if="sub.endpoint === currentEndpoint" class="current-tag">(Este aparelho)</em>
            </span>
            <span class="sub-date">Conectado em {{ formatShortDate(sub.created_at) }}</span>
          </div>
          <button
            type="button"
            class="remove-btn"
            title="Remover dispositivo"
            @click="$emit('remove', sub)"
          >
            <Trash2 :size="15" />
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Smartphone, Monitor, BellRing, Download, Trash2, CheckCircle2 } from 'lucide-vue-next'
import type { PushSubscriptionItem } from '#shared/types'

defineProps<{
  pushSupported: boolean
  permissionState: NotificationPermission
  isIOS: boolean
  isStandalonePWA: boolean
  canInstallPWA: boolean
  isThisDeviceSubscribed: boolean
  currentEndpoint: string
  subscribing: boolean
  subscriptions: PushSubscriptionItem[]
}>()

defineEmits<{
  (e: 'install'): void
  (e: 'subscribe'): void
  (e: 'remove', sub: PushSubscriptionItem): void
}>()

const isMobileLabel = (label: string) => /iphone|ipad|android/i.test(label || '')

const formatShortDate = (iso: string) => {
  if (!iso) return ''
  return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' })
}
</script>

<style scoped>
.device-card {
  background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px;
  padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem;
}
.card-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; flex-wrap: wrap; }
.title-row { display: flex; align-items: flex-start; gap: 0.75rem; }
.icon-badge {
  width: 38px; height: 38px; border-radius: 10px; background: var(--brand-soft); color: var(--brand);
  border: 1px solid var(--brand-border); display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.card-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0 0 0.15rem; }
.card-desc { font-size: var(--fs-base); color: var(--text-2); margin: 0; }
.ios-guide {
  padding: 0.875rem 1rem; border-radius: 10px; background: var(--brand-soft);
  border: 1px solid var(--brand-border); color: var(--text-1);
}
.ios-title { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--brand); margin: 0 0 0.25rem; }
.ios-text { font-size: var(--fs-sm); color: var(--text-2); margin: 0; line-height: 1.5; }
.denied-box {
  padding: 0.75rem 1rem; border-radius: 10px; background: var(--danger-soft);
  border: 1px solid var(--danger); color: var(--danger); font-size: var(--fs-sm);
}
.action-row { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; }
.active-hint { display: inline-flex; align-items: center; gap: 0.45rem; font-size: var(--fs-base); color: var(--success); font-weight: var(--fw-medium); }
.subs-section { border-top: 1px solid var(--border-soft); padding-top: 0.875rem; display: flex; flex-direction: column; gap: 0.5rem; }
.subs-heading { font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-3); }
.subs-list { display: flex; flex-direction: column; gap: 0.375rem; }
.sub-row {
  display: flex; align-items: center; gap: 0.65rem; padding: 0.55rem 0.75rem;
  border-radius: 8px; background: var(--surface-1); border: 1px solid var(--border-soft);
}
.sub-icon { color: var(--brand); flex-shrink: 0; }
.sub-info { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.sub-name { font-size: var(--fs-base); font-weight: var(--fw-medium); color: var(--text-1); }
.current-tag { font-style: normal; color: var(--brand); font-size: var(--fs-xs); margin-left: 0.25rem; }
.sub-date { font-size: var(--fs-xs); color: var(--text-3); }
.remove-btn {
  width: 30px; height: 30px; border-radius: 6px; border: none; background: transparent;
  color: var(--text-3); display: inline-flex; align-items: center; justify-content: center; cursor: pointer;
}
.remove-btn:hover { background: var(--danger-soft); color: var(--danger); }
</style>
