<template>
  <BaseModal
    :model-value="modelValue"
    size="lg"
    @update:model-value="handleClose"
  >
    <template #header>
      <UserModalHeader :user="activeUser" />
    </template>

    <div v-if="activeUser" class="user-details-wrapper">
      <UserKpiStrip
        :ltv="activeUser.ltv"
        :time-to-purchase="activeUser.time_to_purchase || journeyData?.time_to_purchase"
        :pageview-count="activeUser.pageview_count"
        :checkout-count="activeUser.checkout_count"
        :total-pageviews="journeyData?.total_pageviews"
        :total-checkouts="journeyData?.total_checkouts"
      />

      <UserModalTabs
        v-model="activeTab"
        :timeline-count="timelineEvents.length"
      />

      <div class="tab-content-area">
        <UserJourneyTab
          v-show="activeTab === 'journey'"
          :user="activeUser"
          :purchase-history="purchaseHistory"
          :loading-purchases="loadingPurchases"
          :journey-data="journeyData"
          :timeline-events="timelineEvents"
          :loading-timeline="loadingTimeline"
        />

        <WhatsAppChatPanel
          v-if="activeTab === 'whatsapp'"
          :tracked-user-id="activeUser.id"
        />

        <UserProfileTab
          v-show="activeTab === 'profile'"
          :user="activeUser"
          :form="form"
          :editing="editing"
        />

        <UserAttributionTab
          v-show="activeTab === 'attribution'"
          :user="activeUser"
          :form="form"
          :editing="editing"
        />

        <UserTechnicalTab
          v-show="activeTab === 'technical'"
          :user="activeUser"
          :form="form"
          :editing="editing"
        />
      </div>
    </div>

    <template #footer>
      <template v-if="editing">
        <BaseButton variant="ghost" @click="cancelEdit">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="saving" @click="onSave">
          Salvar alterações
        </BaseButton>
      </template>
      <template v-else>
        <BaseButton
          v-if="canManageUsers && activeUser"
          variant="danger"
          class="mr-auto"
          @click="showDeleteConfirm = true"
        >
          <Trash2 :size="14" />
          Excluir
        </BaseButton>
        <BaseButton variant="ghost" @click="handleClose(false)">Fechar</BaseButton>
        <BaseButton
          v-if="canManageUsers && activeUser"
          variant="outline"
          @click="startEdit"
        >
          <Pencil :size="14" />
          Editar dados
        </BaseButton>
      </template>
    </template>
  </BaseModal>

  <ConfirmModal
    v-model="showDeleteConfirm"
    title="Excluir usuário rastreado?"
    message="Os dados de sessão, cookies e identificação deste visitante serão removidos permanentemente. Esta ação não pode ser desfeita."
    confirm-label="Excluir permanentemente"
    variant="danger"
    :loading="deleting"
    @confirm="confirmDelete"
  />
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Pencil, Trash2 } from 'lucide-vue-next'
import UserModalHeader from '~/components/users/UserModalHeader.vue'
import UserKpiStrip from '~/components/users/UserKpiStrip.vue'
import UserModalTabs, { type UserModalTabKey } from '~/components/users/UserModalTabs.vue'
import UserJourneyTab from '~/components/users/UserJourneyTab.vue'
import UserProfileTab from '~/components/users/UserProfileTab.vue'
import UserAttributionTab from '~/components/users/UserAttributionTab.vue'
import UserTechnicalTab from '~/components/users/UserTechnicalTab.vue'
import WhatsAppChatPanel from '~/components/whatsapp/WhatsAppChatPanel.vue'
import { useAuthStore } from '~/stores/useAuthStore'
import { errorMessage } from '~/utils/errorMessage'
import type { TrackedUser } from '#shared/types'

interface Props {
  modelValue: boolean
  user: TrackedUser | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [user: TrackedUser]
  deleted: [userId: number]
}>()

const authStore = useAuthStore()
const trackedUsers = useTrackedUsers()
const toast = useToast()

const canManageUsers = computed(() => authStore.canManageUsers)

const detailedUser = ref<TrackedUser | null>(null)
const activeUser = computed<TrackedUser | null>(() => detailedUser.value || props.user)
const activeTab = ref<UserModalTabKey>('journey')
const showDeleteConfirm = ref(false)
const deleting = ref(false)
let detailSeq = 0

const loadDetailedUser = async (id: number) => {
  const seq = ++detailSeq
  try {
    const full = await trackedUsers.find(id)
    if (seq !== detailSeq || !props.modelValue || props.user?.id !== id) return
    detailedUser.value = full
  } catch {
    /* mantém resumo da listagem como fallback */
  }
}

watch(
  [() => props.modelValue, () => props.user?.id],
  ([open, id], [prevOpen, prevId]) => {
    if (open && id) {
      if (!prevOpen || id !== prevId) {
        activeTab.value = 'journey'
        showDeleteConfirm.value = false
        detailedUser.value = props.user ? { ...props.user } : null
        void loadDetailedUser(id)
      }
    } else if (!open) {
      detailSeq++
      showDeleteConfirm.value = false
      if (!props.user) detailedUser.value = null
    }
  },
  { immediate: true }
)

const {
  purchaseHistory,
  loadingPurchases,
  loadingTimeline,
  journeyData,
  timelineEvents,
} = useUserTimeline(activeUser)

const {
  form,
  editing,
  saving,
  startEdit,
  cancelEdit,
  onSave,
} = useUserEditForm(activeUser, (updated) => {
  detailedUser.value = updated
  emit('saved', updated)
})

const confirmDelete = async () => {
  const target = activeUser.value
  if (!target || deleting.value) return
  deleting.value = true
  try {
    await trackedUsers.remove(target.id)
    toast.success('Usuário excluído', 'Os dados do visitante foram removidos com sucesso.')
    showDeleteConfirm.value = false
    emit('deleted', target.id)
    emit('update:modelValue', false)
  } catch (err: unknown) {
    toast.error('Erro ao excluir', errorMessage(err, 'Não foi possível excluir o usuário.'))
  } finally {
    deleting.value = false
  }
}

const handleClose = (v: boolean) => {
  if (!v) {
    showDeleteConfirm.value = false
    if (editing.value) cancelEdit()
  }
  emit('update:modelValue', v)
}
</script>

<style scoped>
.user-details-wrapper {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  width: 100%;
  min-width: 0;
}
.tab-content-area {
  min-height: 280px;
  width: 100%;
  min-width: 0;
}
</style>
