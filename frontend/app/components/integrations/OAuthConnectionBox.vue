<template>
  <section class="form-section">
    <header class="section-header">
      <h4>{{ title }}</h4>
      <p class="section-hint">{{ description }}</p>
    </header>

    <div v-if="connected || tokenExpired" class="oauth-box" :class="{ 'is-expired': tokenExpired }">
      <div class="oauth-info">
        <div class="oauth-avatar">
          <img
            v-if="avatarUrl && !brokenUrls[avatarUrl]"
            :src="avatarUrl"
            alt=""
            referrerpolicy="no-referrer"
            class="oauth-avatar-img"
            @error="markBroken(avatarUrl)"
          >
          <span v-else class="oauth-initials">{{ initials(identityLabel) }}</span>
        </div>

        <div class="oauth-stack">
          <div class="oauth-title-row">
            <span class="oauth-identity">{{ identityLabel }}</span>
            <span v-if="tokenExpired" class="oauth-pill pill-danger">Token expirado</span>
            <span v-else-if="expiryWarning" class="oauth-pill pill-warn">{{ expiryWarning }}</span>
            <span v-else class="oauth-pill pill-ok">
              {{ channel === 'meta' ? 'Meta conectada' : 'Google conectado' }}
            </span>
          </div>
          <span v-if="identitySub && identitySub !== identityLabel" class="oauth-sub">
            {{ identitySub }}
          </span>
        </div>
      </div>

      <div class="oauth-actions">
        <BaseButton
          v-if="tokenExpired || expiryWarning"
          type="button"
          :variant="tokenExpired ? 'primary' : 'ghost'"
          size="sm"
          :loading="loading"
          @click="$emit('connect', false)"
        >
          {{ tokenExpired ? 'Reconectar agora' : 'Renovar conexão' }}
        </BaseButton>
        <BaseButton
          type="button"
          variant="ghost"
          size="sm"
          :loading="loading"
          @click="$emit('connect', true)"
        >
          Conectar nova conta
        </BaseButton>
        <BaseButton
          type="button"
          variant="ghost"
          size="sm"
          :loading="loading"
          @click="showDisconnectConfirm = true"
        >
          Desconectar
        </BaseButton>
      </div>
    </div>

    <div v-else-if="reuseOptions.length" class="reuse-container">
      <BaseAdAccountSelect
        v-model="selectedSourceId"
        :label="channel === 'meta' ? 'Conta Facebook já conectada neste painel' : 'Conta Google já conectada neste painel'"
        :options="reuseOptions"
      />
      <div class="oauth-actions-bar">
        <BaseButton
          type="button"
          variant="primary"
          size="sm"
          :loading="loading"
          @click="emitReuse"
        >
          Conectar com conta existente
        </BaseButton>
        <BaseButton
          type="button"
          variant="ghost"
          size="sm"
          :loading="loading"
          @click="$emit('connect', true)"
        >
          Conectar nova conta
        </BaseButton>
      </div>
    </div>

    <div v-else class="oauth-box">
      <p class="oauth-text">
        Abra a janela segura de login para autorizar em 1 clique sem sair desta tela.
      </p>
      <BaseButton
        type="button"
        variant="primary"
        size="sm"
        :loading="loading"
        @click="$emit('connect', false)"
      >
        Conectar nova conta
      </BaseButton>
    </div>

    <ConfirmModal
      v-model="showDisconnectConfirm"
      :title="disconnectTitle"
      :message="disconnectMessage"
      confirm-label="Desconectar"
      variant="danger"
      :loading="loading"
      @confirm="onConfirmDisconnect"
    />
  </section>
</template>

<script setup lang="ts">
import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'

const props = defineProps<{
  title: string
  description: string
  channel: 'meta' | 'google_ads'
  connected: boolean
  tokenExpired?: boolean
  expiryWarning?: string
  identityLabel: string
  identitySub?: string
  avatarUrl?: string
  disconnectTitle: string
  disconnectMessage: string
  reuseOptions: AdAccountSelectOption[]
  loading?: boolean
}>()

const emit = defineEmits<{
  connect: [switchAccount: boolean]
  reuse: [sourceAccountId: number]
  disconnect: []
}>()

const showDisconnectConfirm = ref(false)
const selectedSourceId = ref<number>(0)
const brokenUrls = reactive<Record<string, boolean>>({})

const markBroken = (url?: string) => {
  if (url) brokenUrls[url] = true
}

watch(
  () => props.reuseOptions,
  (opts) => {
    if (opts.length && !opts.some((o) => Number(o.value) === Number(selectedSourceId.value))) {
      selectedSourceId.value = Number(opts[0]!.value) || 0
    }
  },
  { immediate: true },
)

const initials = (name: string) => {
  const clean = (name || '').split('@')[0]?.trim() || 'AD'
  const parts = clean.split(/\s+/).filter(Boolean)
  if (!parts.length) return 'AD'
  return parts.length === 1
    ? parts[0]!.slice(0, 2).toUpperCase()
    : (parts[0]![0]! + parts[1]![0]!).toUpperCase()
}

const emitReuse = () => {
  const id = Number(selectedSourceId.value) || Number(props.reuseOptions[0]?.value) || 0
  if (id > 0) emit('reuse', id)
}

const triggerDefaultAction = () => {
  if (props.reuseOptions.length) {
    emitReuse()
  } else {
    emit('connect', false)
  }
}

const onConfirmDisconnect = () => {
  showDisconnectConfirm.value = false
  emit('disconnect')
}

defineExpose({ triggerDefaultAction })
</script>

<style scoped>
.form-section { display: flex; flex-direction: column; gap: 0.75rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.section-header h4 { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); margin: 0.25rem 0 0; line-height: 1.5; }
.reuse-container { display: flex; flex-direction: column; gap: 0.65rem; }
.oauth-box { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 0.875rem; background: var(--surface-soft); border: 1px solid var(--border); border-radius: 8px; flex-wrap: wrap; }
.oauth-box.is-expired { border-color: var(--danger-border); background: var(--danger-soft); }
.oauth-info, .oauth-actions, .oauth-actions-bar { display: flex; align-items: center; gap: 0.55rem; flex-wrap: wrap; }
.oauth-avatar { width: 34px; height: 34px; border-radius: 8px; background: var(--brand-soft); border: 1px solid var(--brand-border); display: flex; align-items: center; justify-content: center; overflow: hidden; flex-shrink: 0; }
.oauth-avatar-img { width: 100%; height: 100%; object-fit: cover; display: block; }
.oauth-initials { font-size: var(--fs-xs); font-weight: var(--fw-bold); color: var(--brand); }
.oauth-stack { display: flex; flex-direction: column; gap: 0.15rem; min-width: 0; }
.oauth-title-row { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.oauth-identity { font-size: var(--fs-md); color: var(--text-1); font-weight: var(--fw-medium); }
.oauth-sub { font-size: var(--fs-xs); color: var(--text-3); font-family: var(--font-mono); }
.oauth-pill { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); padding: 0.18rem 0.55rem; border-radius: 999px; white-space: nowrap; }
.pill-ok { background: var(--success-soft); color: var(--success); }
.pill-warn { background: var(--warning-soft); color: var(--warning); border: 1px solid var(--warning-border); }
.pill-danger { background: var(--danger-soft); color: var(--danger); border: 1px solid var(--danger-border); }
.oauth-text { font-size: var(--fs-md); color: var(--text-2); margin: 0; }
</style>
