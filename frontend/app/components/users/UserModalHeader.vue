<template>
  <div v-if="user" class="user-modal-header">
    <div class="user-header-left">
      <div class="user-avatar-circle" :class="{ 'is-anonymous': isAnonymousUser(user) }">
        <User v-if="isAnonymousUser(user)" :size="18" />
        <template v-else>{{ userInitials(user) }}</template>
      </div>
      <div class="user-header-info">
        <div class="user-name-row">
          <h2 class="user-header-name">{{ userPrimaryLabel(user) }}</h2>
          <span
            v-if="user.traffic_type"
            class="header-traffic-badge"
            :class="trafficTypeClass"
            :title="trafficTypeSummary"
          >
            <ShieldCheck v-if="user.traffic_type !== 'bot'" class="w-3.5 h-3.5" />
            <Bot v-else class="w-3.5 h-3.5" />
            <span>{{ trafficTypeLabel }}</span>
          </span>
        </div>
        <div class="user-sub-row">
          <span class="user-id-code">ID: {{ user.external_id }}</span>
          <button
            type="button"
            class="id-copy-action"
            :class="{ copied: copiedField === 'user_id' }"
            :title="copiedField === 'user_id' ? 'ID Copiado!' : 'Copiar ID do usuário'"
            @click="copyToClipboard(user.external_id, 'user_id')"
          >
            <Check v-if="copiedField === 'user_id'" class="w-3 h-3 copy-ok" />
            <Copy v-else class="w-3 h-3" />
            <span>{{ copiedField === 'user_id' ? 'Copiado' : 'Copiar' }}</span>
          </button>
          <span v-if="user.pixel_name" class="pixel-pill">
            <span class="pixel-dot" />
            {{ user.pixel_name }}
          </span>
        </div>
      </div>
    </div>
  </div>
  <h2 v-else class="user-loading-title">Carregando usuário...</h2>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount } from 'vue'
import { ShieldCheck, Bot, Copy, Check, User } from 'lucide-vue-next'
import { isAnonymousUser, userInitials, userPrimaryLabel } from '~/components/users/userFormatters'
import type { TrackedUser } from '#shared/types'

interface Props {
  user: TrackedUser | null
}

const props = defineProps<Props>()

const copiedField = ref<string | null>(null)
let copyTimer: ReturnType<typeof setTimeout> | null = null

const copyToClipboard = async (text: string | null | undefined, field: string) => {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    copiedField.value = field
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      if (copiedField.value === field) copiedField.value = null
    }, 2000)
  } catch {
    /* silent */
  }
}

onBeforeUnmount(() => {
  if (copyTimer) clearTimeout(copyTimer)
})

const trafficTypeClass = computed(() => {
  const t = props.user?.traffic_type
  if (t === 'bot') return 'is-bot'
  if (t === 'inconclusive') return 'is-inconclusive'
  return 'is-human'
})

const trafficTypeLabel = computed(() => {
  const t = props.user?.traffic_type
  if (t === 'bot') {
    if (props.user?.traffic_category === 'ad_verifier') return 'Crawler de Anúncio'
    if (props.user?.traffic_category === 'search_engine') return 'Crawler de Busca'
    if (props.user?.traffic_category === 'headless_scraper') return 'Automação / Headless'
    return 'Provável Bot'
  }
  if (t === 'inconclusive') return 'Inconclusivo'
  return 'Provavelmente Humano'
})

const trafficTypeSummary = computed(() => {
  const t = props.user?.traffic_type
  if (t === 'bot') return 'Identificados padrões característicos de requisições automatizadas ou robôs de plataformas.'
  if (t === 'inconclusive') return 'Dados de cabeçalho incompletos ou não padronizados para afirmação conclusiva.'
  return 'Navegador e sistema legítimos reconhecidos, sem traços de controle remoto.'
})
</script>

<style scoped>
.user-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.user-header-left {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  min-width: 0;
}

.user-avatar-circle {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  color: var(--brand);
  font-weight: var(--fw-bold);
  font-size: var(--fs-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  letter-spacing: 0.02em;
}

.user-avatar-circle.is-anonymous {
  background: var(--surface-soft);
  border-color: var(--border);
  color: var(--text-3);
}

.user-header-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.user-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.user-header-name {
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  color: var(--text-1);
  letter-spacing: -0.01em;
  margin: 0;
  line-height: 1.25;
}

.header-traffic-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
}
.header-traffic-badge.is-human { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.header-traffic-badge.is-bot { background: var(--danger-soft); color: var(--danger-light); border: 1px solid var(--danger-border); }
.header-traffic-badge.is-inconclusive { background: var(--warning-soft); color: var(--warning-light); border: 1px solid var(--warning-border); }
.copy-ok { color: var(--success); }
.user-loading-title { font-size: var(--fs-xl); font-weight: var(--fw-bold); color: var(--text-1); }

.user-sub-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  font-size: var(--fs-sm);
  color: var(--text-3);
}

.user-id-code {
  font-family: var(--font-mono);
  color: var(--text-2);
}

.id-copy-action {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.1rem 0.375rem;
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  border-radius: 4px;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  color: var(--text-3);
  cursor: pointer;
  transition: all 0.15s ease;
}

.id-copy-action:hover { border-color: var(--brand); color: var(--brand); }
.id-copy-action.copied { color: var(--success); border-color: var(--success); }
.pixel-pill { display: inline-flex; align-items: center; gap: 0.375rem; font-size: var(--fs-xs); font-weight: var(--fw-medium); color: var(--text-2); background: var(--surface-deep); padding: 0.1rem 0.4rem; border-radius: 6px; border: 1px solid var(--border-soft); }
.pixel-dot { width: 5px; height: 5px; border-radius: 9999px; background: var(--brand); }

@media (max-width: 640px) {
  .user-modal-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
