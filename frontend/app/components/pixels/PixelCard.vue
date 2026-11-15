<template>
  <div class="pixel-card">
    <div class="card-header">
      <div class="header-top">
        <div class="title-block">
          <div class="name-row">
            <h3 class="card-name">{{ pixel.label || pixel.name }}</h3>
            <code v-if="showSlugPill" class="slug-pill">{{ pixel.name }}</code>
          </div>
          <p v-if="accountName" class="company-tag">{{ accountName }}</p>
        </div>
        <div class="badges-col">
          <BaseBadge :variant="pixel.enabled ? 'success' : 'default'">
            {{ pixel.enabled ? 'Ativo' : 'Pausado' }}
          </BaseBadge>
        </div>
      </div>
    </div>

    <div class="card-platforms">
      <div class="platform-chip" :class="{ inactive: !pixel.ga4_measurement_id }">
        <BrandIcon channel="ga4" size="sm" :colored="!!pixel.ga4_measurement_id" />
        <span class="platform-name">GA4</span>
        <span v-if="pixel.ga4_measurement_id" class="platform-linked">ativo</span>
        <span v-else class="platform-count">0</span>
      </div>

      <div class="platform-chip" :class="{ inactive: !hasMetaAny }">
        <BrandIcon channel="meta" size="sm" :colored="hasMetaAny" />
        <span class="platform-name">Meta</span>
        <span v-if="metaWarning" class="platform-warn">{{ metaWarning }}</span>
        <span v-else-if="pixel.pixel_count" class="platform-linked">
          {{ pixel.pixel_count }}{{ pixel.meta_account_id ? ' · OAuth' : '' }}
        </span>
        <span v-else class="platform-count">0</span>
      </div>

      <div class="platform-chip" :class="{ inactive: !(pixel.tiktok_pixel_count ?? 0) }">
        <BrandIcon channel="tiktok" size="sm" :colored="!!(pixel.tiktok_pixel_count ?? 0)" />
        <span class="platform-name">TikTok</span>
        <span v-if="pixel.tiktok_pixel_count" class="platform-linked">{{ pixel.tiktok_pixel_count }}</span>
        <span v-else class="platform-count">0</span>
      </div>

      <div class="platform-chip" :class="{ inactive: !pixel.google_ads_account_id }">
        <BrandIcon channel="google_ads" size="sm" :colored="!!pixel.google_ads_account_id" />
        <span class="platform-name">Google Ads</span>
        <span v-if="googleAdsWarning" class="platform-warn">{{ googleAdsWarning }}</span>
        <span v-else-if="pixel.google_ads_account_id" class="platform-linked">{{ googleAdsStatusLabel }}</span>
        <span v-else class="platform-count">0</span>
      </div>
    </div>

    <div class="card-domains">
      <template v-if="pixel.domains?.length">
        <span v-for="d in pixel.domains.slice(0, 3)" :key="d" class="domain-pill">{{ d }}</span>
        <span v-if="pixel.domains.length > 3" class="domain-more">+{{ pixel.domains.length - 3 }}</span>
      </template>
      <span v-else class="domain-empty">Sem domínios cadastrados</span>
    </div>

    <div class="card-footer">
      <span class="updated-at">{{ formatDate(pixel.updated_at) }}</span>
      <div class="card-actions">
        <button
          v-if="canEdit && canTransfer"
          type="button"
          class="btn-icon"
          title="Transferir pixel para outra empresa"
          @click="$emit('transfer', pixel)"
        >
          <ArrowRightLeft :size="15" />
        </button>
        <button
          v-if="canEdit"
          type="button"
          class="btn-icon danger"
          title="Excluir pixel"
          @click="$emit('delete', pixel)"
        >
          <Trash2 :size="15" />
        </button>
        <BaseButton v-if="canEdit" variant="ghost" size="sm" @click="$emit('edit', pixel)">
          Editar
        </BaseButton>
        <BaseButton variant="primary" size="sm" @click="$emit('install', pixel)">
          Instalar
        </BaseButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Trash2, ArrowRightLeft } from 'lucide-vue-next'
import BrandIcon from '~/components/events/BrandIcon.vue'
import type { GoogleAdsAccount, MetaAccount, Pixel } from '#shared/types'
import { formatDateSP } from '~/utils/formatDate'

const props = withDefaults(defineProps<{
  pixel: Pixel
  canEdit?: boolean
  canTransfer?: boolean
  accountName?: string
  metaAccount?: MetaAccount | null
  googleAdsAccount?: GoogleAdsAccount | null
}>(), {
  canEdit: true,
  canTransfer: false,
  accountName: '',
  metaAccount: null,
  googleAdsAccount: null,
})

defineEmits<{
  edit: [pixel: Pixel]
  delete: [pixel: Pixel]
  install: [pixel: Pixel]
  transfer: [pixel: Pixel]
}>()

const showSlugPill = computed(() => {
  const lbl = (props.pixel.label || '').trim()
  if (!lbl) return false
  return lbl.toLowerCase() !== props.pixel.name.trim().toLowerCase()
})

const hasMetaAny = computed(() => Boolean((props.pixel.pixel_count ?? 0) > 0 || props.pixel.meta_account_id))

const metaWarning = computed(() => {
  if (props.pixel.meta_account_id && !(props.pixel.pixel_count ?? 0)) return 'sem ID'
  if (props.metaAccount && (!props.metaAccount.connected || props.metaAccount.token_expired)) return 'reconectar'
  return ''
})

const effectiveGadsCid = computed(
  () => (props.pixel.google_ads_customer_id || props.googleAdsAccount?.customer_id || '').trim(),
)

const googleAdsWarning = computed(() => {
  if (!props.pixel.google_ads_account_id || !props.googleAdsAccount) return ''
  if (!props.googleAdsAccount.enabled) return 'pausada'
  if (!props.googleAdsAccount.oauth_connected && !props.googleAdsAccount.connected) return 'reconectar'
  if (!effectiveGadsCid.value) return 'sem conta'
  return ''
})

const googleAdsStatusLabel = computed(() => {
  const name =
    props.pixel.google_ads_customer_name ||
    props.pixel.google_ads_customer_id ||
    props.googleAdsAccount?.customer_name ||
    props.googleAdsAccount?.name ||
    ''
  if (!name) return 'vinculado'
  return name.length > 14 ? `${name.slice(0, 14)}…` : name
})

const formatDate = (iso: string) => formatDateSP(iso, { year: '2-digit' })
</script>

<style scoped>
.pixel-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px; box-shadow: var(--shadow-card); transition: box-shadow 0.2s, border-color 0.2s; display: flex; flex-direction: column; overflow: hidden; }
.pixel-card:hover { box-shadow: var(--shadow-elevated); border-color: var(--brand-border); }
.card-header { padding: 1.125rem 1.25rem 0.875rem; }
.header-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; }
.title-block { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 0.25rem; }
.name-row { display: flex; align-items: center; gap: 0.5rem; min-width: 0; flex-wrap: wrap; }
.card-name { font-family: var(--font-sans); font-size: var(--fs-lg); font-weight: var(--fw-bold); color: var(--text-strong); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; }
.slug-pill { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); background: var(--surface-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.company-tag { font-size: var(--fs-xs); font-weight: var(--fw-medium); color: var(--brand); margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.badges-col { display: flex; align-items: center; gap: 0.375rem; flex-shrink: 0; flex-wrap: wrap; justify-content: flex-end; }
.card-platforms { display: flex; gap: 0.5rem; padding: 0 1.25rem 0.875rem; flex-wrap: wrap; }
.platform-chip { display: inline-flex; align-items: center; gap: 0.375rem; padding: 0.375rem 0.625rem; border-radius: 6px; border: 1px solid var(--border-soft); background: var(--surface-1); font-size: var(--fs-sm); transition: all 0.15s; }
.platform-chip.inactive { opacity: 0.45; }
.platform-name { color: var(--text-2); font-weight: var(--fw-medium); }
.platform-count { color: var(--text-3); font-size: var(--fs-xs); background: var(--surface-3); padding: 0.0625rem 0.3125rem; border-radius: 4px; font-weight: var(--fw-semibold); }
.platform-linked { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--success); background: var(--success-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.platform-warn { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--warning); background: var(--warning-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.card-domains { flex: 1; display: flex; flex-wrap: wrap; gap: 0.375rem; align-items: center; padding: 0.625rem 1.25rem; border-top: 1px solid var(--border-faint); min-height: 42px; }
.domain-pill { font-size: var(--fs-xs); font-weight: var(--fw-medium); background: var(--surface-soft); border: 1px solid var(--border-faint); color: var(--text-2); padding: 0.1875rem 0.5rem; border-radius: 6px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 180px; }
.domain-more { font-size: var(--fs-xs); color: var(--text-3); font-weight: var(--fw-semibold); padding: 0.1875rem 0.375rem; }
.domain-empty { font-size: var(--fs-sm); color: var(--text-3); font-style: italic; }
.card-footer { display: flex; align-items: center; justify-content: space-between; padding: 0.625rem 1rem 0.625rem 1.25rem; border-top: 1px solid var(--border-soft); background: var(--surface-soft-2); }
.updated-at { font-size: var(--fs-xs); color: var(--text-3); }
.card-actions { display: flex; align-items: center; gap: 0.25rem; }
.btn-icon { display: inline-flex; align-items: center; justify-content: center; width: 30px; height: 30px; border: none; background: transparent; color: var(--text-3); border-radius: 6px; cursor: pointer; transition: all 0.15s; }
.btn-icon:hover { background: var(--surface-soft); color: var(--brand); }
.btn-icon.danger:hover { background: var(--danger-soft); color: var(--danger); }
</style>
