<template>
  <div v-if="capturedCookies.length > 0" class="cookies-strip">
    <span class="cookies-strip-label">Click IDs &amp; Cookies Capturados</span>
    <div class="cookies-rows">
      <div v-for="item in capturedCookies" :key="item.key" class="cookie-row">
        <div class="cookie-key-group">
          <BrandIcon :channel="item.channel" size="sm" />
          <span class="cookie-key">{{ item.label }}</span>
        </div>
        <span class="cookie-val" :title="item.value">{{ item.value }}</span>
        <button
          type="button"
          class="copy-btn"
          :class="{ copied: copiedKey === item.key }"
          :title="`Copiar ${item.label}`"
          @click="copyValue(item.key, item.value)"
        >
          <Check v-if="copiedKey === item.key" :size="12" />
          <Copy v-else :size="12" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Copy, Check } from 'lucide-vue-next'
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { TrackedUser } from '#shared/types'

const props = defineProps<{
  trackedUser: TrackedUser | null
}>()

interface CookieItem {
  key: string
  label: string
  channel: 'meta' | 'google_ads' | 'tiktok' | 'ga4'
  value: string
}

const copiedKey = ref<string | null>(null)

const capturedCookies = computed<CookieItem[]>(() => {
  const u = props.trackedUser
  if (!u) return []

  const candidates: Array<{ key: string; label: string; channel: CookieItem['channel']; value?: string }> = [
    { key: 'gclid', label: 'GCLID', channel: 'google_ads', value: u.gclid },
    { key: 'gbraid', label: 'GBraid', channel: 'google_ads', value: u.gbraid },
    { key: 'wbraid', label: 'WBraid', channel: 'google_ads', value: u.wbraid },
    { key: 'ctwa_clid', label: 'CTWA Clid', channel: 'meta', value: u.ctwa_clid },
    { key: 'fbc', label: 'FBC', channel: 'meta', value: u.fbc },
    { key: 'fbp', label: 'FBP', channel: 'meta', value: u.fbp },
    { key: 'ttclid', label: 'TTClid', channel: 'tiktok', value: u.ttclid },
    { key: 'ttp', label: 'TTP', channel: 'tiktok', value: u.ttp },
    { key: 'ga4_client_id', label: 'GA4 Client', channel: 'ga4', value: u.ga4_client_id },
    { key: 'ga4_session_id', label: 'GA4 Session', channel: 'ga4', value: u.ga4_session_id },
  ]

  return candidates
    .filter((c): c is CookieItem => Boolean(c.value && c.value.trim()))
    .map((c) => ({ ...c, value: c.value.trim() }))
})

const copyValue = async (key: string, value: string) => {
  try {
    await navigator.clipboard.writeText(value)
    copiedKey.value = key
    setTimeout(() => {
      if (copiedKey.value === key) copiedKey.value = null
    }, 2000)
  } catch {
    /* silent */
  }
}
</script>

<style scoped>
.cookies-strip {
  margin-top: 0.75rem;
  padding-top: 0.625rem;
  border-top: 1px solid var(--border-soft);
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  min-width: 0;
}

.cookies-strip-label {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.cookies-rows {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  min-width: 0;
}

.cookie-row {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.5rem;
  padding: 0.3rem 0.5rem;
  border-radius: 6px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  min-width: 0;
}

.cookie-key-group {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
}

.cookie-key {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
  white-space: nowrap;
}

.cookie-val {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.copy-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--border-soft);
  border-radius: 4px;
  color: var(--text-3);
  padding: 3px;
  cursor: pointer;
  flex-shrink: 0;
  transition: color 0.15s, border-color 0.15s;
}

.copy-btn:hover {
  color: var(--brand);
  border-color: var(--brand);
}

.copy-btn.copied {
  color: var(--success);
  border-color: var(--success);
}
</style>
