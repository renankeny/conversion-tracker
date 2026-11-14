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
          <span v-if="!publicMode" class="channels-hint">Clique no canal para ver detalhes da transmissão</span>
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
import { Clock, AlertCircle } from 'lucide-vue-next'
import ChannelCard from '~/components/events/EventChannelCard.vue'
import EventUserMatchCard from '~/components/events/EventUserMatchCard.vue'
import EventSummaryStrip from '~/components/events/EventSummaryStrip.vue'
import UserDetailsModal from '~/components/users/UserDetailsModal.vue'
import type { Channel, EventDetail, GroupedEvent, SentEvent, TrackedUser } from '#shared/types'
import { formatDateTimeSP } from '~/utils/formatDate'
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
defineEmits<{ 'update:modelValue': [value: boolean] }>()

const eventsApi = useEvents()
const trackedUsersApi = useTrackedUsers()
const event = ref<EventDetail | null>(null)
const trackedUser = ref<TrackedUser | null>(null)
const loading = ref(false)
const errorMessage = ref('')
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
  if (props.publicMode) return
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
  Purchase: { label: 'Conversão', cls: 'cat-purchase' }, InitiateCheckout: { label: 'Checkout', cls: 'cat-checkout' },
  CompleteRegistration: { label: 'Cadastro', cls: 'cat-reg' }, Lead: { label: 'Lead', cls: 'cat-lead' },
  QualifiedLead: { label: 'Lead', cls: 'cat-lead' }, Contact: { label: 'Contato', cls: 'cat-contact' },
  LeadSubmitted: { label: 'Contato', cls: 'cat-contact' },
}
const eventCategoryLabel = computed(() => CATEGORY_MAP[event.value?.event_name || '']?.label || 'Evento')
const eventCategoryClass = computed(() => CATEGORY_MAP[event.value?.event_name || '']?.cls || 'cat-default')

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

const fetchDetails = async () => {
  if (!props.source) return
  errorMessage.value = ''; expandedChannel.value = null; trackedUser.value = null
  if (props.publicMode) { event.value = buildPublicDetail(props.source); loading.value = false; return }
  loading.value = true
  try {
    const externalId = props.source.external_id
    const [ev, tu] = await Promise.all([
      eventsApi.findByEventID(props.source.event_id, props.source.pixel_name),
      props.allowUserDetails && externalId ? trackedUsersApi.lookup({ external_id: externalId }).catch(() => null) : Promise.resolve(null),
    ])
    event.value = ev; trackedUser.value = tu
  } catch (e: unknown) {
    errorMessage.value = extractErrorMessage(e, 'Erro ao carregar detalhes do evento')
    event.value = null
  } finally { loading.value = false }
}

watch(
  () => [props.modelValue, props.source?.event_id, props.source?.pixel_name] as const,
  ([open]) => {
    if (open && props.source) fetchDetails()
    else if (!open) event.value = null
  },
)

const formatTime = (iso: string): string => formatDateTimeSP(iso, true)
</script>

<style scoped>
.loading-block, .error-block { display: flex; flex-direction: column; align-items: center; gap: 0.75rem; padding: 3.5rem 1rem; color: var(--text-3); text-align: center; }
.error-icon { color: var(--danger); }
.error-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.error-desc { font-size: var(--fs-sm); color: var(--text-3); margin: 0; }
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
.status-dot.dot-success { background: var(--success); box-shadow: 0 0 6px var(--brand-border); }
.status-dot.dot-error { background: var(--danger); box-shadow: 0 0 6px var(--danger-border); }

.category-pill { font-size: var(--fs-2xs); font-weight: var(--fw-bold); text-transform: uppercase; letter-spacing: 0.05em; padding: 0.125rem 0.5rem; border-radius: 9999px; }
.category-pill.cat-purchase { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.category-pill.cat-checkout { background: var(--warning-soft); color: var(--warning-light); border: 1px solid var(--warning-border); }
.category-pill.cat-reg { background: var(--accent-sky-soft); color: var(--accent-sky); border: 1px solid var(--accent-sky-border); }
.category-pill.cat-lead { background: var(--accent-purple-soft); color: var(--accent-purple); border: 1px solid var(--accent-purple-border); }
.category-pill.cat-contact { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.category-pill.cat-default { background: var(--surface-soft); color: var(--text-2); border: 1px solid var(--border); }

.pixel-pill { display: inline-flex; align-items: center; gap: 0.375rem; font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-2); background: var(--surface-deep); padding: 0.125rem 0.5rem; border-radius: 6px; border: 1px solid var(--border-soft); }
.pixel-dot { width: 6px; height: 6px; border-radius: 9999px; background: var(--brand); }
.channels-section { display: flex; flex-direction: column; gap: 0.5rem; }
.channels-title { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-1); margin: 0; }
.channels-hint { font-size: var(--fs-base); color: var(--text-3); }
.channels-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 0.75rem; }
</style>
