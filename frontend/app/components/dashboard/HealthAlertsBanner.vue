<template>
  <div v-if="visibleAlerts.length" class="health-alerts-container">
    <div
      v-for="alert in visibleAlerts"
      :key="alert.id"
      class="health-alert-card"
      :class="`severity-${alert.severity}`"
    >
      <div class="alert-icon-wrap">
        <svg v-if="alert.severity === 'danger'" class="alert-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <svg v-else-if="alert.severity === 'warning'" class="alert-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" /><line x1="12" y1="9" x2="12" y2="13" /><line x1="12" y1="17" x2="12.01" y2="17" />
        </svg>
        <svg v-else class="alert-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" />
        </svg>
      </div>

      <div class="alert-content">
        <div class="alert-header">
          <span class="alert-title">{{ alert.title }}</span>
          <span class="alert-badge">{{ alert.severity === 'danger' ? 'Crítico' : 'Atenção' }}</span>
        </div>
        <p class="alert-message">{{ alert.message }}</p>
      </div>

      <div class="alert-actions">
        <NuxtLink v-if="alert.action_url" :to="alert.action_url" class="alert-btn-action">
          {{ alert.action_text || 'Resolver' }}
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M5 12h14" /><path d="m12 5 7 7-7 7" />
          </svg>
        </NuxtLink>

        <button type="button" class="alert-btn-dismiss" title="Dispensar aviso" @click="dismissAlert(alert.id)">
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useSystemAlertsStore } from '~/stores/useSystemAlertsStore'
import { useTenantStore } from '~/stores/useTenantStore'

const alertsStore = useSystemAlertsStore()
const tenantStore = useTenantStore()
const { visibleAlerts } = storeToRefs(alertsStore)
const { dismissAlert } = alertsStore

watch(() => tenantStore.switchVersion, () => {
  alertsStore.fetchAlerts(true)
})

onMounted(() => {
  alertsStore.fetchAlerts()
})
</script>

<style scoped>
.health-alerts-container {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
}

.health-alert-card {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  padding: 0.75rem 1rem;
  border-radius: 8px;
  border: 1px solid var(--border-soft);
  backdrop-filter: blur(8px);
  transition: all 0.2s ease;
}

.health-alert-card.severity-danger { background: rgba(239, 68, 68, 0.08); border-color: rgba(239, 68, 68, 0.25); }
.health-alert-card.severity-warning { background: rgba(245, 158, 11, 0.08); border-color: rgba(245, 158, 11, 0.25); }
.health-alert-card.severity-info { background: rgba(56, 189, 248, 0.08); border-color: rgba(56, 189, 248, 0.25); }

.alert-icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.alert-icon { width: 1.25rem; height: 1.25rem; }
.severity-danger .alert-icon { color: var(--danger); }
.severity-warning .alert-icon { color: var(--warning); }
.severity-info .alert-icon { color: var(--info); }

.alert-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.alert-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.alert-title {
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
}

.alert-badge {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
}

.severity-danger .alert-badge { background: rgba(239, 68, 68, 0.2); color: var(--danger); }
.severity-warning .alert-badge { background: rgba(245, 158, 11, 0.2); color: var(--warning); }
.severity-info .alert-badge { background: rgba(56, 189, 248, 0.2); color: var(--info); }

.alert-message {
  font-size: var(--fs-sm);
  color: var(--text-2);
  margin: 0;
  line-height: 1.4;
}

.alert-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.alert-btn-action {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  padding: 0.3125rem 0.625rem;
  border-radius: 6px;
  text-decoration: none;
  background: var(--surface-deep);
  color: var(--text-1);
  border: 1px solid var(--border);
  transition: all 0.15s ease;
}

.alert-btn-action:hover {
  border-color: var(--brand);
  color: var(--brand);
}

.alert-btn-dismiss {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.25rem;
  border-radius: 6px;
  background: transparent;
  border: none;
  color: var(--text-3);
  cursor: pointer;
  transition: color 0.15s ease;
}

.alert-btn-dismiss:hover {
  color: var(--text-1);
}
</style>
