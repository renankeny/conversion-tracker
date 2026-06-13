<template>
  <div class="dashboard-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Dashboard</h1>
        <p class="page-subtitle">Visão geral dos seus eventos de conversão</p>
      </div>
      <BaseButton variant="outline" size="sm" @click="handleRefresh" :loading="loading">
        Atualizar
      </BaseButton>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path d="M2 10a8 8 0 018-8v8h8a8 8 0 11-16 0z" />
            <path d="M12 2.252A8.014 8.014 0 0117.748 8H12V2.252z" />
          </svg>
        </div>
        <div>
          <p class="stat-label">Total de Eventos</p>
          <p class="stat-value">{{ stats?.total_events ?? 0 }}</p>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
          </svg>
        </div>
        <div>
          <p class="stat-label">Enviados</p>
          <p class="stat-value text-success">{{ stats?.status_totals.sent ?? 0 }}</p>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clip-rule="evenodd" />
          </svg>
        </div>
        <div>
          <p class="stat-label">Pendentes</p>
          <p class="stat-value text-warning">{{ stats?.status_totals.pending ?? 0 }}</p>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
          </svg>
        </div>
        <div>
          <p class="stat-label">Falhas</p>
          <p class="stat-value text-danger">{{ stats?.status_totals.failed ?? 0 }}</p>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon stat-icon-info">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path d="M9 6a3 3 0 11-6 0 3 3 0 016 0zM17 6a3 3 0 11-6 0 3 3 0 016 0zM12.93 17c.046-.327.07-.66.07-1a6.97 6.97 0 00-1.5-4.33A5 5 0 0119 16v1h-6.07zM6 11a5 5 0 015 5v1H1v-1a5 5 0 015-5z" />
          </svg>
        </div>
        <div>
          <p class="stat-label">Usuários Rastreados</p>
          <p class="stat-value">{{ stats?.tracked_users ?? 0 }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DashboardStats } from '#shared/types'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

const api = useAPI()
const toast = useToast()

const stats = ref<DashboardStats | null>(null)
const loading = ref(false)

const fetchStats = async () => {
  loading.value = true
  try {
    stats.value = await api.get<DashboardStats>('/admin/stats')
  } catch (error: any) {
    toast.error('Erro', error.message || 'Não foi possível carregar as estatísticas')
  } finally {
    loading.value = false
  }
}

const handleRefresh = () => {
  fetchStats()
}

onMounted(() => {
  fetchStats()
})
</script>

<style scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: wrap;
}

.page-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.page-subtitle {
  color: var(--text-2);
  font-size: 0.9375rem;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  box-shadow: var(--shadow-card);
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 10px;
  flex-shrink: 0;
}

.stat-icon-primary {
  background: var(--brand-soft);
  color: var(--brand);
}

.stat-icon-success {
  background: var(--success-soft);
  color: var(--success);
}

.stat-icon-warning {
  background: var(--warning-soft);
  color: var(--warning);
}

.stat-icon-danger {
  background: var(--danger-soft);
  color: var(--danger);
}

.stat-icon-info {
  background: var(--info-soft);
  color: var(--info);
}

.stat-label {
  font-size: 0.8125rem;
  color: var(--text-2);
  margin-bottom: 0.25rem;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-strong);
}

.text-success {
  color: var(--success);
}

.text-warning {
  color: var(--warning);
}

.text-danger {
  color: var(--danger);
}
</style>
