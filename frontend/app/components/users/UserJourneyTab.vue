<template>
  <div class="user-journey-tab">
    <!-- Auditoria de Tráfego Heurística -->
    <UserTrafficAuditBox :user="user" />

    <!-- Histórico de Compras Aprovadas -->
    <section class="section">
      <div class="section-header-row">
        <h4 class="section-title">Histórico de Compras Aprovadas ({{ purchaseHistory.length }})</h4>
        <span v-if="purchaseHistory.length && user" class="text-xs text-success font-semibold">
          Total: {{ formatCurrency(user.ltv) }}
        </span>
      </div>

      <div v-if="loadingPurchases" class="muted py-2">Carregando compras…</div>
      <div v-else-if="!purchaseHistory.length" class="empty-hint-card">
        Nenhuma compra aprovada registrada para este perfil.
      </div>
      <div v-else class="purchase-history">
        <div v-for="p in purchaseHistory" :key="p.event_id" class="purchase-history-row">
          <span class="purchase-history-date">{{ formatDate(p.created_at) }}</span>
          <span class="purchase-history-pixel">{{ p.pixel_name }}</span>
          <span class="purchase-history-value">{{ formatCurrency(p.purchase_value) }}</span>
        </div>
      </div>
    </section>

    <!-- Jornada do Cliente (Linha do Tempo Completa) -->
    <section class="section">
      <div class="section-header-row">
        <div class="flex items-center gap-2">
          <h4 class="section-title !mb-0">Jornada do Cliente (Linha do Tempo)</h4>
          <span
            v-if="journeyData && journeyData.sessions_count > 1"
            class="px-2 py-0.5 rounded-full text-[10px] font-bold bg-pink-500/15 text-pink-400 border border-pink-500/30"
            title="Identidade unificada em múltiplos dispositivos"
          >
            {{ journeyData.sessions_count }} dispositivos unificados
          </span>
        </div>
        <span class="text-xs" style="color: var(--text-3)">{{ timelineEvents.length }} marcos identificados</span>
      </div>

      <UserJourneyTimeline
        :timeline-events="timelineEvents"
        :loading-timeline="loadingTimeline"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import UserTrafficAuditBox from '~/components/users/UserTrafficAuditBox.vue'
import UserJourneyTimeline, { type TimelineItem } from '~/components/users/UserJourneyTimeline.vue'
import type { GroupedEvent, TrackedUser, CustomerJourneyResponse } from '#shared/types'

interface Props {
  user: TrackedUser | null
  purchaseHistory: GroupedEvent[]
  loadingPurchases: boolean
  journeyData: CustomerJourneyResponse | null
  timelineEvents: TimelineItem[]
  loadingTimeline: boolean
}

defineProps<Props>()

const formatCurrency = (val: number | undefined): string => {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(val || 0)
}

const formatDate = (iso: string): string => {
  if (!iso) return ''
  return new Date(iso).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
</script>

<style scoped>
.user-journey-tab {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.section-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 0.4375rem;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0;
}

.empty-hint-card {
  font-size: var(--fs-sm);
  color: var(--text-3);
  font-style: italic;
  padding: 1.25rem;
  text-align: center;
  background: var(--surface-soft-2);
  border: 1px dashed var(--border-soft);
  border-radius: 8px;
}

.muted {
  color: var(--text-3);
  font-style: italic;
  font-size: var(--fs-sm);
}

/* Histórico de Compras */
.purchase-history {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  max-height: 220px;
  overflow-y: auto;
}

.purchase-history-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  border-radius: 8px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  font-size: var(--fs-base);
}

.purchase-history-date {
  color: var(--text-2);
  white-space: nowrap;
  font-size: var(--fs-sm);
}

.purchase-history-pixel {
  color: var(--text-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  font-size: var(--fs-sm);
}

.purchase-history-value {
  color: var(--success);
  font-weight: var(--fw-semibold);
  white-space: nowrap;
  font-size: var(--fs-md);
}
</style>
