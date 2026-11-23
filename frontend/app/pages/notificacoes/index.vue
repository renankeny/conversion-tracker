<template>
  <div class="notif-page">
    <header class="page-header">
      <div class="header-titles">
        <h1 class="page-title">Notificações & Aplicativo</h1>
        <p class="page-subtitle">
          Configure os alertas em tempo real para o seu computador ou celular e acompanhe as últimas ocorrências.
        </p>
      </div>

      <NotificationCompanyDropdown
        v-if="accounts.length > 0"
        :accounts="accounts"
        :mode="preferences.account_scope_mode"
        :selected-ids="preferences.enabled_account_ids"
        @change="onScopeChange"
      />
    </header>

    <div class="notif-grid">
      <div class="main-col">
        <NotificationDeviceCard
          :push-supported="pushSupported"
          :permission-state="permissionState"
          :is-i-o-s="isIOS"
          :is-standalone-p-w-a="isStandalonePWA"
          :can-install-p-w-a="canInstallPWA"
          :is-this-device-subscribed="isThisDeviceSubscribed"
          :current-endpoint="currentEndpoint"
          :subscribing="subscribing"
          :subscriptions="subscriptions"
          @install="onInstallPWA"
          @subscribe="onSubscribeDevice"
          @remove="onRemoveDevice"
        />

        <NotificationTriggersCard
          :preferences="preferences"
          :testing-category="testingCategory"
          @toggle="onToggleTrigger"
          @test="onTestTrigger"
        />
      </div>

      <aside class="side-col">
        <NotificationHistoryCard
          :notifications="notifications"
          :unread-count="unreadCount"
          :loading="loading"
          @open="onOpenNotification"
          @read-all="onReadAll"
        />
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import type {
  AccountScopeMode,
  ConfigurableNotificationCategory,
  PushSubscriptionItem,
  UserNotificationItem,
} from '#shared/types'
import NotificationCompanyDropdown from '~/components/notifications/NotificationCompanyDropdown.vue'
import NotificationDeviceCard from '~/components/notifications/NotificationDeviceCard.vue'
import NotificationHistoryCard from '~/components/notifications/NotificationHistoryCard.vue'
import NotificationTriggersCard from '~/components/notifications/NotificationTriggersCard.vue'
import { useNotificationsStore } from '~/stores/useNotificationsStore'
import { errorMessage } from '~/utils/errorMessage'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Notificações')) })

const router = useRouter()
const toast = useToast()
const { accounts, loadAccounts } = useAccountNavigation()
const notifStore = useNotificationsStore()

const {
  unreadCount, preferences, subscriptions, notifications,
  loading, subscribing, testingCategory,
  pushSupported, permissionState, isIOS, isStandalonePWA,
  canInstallPWA, isThisDeviceSubscribed, currentEndpoint,
} = storeToRefs(notifStore)

onMounted(async () => {
  await Promise.all([
    loadAccounts(),
    notifStore.initClientPWA(),
    notifStore.fetchOverview(true).catch(() => {}),
  ])
})

const onInstallPWA = async () => {
  const installed = await notifStore.installPWA()
  if (installed) {
    toast.success('Aplicativo instalado', 'Agora você pode abrir o painel direto da sua tela inicial.')
  }
}

const onSubscribeDevice = async () => {
  try {
    await notifStore.subscribeThisDevice()
    toast.success('Notificações ativadas', 'Este dispositivo já está recebendo alertas em segundo plano.')
  } catch (e: unknown) {
    toast.error('Não foi possível ativar', errorMessage(e, 'Verifique as permissões do navegador.'))
  }
}

const onRemoveDevice = async (sub: PushSubscriptionItem) => {
  try {
    await notifStore.removeSubscription(sub)
    toast.success('Dispositivo removido', `"${sub.device_label}" foi desconectado.`)
  } catch (e: unknown) {
    toast.error('Erro ao remover', errorMessage(e, 'Tente novamente.'))
  }
}

const onScopeChange = async (payload: { account_scope_mode: AccountScopeMode; enabled_account_ids: number[] }) => {
  try {
    await notifStore.savePreferences(payload)
  } catch (e: unknown) {
    toast.error('Erro ao salvar empresas', errorMessage(e, 'Tente novamente.'))
  }
}

const onToggleTrigger = async (key: ConfigurableNotificationCategory, nextVal: boolean) => {
  try {
    await notifStore.savePreferences({ [key]: nextVal })
  } catch (e: unknown) {
    toast.error('Erro ao salvar preferência', errorMessage(e, 'Tente novamente.'))
  }
}

const onTestTrigger = async (key: ConfigurableNotificationCategory) => {
  try {
    await notifStore.sendTest(key)
    toast.success('Notificação de teste enviada', 'Confira o alerta no seu aparelho e no histórico ao lado.')
  } catch (e: unknown) {
    toast.error('Falha no envio do teste', errorMessage(e, 'Tente novamente.'))
  }
}

const onOpenNotification = async (item: UserNotificationItem) => {
  if (!item.is_read) {
    await notifStore.markRead(item.id).catch(() => {})
  }
  if (item.action_url) {
    await router.push(item.action_url)
  }
}

const onReadAll = async () => {
  await notifStore.markAllRead().catch(() => {})
}
</script>

<style scoped>
.notif-page { display: flex; flex-direction: column; gap: 1.5rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.header-titles { flex: 1; min-width: 260px; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin: 0 0 0.25rem; }
.page-subtitle { font-size: var(--fs-md); color: var(--text-2); margin: 0; max-width: 680px; line-height: 1.5; }
.notif-grid { display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(320px, 1fr); gap: 1.25rem; align-items: start; }
.main-col { display: flex; flex-direction: column; gap: 1.25rem; }
@media (max-width: 1080px) {
  .notif-grid { grid-template-columns: 1fr; }
}
</style>
