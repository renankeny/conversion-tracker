<template>
  <div class="settings-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Configurações</h1>
        <p class="page-subtitle">
          Gestão da sua conta, admins, base GeoIP e retenção de dados.
        </p>
      </div>
    </div>

    <div class="tabs">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        class="tab"
        :class="{ active: activeTab === tab.value }"
        @click="activeTab = tab.value"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Aba: Conta -->
    <section v-if="activeTab === 'account'" class="card">
      <h3 class="card-title">Trocar minha senha</h3>
      <p class="card-hint">
        Logado como <strong>{{ user?.email }}</strong>. A nova senha precisa ter no mínimo 6 caracteres.
      </p>
      <form class="form" @submit.prevent="onChangePassword">
        <BaseInput
          v-model="passwordForm.current"
          type="password"
          label="Senha atual"
          required
        />
        <BaseInput
          v-model="passwordForm.next"
          type="password"
          label="Nova senha"
          required
        />
        <BaseInput
          v-model="passwordForm.confirm"
          type="password"
          label="Confirmar nova senha"
          :error="passwordError"
          required
        />
        <div class="form-actions">
          <BaseButton
            variant="primary"
            type="submit"
            :loading="changingPassword"
          >
            Salvar nova senha
          </BaseButton>
        </div>
      </form>
    </section>

    <!-- Aba: Admins -->
    <section v-if="activeTab === 'admins'" class="admins-section">
      <div class="card">
        <div class="card-header">
          <h3 class="card-title">Administradores</h3>
          <BaseButton variant="primary" size="sm" @click="newAdminOpen = true">
            + Novo admin
          </BaseButton>
        </div>

        <div v-if="loadingAdmins" class="muted">Carregando...</div>

        <div v-else-if="!admins.length" class="muted">Nenhum admin cadastrado.</div>

        <div v-else class="admin-list">
          <div v-for="a in admins" :key="a.id" class="admin-row">
            <div class="admin-info">
              <span class="admin-email">{{ a.email }}</span>
              <span class="admin-meta">ID #{{ a.id }} · criado {{ formatDate(a.created_at) }}</span>
            </div>
            <BaseButton
              v-if="a.id !== user?.id"
              variant="ghost"
              size="sm"
              @click="askDelete(a)"
            >
              <span class="text-danger">Remover</span>
            </BaseButton>
            <BaseBadge v-else variant="info">Você</BaseBadge>
          </div>
        </div>
      </div>
    </section>

    <!-- Aba: GeoIP -->
    <section v-if="activeTab === 'geoip'" class="card">
      <h3 class="card-title">Base GeoIP</h3>
      <p class="card-hint">
        A base GeoLite2-City da MaxMind enriquece os <code>tracked_users</code> com
        país, estado, cidade e CEP. Sem ela o sistema funciona, só não tem geo.
      </p>

      <div v-if="loadingGeoIP" class="muted">Carregando...</div>

      <div v-else class="geoip-status">
        <div class="status-grid">
          <div class="status-item">
            <span class="status-label">Status</span>
            <BaseBadge :variant="geoipStatusVariant">
              {{ geoipStatusLabel }}
            </BaseBadge>
          </div>
          <div class="status-item">
            <span class="status-label">License Key</span>
            <BaseBadge :variant="geoip?.license_key_set ? 'success' : 'warning'">
              {{ geoip?.license_key_set ? 'Configurada' : 'Não configurada' }}
            </BaseBadge>
          </div>
          <div class="status-item">
            <span class="status-label">Path</span>
            <code>{{ geoip?.path || '—' }}</code>
          </div>
          <div v-if="geoip?.size_bytes" class="status-item">
            <span class="status-label">Tamanho</span>
            <span>{{ formatBytes(geoip.size_bytes) }}</span>
          </div>
          <div v-if="geoip?.modified_at" class="status-item">
            <span class="status-label">Atualizado em</span>
            <span>{{ formatDate(geoip.modified_at) }}</span>
          </div>
        </div>

        <div class="geoip-actions">
          <BaseButton
            variant="primary"
            :loading="refreshingGeoIP"
            :disabled="!geoip?.license_key_set"
            @click="onRefreshGeoIP"
          >
            Baixar / atualizar base agora
          </BaseButton>
          <BaseButton variant="outline" size="sm" @click="loadGeoIP">
            Recarregar status
          </BaseButton>
        </div>

        <p v-if="!geoip?.license_key_set" class="hint-warning">
          ⚠️ Configure <code>MAXMIND_LICENSE_KEY</code> nas envs do servidor.
          A chave gratuita é obtida em
          <a href="https://www.maxmind.com/en/geolite2/signup" target="_blank" rel="noopener">
            maxmind.com/en/geolite2/signup
          </a>.
        </p>
      </div>
    </section>

    <!-- Aba: Retenção -->
    <section v-if="activeTab === 'retention'" class="card">
      <h3 class="card-title">Retenção de dados</h3>
      <p class="card-hint">
        Os registros são apagados automaticamente todo dia às 03:00 (horário São Paulo).
        Defina via env <code>SENT_EVENTS_RETENTION_DAYS</code> e
        <code>EVENT_COUNTERS_RETENTION_DAYS</code>.
      </p>

      <div v-if="!maintenanceData" class="muted">Carregando...</div>

      <div v-else class="retention-grid">
        <div class="retention-item">
          <span class="retention-label">Eventos completos (sent_events)</span>
          <span class="retention-value">
            {{ maintenanceData.sent_events_retention_days || '∞' }}
            <small v-if="maintenanceData.sent_events_retention_days">dias</small>
          </span>
        </div>
        <div class="retention-item">
          <span class="retention-label">Agregados (event_counters)</span>
          <span class="retention-value">
            {{ maintenanceData.event_counters_retention_days || '∞' }}
            <small v-if="maintenanceData.event_counters_retention_days">dias</small>
          </span>
        </div>
      </div>

      <div class="retention-actions">
        <BaseButton variant="danger" :loading="pruning" @click="confirmPrune = true">
          Executar limpeza agora
        </BaseButton>
      </div>

      <div v-if="lastPruneResults.length" class="prune-results">
        <h4>Resultado da última limpeza:</h4>
        <div v-for="r in lastPruneResults" :key="r.table" class="prune-row">
          <code>{{ r.table }}</code>
          <span>
            <strong>{{ formatNumber(r.deleted) }}</strong> rows apagadas
            <small v-if="r.retention_days">({{ r.retention_days }} dias)</small>
          </span>
        </div>
      </div>
    </section>

    <!-- Modais -->
    <BaseModal v-model="newAdminOpen" title="Criar novo administrador" size="sm">
      <div class="form">
        <BaseInput
          v-model="newAdminForm.email"
          type="email"
          label="E-mail"
          placeholder="admin@trackfind.com.br"
          required
        />
        <BaseInput
          v-model="newAdminForm.password"
          type="password"
          label="Senha (mínimo 6 caracteres)"
          required
        />
      </div>
      <template #footer>
        <BaseButton variant="ghost" @click="newAdminOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="creatingAdmin" @click="onCreateAdmin">
          Criar
        </BaseButton>
      </template>
    </BaseModal>

    <ConfirmModal
      v-model="deleteAdminOpen"
      title="Remover administrador?"
      :message="deletingAdmin
        ? `O admin '${deletingAdmin.email}' perderá acesso ao painel imediatamente.`
        : ''"
      confirm-label="Remover"
      variant="danger"
      :loading="deletingNow"
      @confirm="onConfirmDeleteAdmin"
    />

    <ConfirmModal
      v-model="confirmPrune"
      title="Executar limpeza agora?"
      message="Vai apagar todos os registros mais antigos que a retenção configurada. Operação irreversível."
      confirm-label="Sim, limpar agora"
      variant="danger"
      :loading="pruning"
      @confirm="onPrune"
    />
  </div>
</template>

<script setup lang="ts">
import type {
  AdminUser,
  GeoIPStatus,
  MaintenanceStatusResponse,
  PruneResultRow,
} from '#shared/types'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

useHead({ title: 'Configurações - TrackFind' })

const settings = useSettings()
const { user } = useAuth()
const toast = useToast()

const tabs = [
  { value: 'account', label: 'Conta' },
  { value: 'admins', label: 'Administradores' },
  { value: 'geoip', label: 'GeoIP' },
  { value: 'retention', label: 'Retenção' },
] as const

type TabValue = (typeof tabs)[number]['value']

const activeTab = ref<TabValue>('account')

// === Senha ===
const passwordForm = reactive({
  current: '',
  next: '',
  confirm: '',
})
const passwordError = ref('')
const changingPassword = ref(false)

const onChangePassword = async () => {
  passwordError.value = ''
  if (passwordForm.next !== passwordForm.confirm) {
    passwordError.value = 'As senhas não conferem'
    return
  }
  if (passwordForm.next.length < 6) {
    passwordError.value = 'Mínimo 6 caracteres'
    return
  }
  changingPassword.value = true
  try {
    await settings.changeMyPassword({
      current_password: passwordForm.current,
      new_password: passwordForm.next,
    })
    toast.success('Senha atualizada', 'Use a nova no próximo login')
    passwordForm.current = ''
    passwordForm.next = ''
    passwordForm.confirm = ''
  } catch (error: any) {
    toast.error('Erro', error.message || 'Não foi possível alterar a senha')
  } finally {
    changingPassword.value = false
  }
}

// === Admins ===
const admins = ref<AdminUser[]>([])
const loadingAdmins = ref(false)
const newAdminOpen = ref(false)
const newAdminForm = reactive({ email: '', password: '' })
const creatingAdmin = ref(false)

const deleteAdminOpen = ref(false)
const deletingAdmin = ref<AdminUser | null>(null)
const deletingNow = ref(false)

const loadAdmins = async () => {
  loadingAdmins.value = true
  try {
    const res = await settings.listAdmins()
    admins.value = res.data || []
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao listar admins')
  } finally {
    loadingAdmins.value = false
  }
}

const onCreateAdmin = async () => {
  creatingAdmin.value = true
  try {
    const created = await settings.createAdmin({
      email: newAdminForm.email.trim().toLowerCase(),
      password: newAdminForm.password,
    })
    admins.value = [...admins.value, created]
    toast.success('Admin criado', `${created.email} já pode logar`)
    newAdminOpen.value = false
    newAdminForm.email = ''
    newAdminForm.password = ''
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao criar admin')
  } finally {
    creatingAdmin.value = false
  }
}

const askDelete = (admin: AdminUser) => {
  deletingAdmin.value = admin
  deleteAdminOpen.value = true
}

const onConfirmDeleteAdmin = async () => {
  if (!deletingAdmin.value) return
  deletingNow.value = true
  try {
    await settings.deleteAdmin(deletingAdmin.value.id)
    admins.value = admins.value.filter((a) => a.id !== deletingAdmin.value!.id)
    toast.success('Admin removido')
    deleteAdminOpen.value = false
    deletingAdmin.value = null
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao remover')
  } finally {
    deletingNow.value = false
  }
}

// === GeoIP ===
const geoip = ref<GeoIPStatus | null>(null)
const loadingGeoIP = ref(false)
const refreshingGeoIP = ref(false)

const geoipStatusVariant = computed(() => {
  if (!geoip.value) return 'default'
  if (geoip.value.loaded) return 'success'
  if (geoip.value.exists) return 'warning'
  return 'danger'
})

const geoipStatusLabel = computed(() => {
  if (!geoip.value) return ''
  if (geoip.value.loaded) return 'Carregada'
  if (geoip.value.exists) return 'Arquivo existe mas não carregou'
  return 'Não instalada'
})

const loadGeoIP = async () => {
  loadingGeoIP.value = true
  try {
    geoip.value = await settings.geoipStatus()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao consultar GeoIP')
  } finally {
    loadingGeoIP.value = false
  }
}

const onRefreshGeoIP = async () => {
  refreshingGeoIP.value = true
  try {
    const res = await settings.geoipRefresh()
    if (res.installed) {
      toast.success(
        'GeoIP atualizada',
        res.reloaded ? 'Base recarregada com sucesso' : 'Base salva, mas não recarregada'
      )
    }
    await loadGeoIP()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao baixar GeoIP')
  } finally {
    refreshingGeoIP.value = false
  }
}

// === Retenção ===
const maintenanceData = ref<MaintenanceStatusResponse | null>(null)
const pruning = ref(false)
const confirmPrune = ref(false)
const lastPruneResults = ref<PruneResultRow[]>([])

const loadMaintenance = async () => {
  try {
    maintenanceData.value = await settings.maintenanceStatus()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao consultar retenção')
  }
}

const onPrune = async () => {
  pruning.value = true
  try {
    const res = await settings.pruneNow()
    lastPruneResults.value = res.results || []
    toast.success('Limpeza executada', 'Veja resultado abaixo')
    confirmPrune.value = false
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao executar limpeza')
  } finally {
    pruning.value = false
  }
}

// === Helpers ===
const formatDate = (iso: string): string => {
  if (!iso) return ''
  return new Date(iso).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const formatBytes = (bytes: number): string => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(1)} ${units[i]}`
}

const formatNumber = (n: number): string => new Intl.NumberFormat('pt-BR').format(n)

// === Lifecycle ===
watch(activeTab, (tab) => {
  if (tab === 'admins') loadAdmins()
  if (tab === 'geoip') loadGeoIP()
  if (tab === 'retention') loadMaintenance()
})
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
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

.tabs {
  display: flex;
  gap: 0.25rem;
  border-bottom: 1px solid var(--border);
}

.tab {
  padding: 0.75rem 1.25rem;
  background: transparent;
  border: none;
  color: var(--text-2);
  font-size: 0.9375rem;
  font-weight: 500;
  cursor: pointer;
  position: relative;
  transition: color 0.15s;
}

.tab:hover {
  color: var(--text-1);
}

.tab.active {
  color: var(--brand);
}

.tab.active::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--brand);
}

.card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: var(--shadow-card);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.card-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-strong);
  margin-bottom: 0.5rem;
}

.card-hint {
  font-size: 0.875rem;
  color: var(--text-2);
  margin-bottom: 1rem;
  line-height: 1.5;
}

.card-hint code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  max-width: 420px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.5rem;
}

.muted {
  color: var(--text-3);
  font-size: 0.875rem;
  font-style: italic;
}

.text-danger {
  color: var(--danger);
}

.admin-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.admin-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  background: var(--surface-soft-2);
}

.admin-info {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.admin-email {
  font-weight: 600;
  color: var(--text-1);
}

.admin-meta {
  font-size: 0.75rem;
  color: var(--text-3);
}

.geoip-status {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.status-grid,
.retention-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem 1rem;
}

.status-item,
.retention-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.status-label,
.retention-label {
  font-size: 0.6875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-3);
  font-weight: 600;
}

.status-item code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  background: var(--surface-soft);
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  word-break: break-all;
}

.retention-value {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-strong);
}

.retention-value small {
  font-size: 0.75rem;
  font-weight: 400;
  color: var(--text-3);
}

.geoip-actions,
.retention-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-top: 0.5rem;
}

.hint-warning {
  font-size: 0.8125rem;
  color: var(--warning);
  background: var(--warning-soft);
  border: 1px solid var(--warning);
  border-radius: 8px;
  padding: 0.625rem 0.875rem;
}

.hint-warning code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: rgba(0, 0, 0, 0.2);
  padding: 0 0.25rem;
  border-radius: 3px;
}

.hint-warning a {
  color: var(--warning);
  text-decoration: underline;
}

.prune-results {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-soft);
}

.prune-results h4 {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-strong);
  margin-bottom: 0.5rem;
}

.prune-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0;
  font-size: 0.875rem;
  border-bottom: 1px solid var(--border-soft);
}

.prune-row:last-child {
  border-bottom: none;
}

.prune-row code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-soft);
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
}
</style>
