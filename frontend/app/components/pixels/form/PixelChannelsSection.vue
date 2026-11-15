<template>
  <div>
    <PixelMetaTabSection
      v-show="activeTab === 'meta'"
      :form="form"
      :pixel-ids-error="pixelIdsError"
      :is-editing="isEditing"
      :show-tokens="showTokens"
      :has-saved-manual-meta-token="hasSavedManualMetaToken"
      @update:show-tokens="$emit('update:showTokens', $event)"
      @clear-manual-meta-token="$emit('clearManualMetaToken', $event)"
    />

    <div v-show="activeTab === 'tiktok'" class="tab-content">
      <div class="section-header">
        <h4 class="section-title">
          <BrandIcon channel="tiktok" size="sm" :colored="true" />
          TikTok Events API
        </h4>
        <BaseButton type="button" variant="ghost" size="sm" @click="addTiktokRow">
          + Adicionar pixel
        </BaseButton>
      </div>
      <p class="section-hint">
        Opcional. Se configurado, espelha eventos em tempo real para a API de Eventos do TikTok.
      </p>

      <div v-if="!form.tiktok_pixel_ids.length" class="empty-channel-box">
        Nenhum pixel do TikTok configurado para este tracker.
      </div>
      <div v-for="(_, idx) in form.tiktok_pixel_ids" :key="idx" class="pixel-row">
        <div class="pixel-row-grid">
          <BaseInput
            v-model="form.tiktok_pixel_ids[idx]"
            label="TikTok Pixel ID"
            placeholder="CXXXXXXXXXXXXXX"
          />
          <BaseInput
            v-model="form.tiktok_access_tokens[idx]"
            label="Access Token"
            :type="showTokens ? 'text' : 'password'"
            :placeholder="isEditing ? 'Mantém o atual se vazio' : 'TTAxxxxxxxxxxxx...'"
          />
        </div>
        <button type="button" class="row-remove" aria-label="Remover pixel TikTok" @click="removeTiktokRow(idx)">×</button>
      </div>

      <BaseCheckbox
        v-if="form.tiktok_pixel_ids.length"
        :model-value="showTokens"
        size="sm"
        label="Exibir tokens"
        @update:model-value="$emit('update:showTokens', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import BrandIcon from '~/components/events/BrandIcon.vue'
import PixelMetaTabSection from '~/components/pixels/form/PixelMetaTabSection.vue'
import type { TrackerConfig } from '#shared/types'

interface ChannelsFormState {
  account_id: number | null
  pixel_ids: string[]
  access_tokens: string[]
  meta_account_id: number | null
  meta_ad_account_id: string
  tracker_config: TrackerConfig
  tiktok_pixel_ids: string[]
  tiktok_access_tokens: string[]
}

const props = defineProps<{
  activeTab: string
  form: ChannelsFormState
  pixelIdsError: string
  isEditing: boolean
  showTokens: boolean
  hasSavedManualMetaToken: (pixelId: string) => boolean
}>()

defineEmits<{
  'update:showTokens': [value: boolean]
  clearManualMetaToken: [idx: number]
}>()

const addTiktokRow = () => {
  props.form.tiktok_pixel_ids.push('')
  props.form.tiktok_access_tokens.push('')
}

const removeTiktokRow = (idx: number) => {
  props.form.tiktok_pixel_ids.splice(idx, 1)
  props.form.tiktok_access_tokens.splice(idx, 1)
}
</script>

<style scoped>
.tab-content { display: flex; flex-direction: column; gap: 1rem; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); display: inline-flex; align-items: center; gap: 0.5rem; }
.section-hint { font-size: var(--fs-base); color: var(--text-3); margin: 0; }
.empty-channel-box { padding: 1.5rem; text-align: center; border: 1px dashed var(--border-soft); border-radius: 6px; font-size: var(--fs-md); color: var(--text-3); }
.pixel-row { position: relative; padding: 0.75rem 2.25rem 0.75rem 0.75rem; background: var(--surface-soft); border: 1px solid var(--border-soft); border-radius: 10px; }
.pixel-row-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
@media (max-width: 768px) { .pixel-row-grid { grid-template-columns: 1fr; } }
.row-remove { position: absolute; top: 0.25rem; right: 0.5rem; background: transparent; border: none; color: var(--text-3); font-size: var(--fs-2xl); line-height: 1; cursor: pointer; padding: 0.25rem 0.5rem; border-radius: 6px; transition: all 0.15s; }
.row-remove:hover { color: var(--danger); background: var(--danger-soft); }
</style>
