<template>
  <div class="journey-sections">
    <div v-if="trackedUser" class="info-card-box">
      <h4 class="box-subtitle">Inteligência de Tráfego</h4>
      <div class="tracking-meta-grid">
        <div class="meta-item">
          <span class="meta-lbl">User ID</span>
          <span class="meta-val font-mono truncate-val" :title="String(trackedUser.external_id || trackedUser.id)">
            {{ trackedUser.external_id || trackedUser.id }}
          </span>
        </div>
        <div class="meta-item">
          <span class="meta-lbl">Origem</span>
          <span class="meta-val truncate-val">{{ getOriginBadge(trackedUser)?.label || 'Direto' }}</span>
        </div>
        <div v-if="effectiveUtmSource" class="meta-item">
          <span class="meta-lbl">UTM Source</span>
          <span class="meta-val truncate-val" :title="effectiveUtmSource">{{ effectiveUtmSource }}</span>
        </div>
        <div v-if="effectiveUtmMedium" class="meta-item">
          <span class="meta-lbl">UTM Medium</span>
          <span class="meta-val truncate-val" :title="effectiveUtmMedium">{{ effectiveUtmMedium }}</span>
        </div>
        <div v-if="effectiveUtmCampaign" class="meta-item">
          <span class="meta-lbl">Campanha</span>
          <span class="meta-val truncate-val" :title="effectiveUtmCampaign">{{ effectiveUtmCampaign }}</span>
        </div>
        <div v-if="trackedUser.traffic_type" class="meta-item">
          <span class="meta-lbl">Tráfego</span>
          <span class="meta-val truncate-val" :class="trackedUser.traffic_type === 'bot' ? 'text-danger-val' : 'text-brand-val'">
            {{ trafficLabel }}
          </span>
        </div>
      </div>

      <CrmLeadCookiesList :tracked-user="trackedUser" />
    </div>

    <div class="info-card-box">
      <div class="tags-header">
        <h4 class="box-subtitle no-margin">Tags do Lead</h4>
        <button type="button" class="new-tag-link" @click="emit('create-tag')">+ Nova Tag</button>
      </div>

      <div class="tags-list">
        <span
          v-for="tag in tags"
          :key="tag.id"
          class="tag-chip"
          :style="{ backgroundColor: tag.color + '18', color: tag.color, borderColor: tag.color + '40' }"
        >
          {{ tag.name }}
          <button type="button" class="remove-tag-btn" title="Desvincular tag do lead" @click="emit('remove-tag', tag.id)">
            &times;
          </button>
        </span>
        <span v-if="!tags?.length" class="empty-tags-hint">Nenhuma tag vinculada</span>
      </div>

      <div class="add-tag-row">
        <div class="tag-select-wrap">
          <BaseDropdown
            :model-value="selectedTagToAdd"
            size="sm"
            placeholder="Selecione uma tag para vincular..."
            :options="[
              { value: '', label: 'Selecione uma tag para vincular...' },
              ...availableTags.map((t) => ({ value: String(t.id), label: t.name }))
            ]"
            @update:model-value="emit('update:selectedTagToAdd', String($event ?? ''))"
          />
        </div>
        <BaseButton variant="outline" size="sm" :disabled="!selectedTagToAdd" @click="emit('add-tag')">
          Vincular
        </BaseButton>
        <button
          v-if="canManageTags && selectedTagToAdd"
          type="button"
          class="delete-tag-icon-btn"
          title="Excluir esta tag da empresa"
          @click="emit('delete-account-tag', Number(selectedTagToAdd))"
        >
          <Trash2 :size="14" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Trash2 } from 'lucide-vue-next'
import type { CrmTag, TrackedUser } from '#shared/types'
import { useCrmHelpers } from '~/composables/useCrmHelpers'
import CrmLeadCookiesList from '~/components/crm/CrmLeadCookiesList.vue'

const props = withDefaults(defineProps<{
  trackedUser: TrackedUser | null
  tags: CrmTag[]
  allTags: CrmTag[]
  selectedTagToAdd: string
  canManageTags?: boolean
}>(), {
  canManageTags: true,
})

const emit = defineEmits<{
  (e: 'update:selectedTagToAdd', val: string): void
  (e: 'create-tag'): void
  (e: 'add-tag'): void
  (e: 'remove-tag', tagId: number): void
  (e: 'delete-account-tag', tagId: number): void
}>()

const { getOriginBadge, cleanUtmValue } = useCrmHelpers()

const effectiveUtmSource = computed(() => {
  const u = props.trackedUser
  if (!u) return ''
  const last = cleanUtmValue(u.last_utm_source)
  const first = cleanUtmValue(u.first_utm_source)
  const candidate = (last && last.toLowerCase() !== 'direto' ? last : '')
    || (first && first.toLowerCase() !== 'direto' ? first : '')

  if (candidate) return candidate
  if (u.gclid || u.gbraid || u.wbraid) return 'google'
  if (u.fbc || u.ctwa_clid) return 'facebook'
  if (u.ttclid) return 'tiktok'
  return last || first
})

const effectiveUtmMedium = computed(() => {
  const u = props.trackedUser
  if (!u) return ''
  return cleanUtmValue(u.last_utm_medium) || cleanUtmValue(u.first_utm_medium)
})

const effectiveUtmCampaign = computed(() => {
  const u = props.trackedUser
  if (!u) return ''
  return cleanUtmValue(u.last_utm_campaign) || cleanUtmValue(u.first_utm_campaign)
})

const trafficLabel = computed(() => {
  const u = props.trackedUser
  if (!u) return ''
  if (u.traffic_type === 'bot') return 'Provável Robô'
  if (u.traffic_score && u.traffic_score > 0) return `Humano (+${u.traffic_score})`
  return 'Humano'
})

const availableTags = computed(() => {
  const assignedIds = new Set((props.tags || []).map((t) => t.id))
  return (props.allTags || []).filter((t) => !assignedIds.has(t.id))
})
</script>

<style scoped>
.journey-sections {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  min-width: 0;
}

.info-card-box {
  background: var(--surface-deep);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.875rem;
  min-width: 0;
}

.box-subtitle {
  font-size: var(--fs-base);
  font-weight: var(--fw-bold);
  color: var(--text-1);
  margin: 0 0 0.625rem 0;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.box-subtitle.no-margin { margin-bottom: 0; }

.tracking-meta-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0.5rem 0.75rem;
  min-width: 0;
}

.meta-item { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.meta-lbl { font-size: var(--fs-xs); color: var(--text-3); }
.meta-val { font-size: var(--fs-sm); color: var(--text-1); font-weight: var(--fw-semibold); min-width: 0; }
.truncate-val { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.text-danger-val { color: var(--danger); }
.text-brand-val { color: var(--brand); }

.tags-header {
  display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.5rem;
}

.new-tag-link {
  background: none; border: none; font-size: var(--fs-sm); color: var(--brand); cursor: pointer;
}
.new-tag-link:hover { text-decoration: underline; }

.tags-list { display: flex; gap: 0.4rem; flex-wrap: wrap; min-height: 24px; }

.tag-chip {
  padding: 0.2rem 0.5rem; border-radius: 6px; font-size: var(--fs-sm); font-weight: var(--fw-semibold);
  border: 1px solid transparent; display: inline-flex; align-items: center; gap: 4px;
}

.remove-tag-btn {
  background: none; border: none; cursor: pointer; color: inherit;
  font-size: var(--fs-lg); line-height: 1; padding: 0; opacity: 0.7;
}
.remove-tag-btn:hover { opacity: 1; }

.empty-tags-hint { font-size: var(--fs-sm); color: var(--text-3); font-style: italic; }

.add-tag-row { display: flex; align-items: center; gap: 0.5rem; margin-top: 0.5rem; min-width: 0; }
.tag-select-wrap { flex: 1; min-width: 0; }

.delete-tag-icon-btn {
  width: 32px; height: 32px; border-radius: 6px; border: 1px solid var(--border);
  background: var(--surface-1); color: var(--danger); cursor: pointer;
  display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0;
  transition: background 0.15s, border-color 0.15s;
}
.delete-tag-icon-btn:hover {
  background: var(--danger-soft);
  border-color: var(--danger);
}
</style>
