<template>
  <BaseModal
    :model-value="modelValue"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <template #header>
      <div v-if="event" class="modal-event-header">
        <div class="header-top-row">
          <span class="category-pill" :class="eventCategoryClass">{{ eventCategoryLabel }}</span>
          <h2 class="event-title">{{ event.event_name }}</h2>
          <span class="pixel-pill" :title="`Pixel: ${event.pixel_name}`">
            <span class="pixel-dot" />
            {{ event.pixel_name }}
          </span>
        </div>
        <div class="header-meta-row">
          <span class="meta-item">
            <Clock class="w-3.5 h-3.5 opacity-60" />
            {{ formatTime(event.created_at) }}
          </span>
          <span class="meta-separator">•</span>
          <span class="meta-item" :style="{ color: hasFailedChannels ? 'var(--danger)' : 'var(--brand)' }">
            <span class="status-dot" :class="hasFailedChannels ? 'dot-error' : 'dot-success'" />
            {{ syncStatusTitle }} ({{ successfulChannelsCount }} de {{ event.channels.length }} canais)
          </span>
        </div>
      </div>
      <div v-else class="header-top-row">
        <h2 class="event-title">Detalhes do Evento</h2>
      </div>
    </template>

    <div v-if="loading" class="loading-block">
      <div class="spinner" />
      <p>Carregando detalhes do evento…</p>
    </div>

    <div v-else-if="event" class="event-details">
      <EventUserMatchCard
        :external-id="event.external_id"
        :tracked-user="trackedUser"
        :allow-user-details="allowUserDetails"
        @open-profile="openUserDetails"
      />

      <EventSummaryStrip
        v-if="event.event_name === 'Purchase'"
        :source="source"
        :tracked-user="trackedUser"
      />

      <div class="channels-section">
        <div class="flex items-center justify-between mb-1">
          <h4 class="channels-title">Canais de Transmissão</h4>
          <span class="text-xs" style="color: var(--text-3)">Clique no canal para ver detalhes da transmissão</span>
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
      <BaseButton variant="ghost" size="sm" @click="$emit('update:modelValue', false)">Fechar</BaseButton>
    </template>
  </BaseModal>

  <UserDetailsModal v-model="userModalOpen" :user="trackedUser" />
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Clock } from 'lucide-vue-next'
import ChannelCard from '~/components/events/EventChannelCard.vue'
import EventUserMatchCard from '~/components/events/EventUserMatchCard.vue'
import EventSummaryStrip from '~/components/events/EventSummaryStrip.vue'
import UserDetailsModal from '~/components/users/UserDetailsModal.vue'
import type { Channel, EventDetail, GroupedEvent, SentEvent, TrackedUser } from '#shared/types'

interface Props {
  modelValue: boolean
  source: GroupedEvent | null
  allowUserDetails?: boolean
}

const props = withDefaults(defineProps<Props>(), { allowUserDetails: true })
defineEmits<{ 'update:modelValue': [value: boolean] }>()

const eventsApi = useEvents()
const trackedUsersApi = useTrackedUsers()
const event = ref<EventDetail | null>(null)
const trackedUser = ref<TrackedUser | null>(null)
const loading = ref(false)
const expandedChannel = ref<Channel | null>(null)
const userModalOpen = ref(false)

const openUserDetails = () => { if (trackedUser.value) userModalOpen.value = true }

const CHANNELS: Array<{ id: Channel; label: string }> = [
  { id: 'ga4', label: 'Google Analytics 4' },
  { id: 'meta', label: 'Meta Conversions API' },
  { id: 'tiktok', label: 'TikTok Events API' },
  { id: 'google_ads', label: 'Google Ads' },
]

const getChannelData = (id: Channel): SentEvent | null =>
  event.value?.channels.find((c) => c.channel === id) || null

const toggleChannel = (id: Channel) => {
  expandedChannel.value = expandedChannel.value === id ? null : id
}

const successfulChannelsCount = computed(() =>
  event.value?.channels.filter((c) => c.status === 'sent' && c.response_code >= 200 && c.response_code < 300).length || 0
)

const hasFailedChannels = computed(() =>
  Boolean(event.value?.channels.some((c) => c.status === 'failed' || (c.response_code >= 400 && c.response_code !== 0) || !!c.error))
)

const syncStatusTitle = computed(() => {
  if (!event.value) return 'Aguardando'
  if (hasFailedChannels.value) return 'Falha na Transmissão'
  return successfulChannelsCount.value > 0 ? 'Transmissão Completa' : 'Em Processamento'
})

const CATEGORY_MAP: Record<string, { label: string; cls: string }> = {
  Purchase: { label: 'Conversão', cls: 'cat-purchase' },
  InitiateCheckout: { label: 'Checkout', cls: 'cat-checkout' },
  CompleteRegistration: { label: 'Cadastro', cls: 'cat-reg' },
  Lead: { label: 'Lead', cls: 'cat-lead' },
  Contact: { label: 'Contato', cls: 'cat-contact' },
  LeadSubmitted: { label: 'Contato', cls: 'cat-contact' },
}

const eventCategoryLabel = computed(() => CATEGORY_MAP[event.value?.event_name || '']?.label || 'Evento')
const eventCategoryClass = computed(() => CATEGORY_MAP[event.value?.event_name || '']?.cls || 'cat-default')

const fetchDetails = async () => {
  if (!props.source) return
  loading.value = true
  expandedChannel.value = null
  trackedUser.value = null
  try {
    const externalId = props.source.external_id
    const [ev, tu] = await Promise.all([
      eventsApi.findByEventID(props.source.event_id, props.source.pixel_name),
      props.allowUserDetails && externalId
        ? trackedUsersApi.lookup({ external_id: externalId }).catch(() => null)
        : Promise.resolve(null),
    ])
    event.value = ev
    trackedUser.value = tu
  } catch (e: any) {
    console.error('Erro ao carregar detalhes', e)
    event.value = null
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.modelValue, props.source?.event_id, props.source?.pixel_name] as const,
  ([open]) => {
    if (open && props.source) fetchDetails()
    else if (!open) event.value = null
  },
)

const formatTime = (iso: string): string =>
  iso ? new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' }) : ''
</script>

<style scoped>
.loading-block { display: flex; flex-direction: column; align-items: center; gap: 0.75rem; padding: 3.5rem 1rem; color: var(--text-3); }
.spinner { width: 1.75rem; height: 1.75rem; border: 3px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.event-details { display: flex; flex-direction: column; gap: 1.25rem; }
.modal-event-header { display: flex; flex-direction: column; gap: 0.25rem; }
.header-top-row { display: flex; align-items: center; flex-wrap: wrap; gap: 0.5rem; }
.event-title { font-size: var(--fs-xl); font-weight: var(--fw-bold); color: var(--text-1); letter-spacing: -0.01em; margin: 0; }
.header-meta-row { display: flex; align-items: center; flex-wrap: wrap; gap: 0.5rem; font-size: var(--fs-sm); color: var(--text-3); }
.meta-item { display: inline-flex; align-items: center; gap: 0.35rem; }
.meta-separator { opacity: 0.3; }
.status-dot { width: 6px; height: 6px; border-radius: 9999px; display: inline-block; }
.status-dot.dot-success { background: var(--success); box-shadow: 0 0 6px rgba(34, 197, 94, 0.4); }
.status-dot.dot-error { background: var(--danger); box-shadow: 0 0 6px rgba(239, 68, 68, 0.4); }

.category-pill { font-size: var(--fs-2xs); font-weight: var(--fw-bold); text-transform: uppercase; letter-spacing: 0.05em; padding: 0.125rem 0.5rem; border-radius: 9999px; }
.category-pill.cat-purchase { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.category-pill.cat-checkout { background: rgba(251, 191, 36, 0.15); color: #fbbf24; border: 1px solid rgba(251, 191, 36, 0.25); }
.category-pill.cat-reg { background: rgba(56, 189, 248, 0.15); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.25); }
.category-pill.cat-lead { background: rgba(168, 85, 247, 0.15); color: #c084fc; border: 1px solid rgba(168, 85, 247, 0.25); }
.category-pill.cat-contact { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.category-pill.cat-default { background: var(--surface-soft); color: var(--text-2); border: 1px solid var(--border); }

.pixel-pill { display: inline-flex; align-items: center; gap: 0.375rem; font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-2); background: var(--surface-deep); padding: 0.125rem 0.5rem; border-radius: 6px; border: 1px solid var(--border-soft); }
.pixel-dot { width: 6px; height: 6px; border-radius: 9999px; background: var(--brand); }
.channels-section { display: flex; flex-direction: column; gap: 0.5rem; }
.channels-title { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.channels-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 0.75rem; }
</style>
