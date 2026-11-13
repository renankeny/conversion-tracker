<template>
  <div class="settings-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Configurações</h1>
        <p class="page-subtitle">
          Gestão da sua conta, equipe, personalização Whitelabel e retenção de dados.
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

    <SettingsProfileTab v-if="activeTab === 'account'" />
    <SettingsWhatsAppAiTab v-else-if="activeTab === 'whatsapp-ai'" @go-profile="activeTab = 'account'" />
    <SettingsAdminsTab v-else-if="activeTab === 'admins'" />
    <SettingsApiKeysTab v-else-if="activeTab === 'api-keys'" />
    <SettingsRetentionTab v-else-if="activeTab === 'retention'" />
    <SettingsBrandingTab v-else-if="activeTab === 'branding'" />
    <SettingsGeoIpTab v-else-if="activeTab === 'geoip'" />
  </div>
</template>

<script setup lang="ts">
import SettingsProfileTab from '~/components/settings/SettingsProfileTab.vue'
import SettingsWhatsAppAiTab from '~/components/settings/SettingsWhatsAppAiTab.vue'
import SettingsAdminsTab from '~/components/settings/SettingsAdminsTab.vue'
import SettingsApiKeysTab from '~/components/settings/SettingsApiKeysTab.vue'
import SettingsRetentionTab from '~/components/settings/SettingsRetentionTab.vue'
import SettingsBrandingTab from '~/components/settings/SettingsBrandingTab.vue'
import SettingsGeoIpTab from '~/components/settings/SettingsGeoIpTab.vue'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Configurações')) })

const { user, isOwner, canManageAccounts } = useAuth()

const isSuperAdmin = computed(() => user.value?.role === 'superadmin' || user.value?.role === 'owner')
const isAdmin = computed(() => user.value?.role === 'admin' || user.value?.role === 'superadmin')
const isEmployee = computed(() => user.value?.role === 'employee')
const canManageBranding = computed(() => isOwner.value || isAdmin.value)

type TabValue = 'account' | 'whatsapp-ai' | 'admins' | 'api-keys' | 'retention' | 'branding' | 'geoip'

const tabs = computed(() => {
  const allTabs: { value: TabValue; label: string }[] = [
    { value: 'account', label: 'Meu perfil' },
    { value: 'api-keys', label: 'API Keys' },
  ]

  if (!isEmployee.value) {
    allTabs.splice(1, 0, { value: 'whatsapp-ai', label: 'IA WhatsApp' }, { value: 'admins', label: 'Usuários' })
  }

  if (canManageAccounts.value) {
    allTabs.push({ value: 'retention', label: 'Retenção' })
  }

  if (canManageBranding.value) {
    allTabs.push({ value: 'branding', label: 'Personalização' })
  }

  if (isSuperAdmin.value) {
    allTabs.push({ value: 'geoip', label: 'GeoIP' })
  }

  return allTabs
})

const activeTab = ref<TabValue>('account')
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.page-title {
  font-family: var(--font-sans);
  font-size: var(--fs-3xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.page-subtitle {
  color: var(--text-2);
  font-size: var(--fs-lg);
}

.tabs {
  display: flex;
  gap: 0.25rem;
  border-bottom: 1px solid var(--border);
  overflow-x: auto;
  white-space: nowrap;
  -webkit-overflow-scrolling: touch;
}

.tab {
  padding: 0.75rem 1.25rem;
  background: transparent;
  border: none;
  color: var(--text-2);
  font-size: var(--fs-lg);
  font-weight: var(--fw-medium);
  cursor: pointer;
  position: relative;
  transition: color 0.15s;
  flex-shrink: 0;
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
</style>
