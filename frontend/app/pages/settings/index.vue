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

    <!-- Aba: Contas (somente owners) -->
    <section v-if="activeTab === 'accounts'" class="accounts-section">
      <div class="card">
        <div class="card-header">
          <div>
            <h3 class="card-title">Contas</h3>
            <p class="card-hint">
              Cada conta isola pixels, eventos e usuários rastreados. Members só
              enxergam dados da própria conta. Owners enxergam tudo.
            </p>
          </div>
          <BaseButton variant="primary" size="sm" @click="newAccountOpen = true">
            + Nova conta
          </BaseButton>
        </div>

        <div v-if="loadingAccounts" class="muted">Carregando...</div>

        <div v-else-if="!accounts.length" class="muted">Nenhuma conta cadastrada.</div>

        <div v-else class="account-list">
          <div v-for="a in accounts" :key="a.id" class="account-row">
            <div class="account-info">
              <div class="account-name">
                {{ a.name }}
                <BaseBadge v-if="!a.enabled" variant="warning">desabilitada</BaseBadge>
              </div>
              <span class="account-meta">
                ID #{{ a.id }} · slug <code>{{ a.slug }}</code> ·
                criada {{ formatDate(a.created_at) }}
              </span>
            </div>
            <div class="account-actions">
              <BaseButton variant="ghost" size="sm" @click="editAccount(a)">
                Editar
              </BaseButton>
              <BaseButton
                v-if="a.id !== 1"
                variant="ghost"
                size="sm"
                @click="askDeleteAccount(a)"
              >
                <span class="text-danger">Remover</span>
              </BaseButton>
            </div>
          </div>
        </div>
      </div>
    </section>

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

    <!-- Aba: API Keys -->
    <section v-if="activeTab === 'api-keys'" class="api-keys-section">
      <div class="card">
        <div class="card-header">
          <div>
            <h3 class="card-title">API Keys</h3>
            <p class="card-hint">
              Tokens server-to-server pra integrar o TrackFind com seu backend.
              Cada key é vinculada a uma conta e atua como member dela.
              <strong>O segredo aparece uma única vez na criação</strong> — copie e guarde.
            </p>
          </div>
          <BaseButton variant="primary" size="sm" @click="newKeyOpen = true">
            + Nova key
          </BaseButton>
        </div>

        <div v-if="loadingKeys" class="muted">Carregando...</div>

        <div v-else-if="!apiKeys.length" class="muted">Nenhuma API key cadastrada.</div>

        <div v-else class="account-list">
          <div v-for="k in apiKeys" :key="k.id" class="account-row">
            <div class="account-info">
              <div class="account-name">
                {{ k.name }}
                <BaseBadge v-if="k.revoked_at" variant="danger">revogada</BaseBadge>
                <BaseBadge v-else variant="success">ativa</BaseBadge>
              </div>
              <span class="account-meta">
                <code>{{ k.key_prefix }}...</code> · conta #{{ k.account_id }} ·
                criada {{ formatDate(k.created_at) }}
                <span v-if="k.last_used_at"> · usada {{ formatDate(k.last_used_at) }}</span>
                <span v-else> · nunca usada</span>
              </span>
            </div>
            <div class="account-actions">
              <BaseButton
                v-if="!k.revoked_at"
                variant="ghost"
                size="sm"
                @click="askRevokeKey(k)"
              >
                <span class="text-danger">Revogar</span>
              </BaseButton>
            </div>
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
        Você pode definir um override por conta abaixo. Use <strong>0</strong> pra desativar
        a retenção (mantém para sempre) ou deixe vazio pra usar o default global.
      </p>

      <div v-if="!maintenanceData" class="muted">Carregando...</div>

      <div v-else class="retention-grid">
        <div class="retention-item">
          <span class="retention-label">Eventos completos (sent_events)</span>
          <span class="retention-value">
            {{ maintenanceData.effective_sent_events_retention_days
              ?? maintenanceData.sent_events_retention_days
              ?? '∞' }}
            <small>dias</small>
          </span>
          <small v-if="maintenanceData.account_sent_events_retention_days != null" class="retention-source">
            override da conta
          </small>
          <small v-else class="retention-source">default global</small>
        </div>
        <div class="retention-item">
          <span class="retention-label">Agregados (event_counters)</span>
          <span class="retention-value">
            {{ maintenanceData.effective_event_counters_retention_days
              ?? maintenanceData.event_counters_retention_days
              ?? '∞' }}
            <small>dias</small>
          </span>
          <small v-if="maintenanceData.account_event_counters_retention_days != null" class="retention-source">
            override da conta
          </small>
          <small v-else class="retention-source">default global</small>
        </div>
      </div>

      <div class="retention-form">
        <h4 class="card-subtitle">Ajustar retenção desta conta</h4>
        <div class="form-row">
          <BaseInput
            v-model.number="retentionForm.sent_events"
            type="number"
            label="sent_events (dias)"
            placeholder="vazio = usa default global"
            hint="0 = nunca apaga; >0 = mantém os últimos N dias"
            min="0"
          />
          <BaseInput
            v-model.number="retentionForm.event_counters"
            type="number"
            label="event_counters (dias)"
            placeholder="vazio = usa default global"
            hint="0 = nunca apaga; >0 = mantém os últimos N dias"
            min="0"
          />
        </div>
        <div class="retention-actions">
          <BaseButton
            variant="primary"
            :loading="savingRetention"
            @click="onSaveRetention"
          >
            Salvar retenção
          </BaseButton>
          <BaseButton
            variant="ghost"
            :loading="savingRetention"
            @click="onResetRetention"
          >
            Voltar pro default global
          </BaseButton>
        </div>
      </div>

      <div class="retention-actions">
        <BaseButton variant="outline" :loading="pruning" @click="confirmPrune = true">
          Executar limpeza agora
        </BaseButton>
        <BaseButton variant="danger" :loading="resetting" @click="confirmReset = true">
          ⚠️ Resetar tudo desta conta
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

      <div v-if="lastResetResult" class="prune-results">
        <h4>Reset executado:</h4>
        <div class="prune-row">
          <code>sent_events</code>
          <span><strong>{{ formatNumber(lastResetResult.deleted_sent_events) }}</strong> rows apagadas</span>
        </div>
        <div class="prune-row">
          <code>event_counters</code>
          <span><strong>{{ formatNumber(lastResetResult.deleted_event_counters) }}</strong> rows apagadas</span>
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
        <div v-if="isOwner" class="form-field">
          <label class="form-label">Tipo de acesso</label>
          <select v-model="newAdminForm.role" class="form-select">
            <option value="owner">Owner — acesso total a todas as contas</option>
            <option value="member">Member — restrito a uma conta</option>
          </select>
        </div>
        <div v-if="isOwner && newAdminForm.role === 'member'" class="form-field">
          <label class="form-label">Conta vinculada</label>
          <select v-model="newAdminForm.account_id" class="form-select">
            <option :value="null" disabled>Selecione...</option>
            <option v-for="a in accounts" :key="a.id" :value="a.id">
              {{ a.name }} (#{{ a.id }})
            </option>
          </select>
        </div>
      </div>
      <template #footer>
        <BaseButton variant="ghost" @click="newAdminOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="creatingAdmin" @click="onCreateAdmin">
          Criar
        </BaseButton>
      </template>
    </BaseModal>

    <BaseModal v-model="newAccountOpen" :title="editingAccount ? 'Editar conta' : 'Nova conta'" size="sm">
      <div class="form">
        <BaseInput
          v-model="accountForm.name"
          label="Nome"
          placeholder="Conta do Cliente"
          required
        />
        <BaseInput
          v-model="accountForm.slug"
          label="Slug"
          placeholder="cliente-foo"
          hint="Apenas a-z, 0-9 e -. Auto-gerado a partir do nome se vazio."
        />
        <label class="checkbox-row">
          <input v-model="accountForm.enabled" type="checkbox" />
          <span>Conta habilitada</span>
        </label>
      </div>
      <template #footer>
        <BaseButton variant="ghost" @click="closeAccountModal">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="savingAccount" @click="onSaveAccount">
          {{ editingAccount ? 'Salvar' : 'Criar' }}
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

    <ConfirmModal
      v-model="confirmReset"
      title="Resetar TUDO desta conta?"
      message="Vai apagar TODOS os sent_events e event_counters desta conta — independente da idade. Histórico de envios à Meta CAPI, contadores de PageView, conversões — tudo. Pixels, tracked_users e configurações ficam intactos. Operação IRREVERSÍVEL."
      confirm-label="Sim, apagar tudo"
      variant="danger"
      :loading="resetting"
      @confirm="onReset"
    />

    <BaseModal v-model="newKeyOpen" title="Criar API Key" size="sm">
      <div class="form">
        <BaseInput
          v-model="keyForm.name"
          label="Nome da key"
          placeholder="Integração CRM, Bot WhatsApp, etc"
          hint="Apenas pra você se localizar no painel."
          required
        />
        <div v-if="isOwner" class="form-field">
          <label class="form-label">Conta vinculada</label>
          <select v-model="keyForm.account_id" class="form-select">
            <option v-for="a in accounts" :key="a.id" :value="a.id">
              {{ a.name }} (#{{ a.id }})
            </option>
          </select>
        </div>
      </div>
      <template #footer>
        <BaseButton variant="ghost" @click="newKeyOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="creatingKey" @click="onCreateKey">
          Criar
        </BaseButton>
      </template>
    </BaseModal>

    <BaseModal v-model="newKeyResultOpen" title="API Key criada" size="sm" :close-on-backdrop="false">
      <div class="form">
        <p class="card-hint">
          <strong>Copie agora.</strong> O segredo não aparece de novo em parte alguma.
          Ao fechar este modal, fica registrado apenas o prefixo no painel.
        </p>
        <div class="key-secret">
          <code>{{ createdKeyPlaintext }}</code>
          <BaseButton size="sm" variant="ghost" @click="copyKey">Copiar</BaseButton>
        </div>
        <p class="card-hint">
          Use no header HTTP:
          <code>Authorization: Bearer {{ createdKeyPlaintext || 'trk_xxx' }}</code>
        </p>
      </div>
      <template #footer>
        <BaseButton variant="primary" @click="closeKeyResult">Entendi, fechar</BaseButton>
      </template>
    </BaseModal>

    <ConfirmModal
      v-model="revokeKeyOpen"
      title="Revogar API Key?"
      :message="revokingKey
        ? `A key '${revokingKey.name}' (${revokingKey.key_prefix}...) deixará de funcionar imediatamente. Operação irreversível.`
        : ''"
      confirm-label="Revogar"
      variant="danger"
      :loading="revokingKeyNow"
      @confirm="onConfirmRevokeKey"
    />

    <ConfirmModal
      v-model="deleteAccountOpen"
      title="Remover conta?"
      :message="deletingAccount
        ? `A conta '${deletingAccount.name}' (#${deletingAccount.id}) será removida. Não é possível remover se houver pixels/admins vinculados.`
        : ''"
      confirm-label="Remover"
      variant="danger"
      :loading="deletingAccountNow"
      @confirm="onConfirmDeleteAccount"
    />
  </div>
</template>

<script setup lang="ts">
import type {
  Account,
  AdminUser,
  APIKey,
  GeoIPStatus,
  MaintenanceStatusResponse,
  PruneResultRow,
  ResetAccountResponse,
} from '#shared/types'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

useHead({ title: 'Configurações - TrackFind' })

const settings = useSettings()
const accountsAPI = useAccounts()
const apiKeysAPI = useAPIKeys()
const { user } = useAuth()
const toast = useToast()

const isOwner = computed(() => {
  // Compatível com tokens antigos (sem role) que são tratados como owner.
  const role = user.value?.role
  return !role || role === 'owner'
})

const tabs = computed(() => {
  const base = [
    { value: 'account', label: 'Conta' },
    { value: 'admins', label: 'Administradores' },
    { value: 'api-keys', label: 'API Keys' },
    { value: 'geoip', label: 'GeoIP' },
    { value: 'retention', label: 'Retenção' },
  ] as const
  if (isOwner.value) {
    return [...base, { value: 'accounts' as const, label: 'Contas' }]
  }
  return base
})

type TabValue = 'account' | 'admins' | 'api-keys' | 'geoip' | 'retention' | 'accounts'

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
const newAdminForm = reactive({
  email: '',
  password: '',
  role: 'owner' as 'owner' | 'member',
  account_id: null as number | null,
})
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
  if (newAdminForm.role === 'member' && !newAdminForm.account_id) {
    toast.error('Erro', 'Members precisam de uma conta vinculada')
    return
  }
  creatingAdmin.value = true
  try {
    const created = await settings.createAdmin({
      email: newAdminForm.email.trim().toLowerCase(),
      password: newAdminForm.password,
      role: isOwner.value ? newAdminForm.role : undefined,
      account_id: isOwner.value && newAdminForm.role === 'member'
        ? newAdminForm.account_id
        : null,
    })
    admins.value = [...admins.value, created]
    toast.success('Admin criado', `${created.email} já pode logar`)
    newAdminOpen.value = false
    newAdminForm.email = ''
    newAdminForm.password = ''
    newAdminForm.role = 'owner'
    newAdminForm.account_id = null
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

// === API Keys ===
const apiKeys = ref<APIKey[]>([])
const loadingKeys = ref(false)
const newKeyOpen = ref(false)
const keyForm = reactive<{ name: string; account_id: number | null }>({
  name: '',
  account_id: null,
})
const creatingKey = ref(false)

const newKeyResultOpen = ref(false)
const createdKeyPlaintext = ref('')

const revokeKeyOpen = ref(false)
const revokingKey = ref<APIKey | null>(null)
const revokingKeyNow = ref(false)

const loadAPIKeys = async () => {
  loadingKeys.value = true
  try {
    const res = await apiKeysAPI.list()
    apiKeys.value = res.data || []
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao listar keys')
  } finally {
    loadingKeys.value = false
  }
}

const onCreateKey = async () => {
  if (!keyForm.name.trim()) {
    toast.error('Erro', 'Nome é obrigatório')
    return
  }
  creatingKey.value = true
  try {
    const res = await apiKeysAPI.create({
      name: keyForm.name.trim(),
      account_id: isOwner.value && keyForm.account_id
        ? keyForm.account_id
        : undefined,
    })
    apiKeys.value = [res.key, ...apiKeys.value]
    createdKeyPlaintext.value = res.plaintext
    newKeyOpen.value = false
    newKeyResultOpen.value = true
    keyForm.name = ''
    keyForm.account_id = null
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao criar key')
  } finally {
    creatingKey.value = false
  }
}

const copyKey = async () => {
  try {
    await navigator.clipboard.writeText(createdKeyPlaintext.value)
    toast.success('Copiado', 'A key foi copiada pra área de transferência')
  } catch {
    toast.error('Erro', 'Não foi possível copiar — selecione manualmente')
  }
}

const closeKeyResult = () => {
  newKeyResultOpen.value = false
  createdKeyPlaintext.value = ''
}

const askRevokeKey = (k: APIKey) => {
  revokingKey.value = k
  revokeKeyOpen.value = true
}

const onConfirmRevokeKey = async () => {
  if (!revokingKey.value) return
  revokingKeyNow.value = true
  try {
    await apiKeysAPI.revoke(revokingKey.value.id)
    apiKeys.value = apiKeys.value.map((k) =>
      k.id === revokingKey.value!.id
        ? { ...k, revoked_at: new Date().toISOString() }
        : k
    )
    toast.success('Key revogada')
    revokeKeyOpen.value = false
    revokingKey.value = null
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao revogar')
  } finally {
    revokingKeyNow.value = false
  }
}

// === Accounts (multi-tenant; só owners) ===
const accounts = ref<Account[]>([])
const loadingAccounts = ref(false)
const newAccountOpen = ref(false)
const editingAccount = ref<Account | null>(null)
const accountForm = reactive({
  name: '',
  slug: '',
  enabled: true,
})
const savingAccount = ref(false)
const deleteAccountOpen = ref(false)
const deletingAccount = ref<Account | null>(null)
const deletingAccountNow = ref(false)

const loadAccounts = async () => {
  loadingAccounts.value = true
  try {
    const res = await accountsAPI.list()
    accounts.value = res.data || []
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao listar contas')
  } finally {
    loadingAccounts.value = false
  }
}

const editAccount = (a: Account) => {
  editingAccount.value = a
  accountForm.name = a.name
  accountForm.slug = a.slug
  accountForm.enabled = a.enabled
  newAccountOpen.value = true
}

const closeAccountModal = () => {
  newAccountOpen.value = false
  editingAccount.value = null
  accountForm.name = ''
  accountForm.slug = ''
  accountForm.enabled = true
}

const onSaveAccount = async () => {
  if (!accountForm.name.trim()) {
    toast.error('Erro', 'Nome é obrigatório')
    return
  }
  savingAccount.value = true
  try {
    if (editingAccount.value) {
      const updated = await accountsAPI.update(editingAccount.value.id, {
        name: accountForm.name,
        slug: accountForm.slug || undefined,
        enabled: accountForm.enabled,
      })
      accounts.value = accounts.value.map((a) =>
        a.id === updated.id ? updated : a
      )
      toast.success('Conta atualizada')
    } else {
      const created = await accountsAPI.create({
        name: accountForm.name,
        slug: accountForm.slug || undefined,
        enabled: accountForm.enabled,
      })
      accounts.value = [...accounts.value, created]
      toast.success('Conta criada', `Slug: ${created.slug}`)
    }
    closeAccountModal()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao salvar conta')
  } finally {
    savingAccount.value = false
  }
}

const askDeleteAccount = (a: Account) => {
  deletingAccount.value = a
  deleteAccountOpen.value = true
}

const onConfirmDeleteAccount = async () => {
  if (!deletingAccount.value) return
  deletingAccountNow.value = true
  try {
    await accountsAPI.remove(deletingAccount.value.id)
    accounts.value = accounts.value.filter((a) => a.id !== deletingAccount.value!.id)
    toast.success('Conta removida')
    deleteAccountOpen.value = false
    deletingAccount.value = null
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao remover conta (talvez ainda tenha pixels/admins vinculados)')
  } finally {
    deletingAccountNow.value = false
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

const retentionForm = reactive<{
  sent_events: number | null
  event_counters: number | null
}>({ sent_events: null, event_counters: null })
const savingRetention = ref(false)

const confirmReset = ref(false)
const resetting = ref(false)
const lastResetResult = ref<ResetAccountResponse | null>(null)

const loadMaintenance = async () => {
  try {
    maintenanceData.value = await settings.maintenanceStatus()
    // Pré-popula o form com o override atual da conta (se houver).
    retentionForm.sent_events = maintenanceData.value.account_sent_events_retention_days ?? null
    retentionForm.event_counters = maintenanceData.value.account_event_counters_retention_days ?? null
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

const onSaveRetention = async () => {
  savingRetention.value = true
  try {
    await settings.updateRetention({
      sent_events_retention_days: retentionForm.sent_events,
      event_counters_retention_days: retentionForm.event_counters,
    })
    toast.success('Retenção atualizada', 'Próxima limpeza usará a nova janela')
    await loadMaintenance()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao salvar retenção')
  } finally {
    savingRetention.value = false
  }
}

const onResetRetention = async () => {
  savingRetention.value = true
  try {
    await settings.updateRetention({
      sent_events_retention_days: null,
      event_counters_retention_days: null,
    })
    retentionForm.sent_events = null
    retentionForm.event_counters = null
    toast.success('Override removido', 'Conta volta a usar o default global')
    await loadMaintenance()
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao remover override')
  } finally {
    savingRetention.value = false
  }
}

const onReset = async () => {
  resetting.value = true
  try {
    const res = await settings.resetAccount()
    lastResetResult.value = res
    toast.success(
      'Conta resetada',
      `${formatNumber(res.deleted_sent_events)} eventos + ${formatNumber(res.deleted_event_counters)} contadores apagados`
    )
    confirmReset.value = false
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao resetar conta')
  } finally {
    resetting.value = false
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
  if (tab === 'accounts') loadAccounts()
  if (tab === 'api-keys') loadAPIKeys()
})

// Carrega lista de contas em background quando o modal de novo admin abre
// (pra alimentar o <select> de account_id pra members).
watch(newAdminOpen, (open) => {
  if (open && isOwner.value && !accounts.value.length) loadAccounts()
})

// Mesma coisa pro modal de nova API key (owner pode escolher conta destino).
watch(newKeyOpen, (open) => {
  if (open && isOwner.value && !accounts.value.length) loadAccounts()
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

/* === Accounts ============================================================= */
.account-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.account-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  background: var(--surface-soft-2);
}

.account-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.account-name {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  color: var(--text-1);
}

.account-meta {
  font-size: 0.75rem;
  color: var(--text-3);
}

.account-meta code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
}

.account-actions {
  display: flex;
  gap: 0.25rem;
}

/* === Form fields extras (select / checkbox) =============================== */
.form-field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.form-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-2);
}

.form-select {
  width: 100%;
  padding: 0.625rem 0.875rem;
  background: var(--surface-deep);
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text-1);
  font-size: 0.9375rem;
  outline: none;
  transition: border-color 0.15s;
}

.form-select:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.checkbox-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.9375rem;
  color: var(--text-1);
  cursor: pointer;
}

.checkbox-row input[type='checkbox'] {
  width: 1rem;
  height: 1rem;
  accent-color: var(--brand);
  cursor: pointer;
}

/* === Secret reveal block (API key plaintext) ============================= */
.key-secret {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 0.875rem;
  background: var(--surface-deep);
  border: 1px solid var(--brand-border);
  border-radius: 8px;
}

.key-secret code {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8125rem;
  color: var(--brand);
  word-break: break-all;
}

/* === Retention form ====================================================== */
.retention-form {
  margin-top: 1.25rem;
  padding-top: 1.25rem;
  border-top: 1px solid var(--border-soft);
}

.card-subtitle {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-strong);
  margin-bottom: 0.875rem;
}

.form-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
  margin-bottom: 1rem;
}

.retention-source {
  font-size: 0.6875rem;
  color: var(--text-3);
  font-style: italic;
  margin-top: 0.125rem;
}
</style>
