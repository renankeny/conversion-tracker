<template>
  <BaseModal :model-value="modelValue" size="lg" @update:model-value="$emit('update:modelValue', $event)">
    <template #header>
      <EventModalHeader :event="event" />
    </template>

    <div v-if="loading" class="loading-block">
      <div class="spinner" />
      <p>Carregando detalhes do evento...</p>
    </div>

    <div v-else-if="errorMessage" class="error-block">
      <AlertCircle class="w-8 h-8 error-icon" />
      <p class="error-title">Não foi possível carregar os detalhes</p>
      <p class="error-desc">{{ errorMessage }}</p>
      <BaseButton variant="outline" size="sm" class="mt-3" @click="fetchDetails">Tentar novamente</BaseButton>
    </div>

    <div v-else-if="event" class="event-details">
      <EventUserMatchCard
        :external-id="event.external_id"
        :tracked-user="trackedUser"
        :source="source"
        :allow-user-details="allowUserDetails"
        @open-profile="openUserDetails"
      />

      <EventSummaryStrip
        v-if="event.event_name === 'Purchase'"
        :source="source"
        :tracked-user="trackedUser"
        :currency="currency"
      />

      <div class="channels-section">
        <div class="flex items-center justify-between mb-1">
          <h4 class="channels-title">Canais de Transmissão</h4>
          <span v-if="!publicMode && !isInternalOnly" class="channels-hint">
            Clique no canal para ver detalhes da transmissão
          </span>
        </div>

        <div v-if="isInternalOnly" class="internal-info-banner">
          <Info class="w-4 h-4 banner-icon" />
          <span>
            Este evento foi registrado internamente no TrackFind, CRM e Dashboard, sem canais externos de mídia configurados no pixel no momento da captura.
          </span>
        </div>

        <div class="channels-grid">
          <ChannelCard
            v-for="ch in CHANNELS"
            :key="ch.id"
            :channel-id="ch.id"
            :label="ch.label"
            :data="getChannelData(ch.id)"
            :expanded="expandedChannel === ch.id"
            @toggle="toggleChannel(ch.id)"
          />
        </div>
      </div>
    </div>

    <template #footer>
      <BaseButton
        v-if="!publicMode && hasFailedChannels && canManageUsers"
        variant="outline"
        size="sm"
        :loading="retrying"
        @click="retryFailedChannels"
      >
        <template #iconLeft><RefreshCw class="w-3.5 h-3.5" /></template>
        Reenviar canais com falha
      </BaseButton>
      <BaseButton variant="ghost" size="sm" @click="$emit('update:modelValue', false)">Fechar</BaseButton>
    </template>
  </BaseModal>

  <UserDetailsModal v-model="userModalOpen" :user="trackedUser" @saved="onUserSaved" @deleted="onUserDeleted" />
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { AlertCircle, Info, RefreshCw } from 'lucide-vue-next'
import ChannelCard from '~/components/events/EventChannelCard.vue'
import EventModalHeader from '~/components/events/EventModalHeader.vue'
import EventUserMatchCard from '~/components/events/EventUserMatchCard.vue'
import EventSummaryStrip from '~/components/events/EventSummaryStrip.vue'
import UserDetailsModal from '~/components/users/UserDetailsModal.vue'
import { useAuthStore } from '~/stores/useAuthStore'
import type { Channel, EventDetail, GroupedEvent, SentEvent, TrackedUser } from '#shared/types'
import { errorMessage as extractErrorMessage } from '~/utils/errorMessage'

interface Props {
  modelValue: boolean
  source: GroupedEvent | null
  allowUserDetails?: boolean
  publicMode?: boolean
  currency?: string
}

const props = withDefaults(defineProps<Props>(), {
  allowUserDetails: true,
  publicMode: false,
  currency: 'BRL',
})

const emit = defineEmits<{ 'update:modelValue': [value: boolean]; retried: [] }>()

const eventsApi = useEvents()
const trackedUsersApi = useTrackedUsers()
const toast = useToast()
const authStore = useAuthStore()

const canManageUsers = computed(() => authStore.canManageUsers)

const event = ref<EventDetail | null>(null)
const trackedUser = ref<TrackedUser | null>(null)
const loading = ref(false)
const retrying = ref(false)
const errorMessage = ref('')
const expandedChannel = ref<Channel | null>(null)
const userModalOpen = ref(false)
let retryDetailTimer: ReturnType<typeof setTimeout> | null = null

onUnmounted(() => { if (retryDetailTimer) clearTimeout(retryDetailTimer) })

const openUserDetails = () => { if (trackedUser.value) userModalOpen.value = true }
const onUserSaved = (u: TrackedUser) => { trackedUser.value = u }
const onUserDeleted = () => { trackedUser.value = null; userModalOpen.value = false }

const CHANNELS: Array<{ id: Channel; label: string }> = [
  { id: 'ga4', label: 'Google Analytics 4' },
  { id: 'meta', label: 'Meta Conversions API' },
  { id: 'tiktok', label: 'TikTok Events API' },
  { id: 'google_ads', label: 'Google Ads' },
]

const externalChannels = computed(() => (event.value?.channels || []).filter((c) => c.channel !== 'internal'))
const isInternalOnly = computed(() => Boolean(event.value?.channels.length) && externalChannels.value.length === 0)
const hasFailedChannels = computed(() =>
  externalChannels.value.some((c) => c.status === 'failed' || (c.response_code >= 400 && c.response_code !== 0) || Boolean(c.error)),
)

const getChannelData = (id: Channel): SentEvent | null => event.value?.channels.find((c) => c.channel === id) || null
const toggleChannel = (id: Channel) => {
  if (!props.publicMode) expandedChannel.value = expandedChannel.value === id ? null : id
}

const buildPublicDetail = (src: GroupedEvent): EventDetail => ({
  event_id: src.event_id, event_name: src.event_name, pixel_name: src.pixel_name,
  external_id: src.external_id, created_at: src.created_at,
  channels: (src.channels || []).map((c) => ({
    id: c.id, event_id: src.event_id, pixel_name: src.pixel_name, event_name: src.event_name,
    external_id: src.external_id, channel: c.channel, status: c.status, response_code: c.response_code,
    error: c.has_error ? 'Falha reportada pela plataforma' : '', response_body: '',
    purchase_value: src.purchase_value, created_at: src.created_at, updated_at: src.created_at,
  })),
})

let detailSeq = 0

const fetchDetails = async () => {
  if (!props.source) return
  const seq = ++detailSeq
  errorMessage.value = ''; expandedChannel.value = null; trackedUser.value = null
  if (props.publicMode) { event.value = buildPublicDetail(props.source); loading.value = false; return }
  loading.value = true
  try {
    const externalId = props.source.external_id
    const [ev, tu] = await Promise.all([
      eventsApi.findByEventID(props.source.event_id, props.source.pixel_name),
      props.allowUserDetails && externalId ? trackedUsersApi.lookup({ external_id: externalId }).catch(() => null) : Promise.resolve(null),
    ])
    if (seq !== detailSeq) return
    event.value = ev; trackedUser.value = tu
  } catch (e: unknown) {
    if (seq !== detailSeq) return
    errorMessage.value = extractErrorMessage(e, 'Erro ao carregar detalhes do evento')
    event.value = null
  } finally {
    if (seq === detailSeq) loading.value = false
  }
}

const retryFailedChannels = async () => {
  if (!event.value || retrying.value) return
  retrying.value = true
  try {
    const r = await eventsApi.retryEvents([{ event_id: event.value.event_id, pixel_name: event.value.pixel_name }])
    if (r.queued === 0) {
      toast.warning('Reenvio indisponível', 'O payload original deste evento já expirou da janela de retenção.')
    } else {
      toast.success('Reenfileirado', 'Evento enviado para reprocessamento nos canais com falha.')
      if (retryDetailTimer) clearTimeout(retryDetailTimer)
      retryDetailTimer = setTimeout(() => {
        if (props.modelValue && props.source) void fetchDetails()
        emit('retried')
      }, 1200)
    }
  } catch (e: unknown) {
    toast.error('Erro', extractErrorMessage(e, 'Não foi possível reenviar o evento'))
  } finally { retrying.value = false }
}

watch(
  () => [props.modelValue, props.source?.event_id, props.source?.pixel_name] as const,
  ([open]) => {
    if (open && props.source) void fetchDetails()
    else if (!open) event.value = null
  },
)
</script>

<style scoped>
.loading-block, .error-block { display: flex; flex-direction: column; align-items: center; gap: 0.75rem; padding: 3.5rem 1rem; color: var(--text-3); text-align: center; }
.error-icon { color: var(--danger); }
.error-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.error-desc { font-size: var(--fs-sm); color: var(--text-3); margin: 0; }
.spinner { width: 1.75rem; height: 1.75rem; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.event-details { display: flex; flex-direction: column; gap: 1.25rem; }
.channels-section { display: flex; flex-direction: column; gap: 0.5rem; }
.channels-title { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.channels-hint { font-size: var(--fs-base); color: var(--text-3); }
.internal-info-banner { display: flex; align-items: flex-start; gap: 0.625rem; padding: 0.625rem 0.875rem; border-radius: 8px; background: var(--surface-soft-2); border: 1px solid var(--border); color: var(--text-2); font-size: var(--fs-sm); line-height: 1.45; }
.banner-icon { color: var(--accent-sky); flex-shrink: 0; margin-top: 0.125rem; }
.channels-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 0.75rem; }
</style>
