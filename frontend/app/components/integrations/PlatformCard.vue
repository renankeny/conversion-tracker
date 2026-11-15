<template>
  <div class="platform-card" @click="emit('configure', platform)">
    <div class="card-body">
      <div class="platform-header">
        <div class="platform-logo">
          <img v-if="platform.logoUrl" :src="platform.logoUrl" :alt="platform.name" loading="lazy" />
          <span v-else>{{ platform.name.charAt(0) }}</span>
        </div>

        <div class="platform-info">
          <div class="title-line">
            <span class="platform-name">{{ platform.name }}</span>
            <BaseBadge v-if="isActive" variant="success">Ativa</BaseBadge>
            <BaseBadge v-else-if="hasOnlyFailures" variant="warning">Falhas</BaseBadge>
          </div>
          <p class="platform-desc">{{ platform.description }}</p>
        </div>
      </div>

      <div class="meta-strip">
        <span v-if="summary?.has_token" class="meta-item meta-protected">
          <ShieldCheck :size="13" />
          Token ativo
        </span>
        <span v-if="summary && summary.total_accepted > 0" class="meta-item">
          {{ summary.total_accepted }} {{ summary.total_accepted === 1 ? 'venda' : 'vendas' }}
        </span>
        <span v-if="summary?.last_received_at" class="meta-item meta-muted">
          Último {{ formatShortDateTimeSP(summary.last_received_at) }}
        </span>
        <span v-if="!summary?.has_token && !summary?.total_accepted && !summary?.total_failed" class="webhook-pill">
          {{ platform.webhookPath }}
        </span>
        <span v-else-if="summary && summary.total_failed > 0" class="meta-item meta-failed">
          {{ summary.total_failed }} {{ summary.total_failed === 1 ? 'falha' : 'falhas' }}
        </span>
      </div>
    </div>

    <div class="card-footer">
      <BaseButton variant="ghost" size="sm" @click.stop="emit('copy', platform)">
        <template #iconLeft>
          <Check v-if="copied" :size="14" />
          <Copy v-else :size="14" />
        </template>
        {{ copied ? 'Copiada' : 'Copiar URL' }}
      </BaseButton>

      <BaseButton variant="primary" size="sm" @click.stop="emit('configure', platform)">
        {{ canManage ? 'Configurar' : 'Instruções' }}
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Check, Copy, ShieldCheck } from 'lucide-vue-next'
import type { PlatformDef } from '#shared/constants/platforms'
import type { PlatformSummaryItem } from '#shared/types'
import { formatShortDateTimeSP } from '~/utils/formatDate'

interface Props {
  platform: PlatformDef
  summary?: PlatformSummaryItem
  copied: boolean
  canManage: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  copy: [platform: PlatformDef]
  configure: [platform: PlatformDef]
}>()

const isActive = computed(() => Boolean(props.summary && (props.summary.has_token || props.summary.total_accepted > 0)))
const hasOnlyFailures = computed(() => Boolean(props.summary && !isActive.value && props.summary.total_failed > 0))
</script>

<style scoped>
.platform-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 1.125rem;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 0.875rem;
  box-shadow: var(--shadow-card);
  transition: border-color 0.18s ease, transform 0.18s ease;
  cursor: pointer;
}
.platform-card:hover {
  transform: translateY(-2px);
  border-color: var(--brand-border);
}
.card-body {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.platform-header {
  display: flex;
  gap: 0.875rem;
  align-items: flex-start;
}
.platform-logo {
  flex-shrink: 0;
  width: 46px;
  height: 46px;
  border-radius: 11px;
  background: var(--surface-1);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-heading);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  overflow: hidden;
  border: 1px solid var(--border);
}
.platform-logo img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.platform-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.title-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}
.platform-name {
  font-family: var(--font-heading);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.platform-desc {
  font-family: var(--font-sans);
  font-size: var(--fs-base);
  color: var(--text-2);
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.meta-strip {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  min-height: 22px;
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  color: var(--text-2);
  background: var(--surface-soft);
  padding: 0.175rem 0.45rem;
  border-radius: 6px;
}
.meta-protected {
  color: var(--success);
}
.meta-muted {
  color: var(--text-3);
}
.meta-failed {
  color: var(--warning);
}
.webhook-pill {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  background: var(--surface-soft);
  padding: 0.175rem 0.5rem;
  border-radius: 6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border-soft);
}
</style>
