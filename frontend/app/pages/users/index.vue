<template>
  <div class="users-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Usuários Rastreados</h1>
        <p class="page-subtitle">Visitantes identificados e sessões unificadas no site.</p>
      </div>
      <BaseButton variant="outline" size="sm" :loading="loading" @click="refresh">
        Atualizar
      </BaseButton>
    </div>

    <div class="filters">
      <div class="filter-search">
        <BaseInput
          v-model="searchInput"
          placeholder="Buscar por nome, e-mail, telefone ou ID..."
          @keyup.enter="applyFilters"
        >
          <template #icon>
            <Search :size="15" />
          </template>
        </BaseInput>
      </div>
      <div class="filter-select">
        <BaseDropdown v-model="segmentInput" :options="segmentOptions" placeholder="Todos os perfis" />
      </div>
      <div class="filter-select">
        <PixelDropdown v-model="pixelInput" />
      </div>
      <BaseButton v-if="hasActiveFilters" variant="ghost" size="md" @click="clearFilters">
        Limpar
      </BaseButton>
    </div>

    <UsersTable :users="users" :loading="loading" @select="onSelect" />

    <div v-if="total > 0" class="pagination">
      <span class="pag-info">
        Mostrando {{ offsetFrom }}-{{ offsetTo }} de {{ formatNumber(total) }}
      </span>
      <div class="pag-buttons">
        <BaseButton variant="ghost" size="sm" :disabled="offset === 0" @click="prevPage">
          ← Anterior
        </BaseButton>
        <BaseButton variant="ghost" size="sm" :disabled="offsetTo >= total" @click="nextPage">
          Próximo →
        </BaseButton>
      </div>
    </div>

    <UserDetailsModal
      v-model="modalOpen"
      :user="selected"
      @saved="onUserSaved"
      @deleted="onUserDeleted"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { Search } from 'lucide-vue-next'
import UsersTable from '~/components/users/UsersTable.vue'
import UserDetailsModal from '~/components/users/UserDetailsModal.vue'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'
import type { TrackedUser, TrackedUserFilters, TrackedUserSegment } from '#shared/types'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Usuários')) })

const trackedUsersApi = useTrackedUsers()
const toast = useToast()
const liveStore = useLiveEventsStore()
const tenantStore = useTenantStore()

const segmentOptions = [
  { label: 'Todos os perfis', value: '' },
  { label: 'Identificados', value: 'identified' },
  { label: 'Compradores (LTV > 0)', value: 'buyers' },
  { label: 'Visitantes anônimos', value: 'anonymous' },
  { label: 'Suspeita de Bot', value: 'bot' },
]

const PAGE_SIZE = 50
const searchInput = ref('')
const pixelInput = ref('')
const segmentInput = ref<'' | TrackedUserSegment>('')
const filters = ref<TrackedUserFilters>({})
const users = ref<TrackedUser[]>([])
const total = ref(0)
const offset = ref(0)
const loading = ref(false)
const modalOpen = ref(false)
const selected = ref<TrackedUser | null>(null)

let _usersRefreshTimer: ReturnType<typeof setTimeout> | null = null
let _searchDebounceTimer: ReturnType<typeof setTimeout> | null = null
let _pollingTimer: ReturnType<typeof setInterval> | null = null
let _unsubEventsUpdated: (() => void) | null = null
let _suppressFilterWatch = false
let usersSeq = 0

const hasActiveFilters = computed(() =>
  Boolean(searchInput.value.trim() || pixelInput.value || segmentInput.value)
)
const offsetFrom = computed(() => (users.value.length ? offset.value + 1 : 0))
const offsetTo = computed(() => offset.value + users.value.length)
const formatNumber = (n: number): string => new Intl.NumberFormat('pt-BR').format(n)

const syncFiltersFromInputs = () => {
  filters.value = {
    search: searchInput.value.trim() || undefined,
    pixel: pixelInput.value.trim() || undefined,
    segment: segmentInput.value || undefined,
  }
}

const fetchUsers = async (silent = false) => {
  const seq = ++usersSeq
  if (!silent) loading.value = true
  try {
    const res = await trackedUsersApi.list({ ...filters.value, limit: PAGE_SIZE, offset: offset.value })
    if (seq !== usersSeq) return
    users.value = res.data || []
    total.value = res.total || 0
  } catch (err: unknown) {
    if (seq !== usersSeq) return
    if (!silent) toast.error('Erro', errorMessage(err, 'Não foi possível carregar os usuários'))
  } finally {
    if (seq === usersSeq) loading.value = false
  }
}

const applyFilters = () => {
  if (_searchDebounceTimer) clearTimeout(_searchDebounceTimer)
  syncFiltersFromInputs()
  offset.value = 0
  void fetchUsers()
}

const resetInputsAndFilters = () => {
  if (_searchDebounceTimer) clearTimeout(_searchDebounceTimer)
  _suppressFilterWatch = true
  searchInput.value = ''
  pixelInput.value = ''
  segmentInput.value = ''
  filters.value = {}
  offset.value = 0
  void nextTick(() => { _suppressFilterWatch = false })
}

const clearFilters = () => { resetInputsAndFilters(); void fetchUsers() }
const refresh = () => { applyFilters() }
const prevPage = () => { offset.value = Math.max(0, offset.value - PAGE_SIZE); void fetchUsers() }
const nextPage = () => { offset.value += PAGE_SIZE; void fetchUsers() }
const onSelect = (u: TrackedUser) => { selected.value = u; modalOpen.value = true }

const onUserSaved = (updated: TrackedUser) => {
  const idx = users.value.findIndex((u) => u.id === updated.id)
  if (idx !== -1) users.value[idx] = updated
  selected.value = updated
}

const onUserDeleted = (deletedId: number) => {
  users.value = users.value.filter((u) => u.id !== deletedId)
  total.value = Math.max(0, total.value - 1)
  if (selected.value?.id === deletedId) selected.value = null
  modalOpen.value = false
  if (users.value.length === 0 && offset.value > 0) offset.value = Math.max(0, offset.value - PAGE_SIZE)
  void fetchUsers(true)
}

watch(searchInput, () => {
  if (_suppressFilterWatch) return
  if (_searchDebounceTimer) clearTimeout(_searchDebounceTimer)
  _searchDebounceTimer = setTimeout(() => applyFilters(), 350)
})

watch([pixelInput, segmentInput], () => {
  if (_suppressFilterWatch) return
  applyFilters()
})

watch(() => liveStore.lastEventSeq, () => {
  if (offset.value > 0 || modalOpen.value) return
  if (_usersRefreshTimer) clearTimeout(_usersRefreshTimer)
  _usersRefreshTimer = setTimeout(() => void fetchUsers(true), 2000)
})

watch(() => tenantStore.switchVersion, () => {
  modalOpen.value = false
  selected.value = null
  resetInputsAndFilters()
  void fetchUsers()
})

onMounted(() => {
  void fetchUsers()
  _unsubEventsUpdated = liveStore.onCustomEvent('events_updated', () => {
    if (offset.value === 0 && !modalOpen.value) void fetchUsers(true)
  })
  _pollingTimer = setInterval(() => {
    if (!liveStore.connected && !document.hidden && offset.value === 0 && !modalOpen.value) {
      void fetchUsers(true)
    }
  }, 30_000)
})

onUnmounted(() => {
  if (_usersRefreshTimer) clearTimeout(_usersRefreshTimer)
  if (_searchDebounceTimer) clearTimeout(_searchDebounceTimer)
  if (_pollingTimer) clearInterval(_pollingTimer)
  if (_unsubEventsUpdated) _unsubEventsUpdated()
})
</script>

<style scoped>
.users-page { display: flex; flex-direction: column; gap: 1.25rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-header > div:first-child { flex: 1; min-width: 280px; }
.page-title { font-family: var(--font-heading); font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); max-width: 640px; }
.filters { display: flex; gap: 0.625rem; align-items: flex-end; flex-wrap: wrap; }
.filter-search { flex: 2; min-width: 240px; }
.filter-select { flex: 1; min-width: 190px; }
.pagination { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 0.75rem 0.25rem; }
.pag-info { font-size: var(--fs-base); color: var(--text-3); }
.pag-buttons { display: flex; gap: 0.375rem; }
</style>
