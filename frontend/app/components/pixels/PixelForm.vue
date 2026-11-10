<template>
  <div class="pixel-form">
    <div class="tabs">
      <button type="button" class="tab" :class="{ active: activeTab === 'basic' }" @click="activeTab = 'basic'">Geral</button>
      <button type="button" class="tab" :class="{ active: activeTab === 'ga4' }" @click="activeTab = 'ga4'">
        GA4
        <span v-if="form.ga4_measurement_id.trim()" class="tab-count">1</span>
      </button>
      <button type="button" class="tab" :class="{ active: activeTab === 'meta' }" @click="activeTab = 'meta'">
        Meta
        <span v-if="form.pixel_ids.filter(Boolean).length || form.meta_account_id" class="tab-count">
          {{ form.pixel_ids.filter(Boolean).length || 1 }}
        </span>
      </button>
      <button type="button" class="tab" :class="{ active: activeTab === 'tiktok' }" @click="activeTab = 'tiktok'">
        TikTok
        <span v-if="form.tiktok_pixel_ids.filter(Boolean).length" class="tab-count">{{ form.tiktok_pixel_ids.filter(Boolean).length }}</span>
      </button>
      <button type="button" class="tab" :class="{ active: activeTab === 'gads' }" @click="activeTab = 'gads'">
        Google Ads
        <span v-if="form.google_ads_account_id" class="tab-count">1</span>
      </button>
      <button type="button" class="tab" :class="{ active: activeTab === 'tracker' }" @click="activeTab = 'tracker'">Tracker</button>
    </div>

    <PixelBasicSection
      v-show="activeTab === 'basic'"
      :form="form"
      v-model:domains-input="domainsInput"
      :name-error="errors.name"
      :is-editing="isEditing"
      @submit="onSubmit"
    />

    <PixelChannelsSection
      v-show="activeTab === 'meta' || activeTab === 'tiktok'"
      :active-tab="activeTab"
      :form="form"
      :pixel-ids-error="errors.pixel_ids"
      :is-editing="isEditing"
      v-model:show-tokens="showTokens"
    />

    <PixelAdvancedSection
      v-show="activeTab === 'gads' || activeTab === 'ga4'"
      :active-tab="activeTab"
      :form="form"
    />

    <div v-show="activeTab === 'tracker'" class="tab-content">
      <TrackerConfigEditor v-model="form.tracker_config" />
    </div>
  </div>
</template>

<script setup lang="ts">
import TrackerConfigEditor from '~/components/pixels/TrackerConfigEditor.vue'
import PixelBasicSection from '~/components/pixels/form/PixelBasicSection.vue'
import PixelChannelsSection from '~/components/pixels/form/PixelChannelsSection.vue'
import PixelAdvancedSection from '~/components/pixels/form/PixelAdvancedSection.vue'
import { usePixelForm, type PixelFormTabKey } from '~/composables/usePixelForm'
import type { Pixel, PixelInput } from '#shared/types'

interface Props {
  pixel?: Pixel | null
  initialTab?: PixelFormTabKey
}

const props = withDefaults(defineProps<Props>(), {
  initialTab: 'basic',
})

const emit = defineEmits<{
  submit: [input: PixelInput]
}>()

const {
  isEditing,
  activeTab,
  form,
  domainsInput,
  showTokens,
  errors,
  validate,
  buildPayload,
} = usePixelForm(props)

const onSubmit = () => {
  if (!validate()) return
  emit('submit', buildPayload())
}

defineExpose({ submit: onSubmit })
</script>

<style scoped>
.pixel-form { display: flex; flex-direction: column; gap: 1.25rem; }
.tabs { display: flex; gap: 0.25rem; border-bottom: 1px solid var(--border); padding: 0; margin: 0; overflow-x: auto; white-space: nowrap; -webkit-overflow-scrolling: touch; }
.tab { padding: 0.75rem 1.25rem; background: transparent; border: none; color: var(--text-2); font-size: var(--fs-lg); font-weight: var(--fw-medium); cursor: pointer; position: relative; transition: color 0.15s; flex-shrink: 0; }
.tab:hover { color: var(--text-1); }
.tab.active { color: var(--brand); }
.tab.active::after { content: ''; position: absolute; bottom: -1px; left: 0; right: 0; height: 2px; background: var(--brand); }
.tab-count { display: inline-flex; align-items: center; justify-content: center; min-width: 18px; height: 18px; padding: 0 4px; border-radius: 9999px; background: var(--brand-soft); color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-bold); }
.tab-content { display: flex; flex-direction: column; gap: 1.25rem; }
</style>
