<template>
  <div class="pixel-card">
    <div class="card-header">
      <div class="header-top">
        <div class="title-block">
          <h3 class="card-name">{{ pixel.name }}</h3>
          <p v-if="pixel.label" class="card-label">{{ pixel.label }}</p>
        </div>
        <BaseBadge :variant="pixel.enabled ? 'success' : 'default'">
          {{ pixel.enabled ? 'Ativo' : 'Pausado' }}
        </BaseBadge>
      </div>
    </div>

    <div class="card-platforms">
      <div class="platform-chip" :class="{ inactive: !pixel.ga4_measurement_id }">
        <BrandIcon channel="ga4" size="sm" :colored="!!pixel.ga4_measurement_id" />
        <span class="platform-name">GA4</span>
        <span v-if="pixel.ga4_measurement_id" class="platform-linked">vinculado</span>
        <span v-else class="platform-count">—</span>
      </div>

      <div class="platform-chip" :class="{ inactive: !pixel.pixel_count }">
        <BrandIcon channel="meta" size="sm" :colored="!!pixel.pixel_count" />
        <span class="platform-name">Meta</span>
        <span class="platform-count">{{ pixel.pixel_count ?? 0 }}</span>
      </div>

      <div class="platform-chip" :class="{ inactive: !(pixel.tiktok_pixel_count ?? 0) }">
        <BrandIcon channel="tiktok" size="sm" :colored="!!(pixel.tiktok_pixel_count ?? 0)" />
        <span class="platform-name">TikTok</span>
        <span class="platform-count">{{ pixel.tiktok_pixel_count ?? 0 }}</span>
      </div>

      <div class="platform-chip" :class="{ inactive: !pixel.google_ads_account_id }">
        <BrandIcon channel="google_ads" size="sm" :colored="!!pixel.google_ads_account_id" />
        <span class="platform-name">Google Ads</span>
        <span v-if="pixel.google_ads_account_id" class="platform-linked">vinculado</span>
        <span v-else class="platform-count">—</span>
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
        <button v-if="canEdit !== false" class="btn-icon" title="Excluir pixel" @click="$emit('delete', pixel)">
          <Trash2 :size="15" />
        </button>
        <button v-if="canTransfer" class="btn-icon" title="Transferir pixel" @click="$emit('transfer', pixel)">
          <ArrowRightLeft :size="15" />
        </button>
        <BaseButton v-if="canEdit !== false" variant="ghost" size="sm" @click="$emit('edit', pixel)">
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
import type { Pixel } from '#shared/types'
import { formatDateSP } from '~/utils/formatDate'

defineProps<{ pixel: Pixel; canEdit?: boolean }>()

const auth = useAuth()
const canTransfer = computed(() => {
  const role = auth.user.value?.role
  return role === 'superadmin' || role === 'owner' || role === 'admin'
})

defineEmits<{
  edit: [pixel: Pixel]
  delete: [pixel: Pixel]
  install: [pixel: Pixel]
  transfer: [pixel: Pixel]
}>()

const formatDate = (iso: string) => formatDateSP(iso, { year: '2-digit' })
</script>

<style scoped>
.pixel-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px; box-shadow: var(--shadow-card); transition: box-shadow 0.2s, border-color 0.2s; display: flex; flex-direction: column; overflow: hidden; }
.pixel-card:hover { box-shadow: var(--shadow-elevated); border-color: var(--brand-border); }
.card-header { padding: 1.125rem 1.25rem 0.875rem; }
.header-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; }
.title-block { flex: 1; min-width: 0; }
.card-name { font-family: var(--font-sans); font-size: var(--fs-lg); font-weight: var(--fw-bold); color: var(--text-strong); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.card-label { font-size: var(--fs-base); color: var(--text-3); margin-top: 0.125rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.card-platforms { display: flex; gap: 0.5rem; padding: 0 1.25rem 0.875rem; flex-wrap: wrap; }
.platform-chip { display: inline-flex; align-items: center; gap: 0.375rem; padding: 0.375rem 0.625rem; border-radius: 6px; border: 1px solid var(--border-soft); background: var(--surface-1); font-size: var(--fs-sm); transition: all 0.15s; }
.platform-chip.inactive { opacity: 0.45; }
.platform-name { color: var(--text-2); font-weight: var(--fw-medium); }
.platform-count { color: var(--text-3); font-size: var(--fs-xs); background: var(--surface-3); padding: 0.0625rem 0.3125rem; border-radius: 4px; font-weight: var(--fw-semibold); }
.platform-linked { font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--success); background: var(--success-soft); padding: 0.0625rem 0.375rem; border-radius: 4px; }
.card-domains { flex: 1; display: flex; flex-wrap: wrap; gap: 0.375rem; align-items: center; padding: 0.625rem 1.25rem; border-top: 1px solid var(--border-faint); min-height: 42px; }
.domain-pill { font-size: var(--fs-xs); font-weight: var(--fw-medium); background: var(--surface-soft); border: 1px solid var(--border-faint); color: var(--text-2); padding: 0.1875rem 0.5rem; border-radius: 6px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 180px; }
.domain-more { font-size: var(--fs-xs); color: var(--text-3); font-weight: var(--fw-semibold); padding: 0.1875rem 0.375rem; }
.domain-empty { font-size: var(--fs-sm); color: var(--text-3); font-style: italic; }
.card-footer { display: flex; align-items: center; justify-content: space-between; padding: 0.625rem 1rem 0.625rem 1.25rem; border-top: 1px solid var(--border-soft); background: var(--surface-soft-2); }
.updated-at { font-size: var(--fs-xs); color: var(--text-3); }
.card-actions { display: flex; align-items: center; gap: 0.25rem; }
.btn-icon { display: inline-flex; align-items: center; justify-content: center; width: 30px; height: 30px; border: none; background: transparent; color: var(--text-3); border-radius: 6px; cursor: pointer; transition: all 0.15s; }
.btn-icon:hover { background: var(--danger-soft); color: var(--danger); }
</style>
