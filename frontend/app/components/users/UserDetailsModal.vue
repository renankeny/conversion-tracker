<template>
  <BaseModal
    :model-value="modelValue"
    size="lg"
    @update:model-value="handleClose"
  >
    <template #header>
      <UserModalHeader :user="user" />
    </template>

    <div v-if="user" class="user-details-wrapper">
      <UserKpiStrip
        :ltv="user.ltv"
        :time-to-purchase="user.time_to_purchase"
        :pageview-count="user.pageview_count"
        :checkout-count="user.checkout_count"
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
          :user="user"
          :purchase-history="purchaseHistory"
          :loading-purchases="loadingPurchases"
          :journey-data="journeyData"
          :timeline-events="timelineEvents"
          :loading-timeline="loadingTimeline"
        />

        <WhatsAppChatPanel
          v-if="activeTab === 'whatsapp'"
          :tracked-user-id="user.id"
        />

        <UserProfileTab
          v-show="activeTab === 'profile'"
          :user="user"
          :form="form"
          :editing="editing"
        />

        <UserAttributionTab
          v-show="activeTab === 'attribution'"
          :user="user"
          :form="form"
          :editing="editing"
        />

        <UserTechnicalTab
          v-show="activeTab === 'technical'"
          :user="user"
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
        <BaseButton variant="ghost" @click="handleClose(false)">Fechar</BaseButton>
        <BaseButton variant="outline" @click="startEdit">
          <Pencil class="w-3.5 h-3.5 mr-1.5" />
          Editar dados
        </BaseButton>
      </template>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, toRef } from 'vue'
import { Pencil } from 'lucide-vue-next'
import UserModalHeader from '~/components/users/UserModalHeader.vue'
import UserKpiStrip from '~/components/users/UserKpiStrip.vue'
import UserModalTabs, { type UserModalTabKey } from '~/components/users/UserModalTabs.vue'
import UserJourneyTab from '~/components/users/UserJourneyTab.vue'
import UserProfileTab from '~/components/users/UserProfileTab.vue'
import UserAttributionTab from '~/components/users/UserAttributionTab.vue'
import UserTechnicalTab from '~/components/users/UserTechnicalTab.vue'
import WhatsAppChatPanel from '~/components/whatsapp/WhatsAppChatPanel.vue'
import type { TrackedUser } from '#shared/types'

interface Props {
  modelValue: boolean
  user: TrackedUser | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [user: TrackedUser]
}>()

const userRef = toRef(props, 'user')
const activeTab = ref<UserModalTabKey>('journey')

const {
  purchaseHistory,
  loadingPurchases,
  loadingTimeline,
  journeyData,
  timelineEvents,
} = useUserTimeline(userRef)

const {
  form,
  editing,
  saving,
  startEdit,
  cancelEdit,
  onSave,
} = useUserEditForm(userRef, (updated) => emit('saved', updated))

const handleClose = (v: boolean) => {
  if (!v && editing.value) {
    cancelEdit()
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
