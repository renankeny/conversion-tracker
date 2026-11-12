<template>
  <div class="origin-banner">
    <div class="origin-main">
      <img
        v-if="chat.ad_thumbnail_url"
        :src="chat.ad_thumbnail_url"
        alt=""
        class="ad-thumb"
      >
      <div class="origin-info">
        <div class="origin-badges">
          <BaseBadge :variant="originBadge.variant">{{ originBadge.label }}</BaseBadge>
          <span v-if="chat.source_app" class="channel-chip">{{ chat.source_app }}</span>
          <span v-if="chat.instance_name" class="inst-chip">{{ chat.instance_name }}</span>
        </div>

        <p v-if="adHierarchy" class="ad-hierarchy" :title="adHierarchy">
          {{ adHierarchy }}
        </p>
      </div>
    </div>

    <span class="readonly-hint">Somente leitura</span>
  </div>
</template>

<script setup lang="ts">
import type { WhatsAppChat } from '#shared/types'

const props = defineProps<{
  chat: WhatsAppChat
}>()

const originBadge = computed<{ label: string; variant: 'success' | 'info' | 'warning' | 'default' }>(() => {
  switch (props.chat.origin_type) {
    case 'meta_ads_ctwa':
      return { label: 'Meta Ads · CTWA', variant: 'success' }
    case 'meta_ads_lp':
      return { label: 'Meta Ads · Site (FBC)', variant: 'info' }
    case 'google_ads':
      return { label: 'Google Ads · Site', variant: 'info' }
    case 'tiktok_ads':
      return { label: 'TikTok Ads · Site', variant: 'info' }
    case 'bio_link_organic':
      return { label: 'Orgânico · Link na Bio', variant: 'warning' }
    case 'website_organic':
      return { label: 'Orgânico · Site', variant: 'default' }
    default:
      return { label: 'WhatsApp Rastreado', variant: 'default' }
  }
})

const adHierarchy = computed(() => {
  const parts = [props.chat.campaign_name, props.chat.adset_name, props.chat.ad_name].filter(Boolean)
  if (parts.length) return parts.join(' › ')
  if (props.chat.source_id) return `Anúncio ID: ${props.chat.source_id}`
  return ''
})
</script>

<style scoped>
.origin-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0.875rem;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  border-radius: 9px;
  flex-wrap: wrap;
}
.origin-main { display: flex; align-items: center; gap: 0.625rem; min-width: 0; flex: 1; }
.ad-thumb { width: 34px; height: 34px; border-radius: 6px; object-fit: cover; flex-shrink: 0; border: 1px solid var(--border); }
.origin-info { display: flex; flex-direction: column; gap: 0.25rem; min-width: 0; }
.origin-badges { display: flex; align-items: center; gap: 0.4rem; flex-wrap: wrap; }
.channel-chip, .inst-chip {
  font-size: var(--fs-2xs); color: var(--text-3); text-transform: uppercase;
  padding: 0.125rem 0.375rem; background: var(--surface-soft); border-radius: 4px;
}
.ad-hierarchy {
  font-size: var(--fs-xs); color: var(--text-2); margin: 0;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.readonly-hint { font-size: var(--fs-2xs); color: var(--text-3); text-transform: uppercase; letter-spacing: 0.04em; white-space: nowrap; flex-shrink: 0; }
</style>

