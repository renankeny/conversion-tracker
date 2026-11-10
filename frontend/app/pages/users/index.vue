<template>
  <div class="users-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Usuários Rastreados</h1>
        <p class="page-subtitle">Visitantes identificados no site.</p>
      </div>
      <BaseButton variant="outline" size="sm" :loading="loading" @click="refresh">
        Atualizar
      </BaseButton>
    </div>

    <div class="filters">
      <BaseInput
        v-model="searchInput"
        placeholder="Buscar usuário"
        @keyup.enter="applyFilters"
      >
        <template #icon>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
          </svg>
        </template>
      </BaseInput>
      <PixelDropdown v-model="pixelInput" />
      <BaseButton variant="primary" size="md" @click="applyFilters">
        Buscar
      </BaseButton>
      <BaseButton variant="ghost" size="md" @click="clearFilters">
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

    <UserDetailsModal v-model="modalOpen" :user="selected" @saved="onUserSaved" />
  </div>
</template>

<script setup lang="ts">
import UsersTable from '~/components/users/UsersTable.vue'
import UserDetailsModal from '~/components/users/UserDetailsModal.vue'
import { useLiveEventsStore } from '~/stores/useLiveEventsStore'
import { useTenantStore } from '~/stores/useTenantStore'
import type { TrackedUser, TrackedUserFilters } from '#shared/types'

definePageMeta({ layout: 'app', middleware: 'auth' })
const { formatTitle } = useBranding()
useHead({ title: computed(() => formatTitle('Usuários')) })

const trackedUsersApi = useTrackedUsers()
const toast = useToast()
const liveStore = useLiveEventsStore()
const tenantStore = useTenantStore()

let _usersRefreshTimer: ReturnType<typeof setTimeout> | null = null
let _pollingTimer: ReturnType<typeof setInterval> | null = null

watch(() => liveStore.lastEventSeq, () => {
  if (offset.value > 0) return
  if (_usersRefreshTimer) clearTimeout(_usersRefreshTimer)
  _usersRefreshTimer = setTimeout(fetchUsers, 2000)
})

watch(() => tenantStore.switchVersion, () => {
  modalOpen.value = false
  selected.value = null
  searchInput.value = ''
  pixelInput.value = ''
  filters.value = {}
  offset.value = 0
  fetchUsers()
})

onUnmounted(() => {
  if (_usersRefreshTimer) clearTimeout(_usersRefreshTimer)
  if (_pollingTimer) clearInterval(_pollingTimer)
})

const PAGE_SIZE = 50
const searchInput = ref('')
const pixelInput = ref('')
const filters = ref<TrackedUserFilters>({})
const users = ref<TrackedUser[]>([])
const total = ref(0)
const offset = ref(0)
const loading = ref(false)

const modalOpen = ref(false)
const selected = ref<TrackedUser | null>(null)

const offsetFrom = computed(() => (users.value.length ? offset.value + 1 : 0))
const offsetTo = computed(() => offset.value + users.value.length)

const fetchUsers = async () => {
  loading.value = true
  try {
    const response = await trackedUsersApi.list({
      ...filters.value,
      limit: PAGE_SIZE,
      offset: offset.value,
    })
    users.value = response.data || []
    total.value = response.total || 0
  } catch (error: any) {
    toast.error('Erro', error.message || 'Não foi possível carregar os usuários')
  } finally {
    loading.value = false
  }
}

const applyFilters = () => {
  filters.value = {
    search: searchInput.value.trim() || undefined,
    pixel: pixelInput.value.trim() || undefined,
  }
  offset.value = 0
  fetchUsers()
}

const clearFilters = () => {
  searchInput.value = ''
  pixelInput.value = ''
  filters.value = {}
  offset.value = 0
  fetchUsers()
}

const refresh = () => { offset.value = 0; fetchUsers() }
const prevPage = () => { offset.value = Math.max(0, offset.value - PAGE_SIZE); fetchUsers() }
const nextPage = () => { offset.value += PAGE_SIZE; fetchUsers() }

const onSelect = (u: TrackedUser) => { selected.value = u; modalOpen.value = true }

const onUserSaved = (updated: TrackedUser) => {
  const idx = users.value.findIndex((u) => u.id === updated.id)
  if (idx !== -1) users.value[idx] = updated
  selected.value = updated
}

const formatNumber = (n: number): string => new Intl.NumberFormat('pt-BR').format(n)

onMounted(() => {
  fetchUsers()
  _pollingTimer = setInterval(() => {
    if (offset.value === 0) fetchUsers()
  }, 30_000)
})
</script>

<style scoped>
.users-page { display: flex; flex-direction: column; gap: 1.25rem; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.page-header > div:first-child { flex: 1; min-width: 280px; }
.page-title { font-family: 'Instrument Sans', sans-serif; font-size: var(--fs-3xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.25rem; }
.page-subtitle { color: var(--text-2); font-size: var(--fs-lg); max-width: 640px; }
.filters { display: flex; gap: 0.625rem; align-items: flex-end; flex-wrap: wrap; }
.filters > :first-child { flex: 2; min-width: 240px; }
.filters > :nth-child(2) { flex: 1; min-width: 180px; }
.pagination { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 0.75rem 0.25rem; }
.pag-info { font-size: var(--fs-base); color: var(--text-3); }
.pag-buttons { display: flex; gap: 0.375rem; }
</style>
