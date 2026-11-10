<template>
  <span class="brand-icon" :class="[size, `brand-${channel}`]" :title="label">
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      :fill="fill"
      aria-hidden="true"
    >
      <path :d="path" />
    </svg>
  </span>
</template>

<script setup lang="ts">
import { siMeta, siTiktok, siGoogleads, siGoogleanalytics } from 'simple-icons'
import type { Channel } from '#shared/types'

interface Props {
  channel: Channel
  size?: 'sm' | 'md' | 'lg'
  /** Quando true usa a cor oficial da marca; false usa currentColor (para chips de status) */
  colored?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  colored: true,   // padrão: cor oficial da marca
})

const icons: Record<Channel, { path: string; hex: string; title: string }> = {
  meta:       { path: siMeta.path,       hex: siMeta.hex,       title: 'Meta'       },
  tiktok:     { path: siTiktok.path,     hex: siTiktok.hex,     title: 'TikTok'     },
  google_ads: { path: siGoogleads.path,  hex: siGoogleads.hex,  title: 'Google Ads' },
  ga4:        { path: siGoogleanalytics.path, hex: siGoogleanalytics.hex, title: 'GA4' },
}

const icon  = computed(() => icons[props.channel] ?? icons.meta)
const path  = computed(() => icon.value.path)
const label = computed(() => icon.value.title)
const fill  = computed(() => props.colored ? `#${icon.value.hex}` : 'currentColor')
</script>

<style scoped>
.brand-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.brand-icon svg {
  display: block;
  width: 100%;
  height: 100%;
}

.brand-icon.sm { width: 14px; height: 14px; }
.brand-icon.md { width: 20px; height: 20px; }
.brand-icon.lg { width: 28px; height: 28px; }
</style>
