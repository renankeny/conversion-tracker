<script setup lang="ts">
import { Eye, EyeOff, Copy, Check, Pencil, RefreshCw } from 'lucide-vue-next'
import { API_KEY_SCOPE_OPTIONS, type APIKey } from '#shared/types'
import { formatDateTimeSP } from '~/utils/formatDate'

const props = defineProps<{
  item: APIKey
  revealed: boolean
  copied: boolean
}>()

const emit = defineEmits<{
  'toggle-reveal': []
  copy: []
  edit: []
  regenerate: []
  revoke: []
}>()

const scopeLabelMap = Object.fromEntries(API_KEY_SCOPE_OPTIONS.map((o) => [o.value, o.label]))

const scopeBadges = computed(() => {
  if (!props.item.scopes?.length || props.item.scopes.includes('*')) return ['Acesso Total (8 módulos)']
  return props.item.scopes.map((s) => scopeLabelMap[s] || s)
})

const formatDate = (iso: string): string => formatDateTimeSP(iso)
</script>

<template>
  <div class="key-row" :class="{ 'is-revoked': !!item.revoked_at }">
    <div class="key-top">
      <div class="key-title-group">
        <span class="key-name">{{ item.name }}</span>
        <BaseBadge v-if="item.revoked_at" variant="danger">revogada</BaseBadge>
        <BaseBadge v-else variant="success">API + MCP</BaseBadge>
        <BaseBadge v-if="item.read_only && !item.revoked_at" variant="warning">somente leitura</BaseBadge>
      </div>

      <div v-if="!item.revoked_at" class="key-actions">
        <BaseButton variant="ghost" size="sm" @click="emit('edit')">
          <Pencil class="icon-xs" />
          Editar
        </BaseButton>
        <BaseButton variant="ghost" size="sm" @click="emit('regenerate')">
          <RefreshCw class="icon-xs" />
          Regerar
        </BaseButton>
        <BaseButton variant="ghost" size="sm" @click="emit('revoke')">
          <span class="text-danger">Revogar</span>
        </BaseButton>
      </div>
    </div>

    <div v-if="!item.revoked_at" class="secret-bar">
      <template v-if="item.plaintext">
        <code class="secret-code">
          {{ revealed ? item.plaintext : `${item.key_prefix}••••••••••••••••••••••••••••••••` }}
        </code>
        <div class="secret-btns">
          <button type="button" class="mini-btn" @click="emit('toggle-reveal')">
            <EyeOff v-if="revealed" class="icon-xs" />
            <Eye v-else class="icon-xs" />
            <span>{{ revealed ? 'Ocultar' : 'Ver chave' }}</span>
          </button>
          <button type="button" class="mini-btn mini-btn-brand" @click="emit('copy')">
            <Check v-if="copied" class="icon-xs" />
            <Copy v-else class="icon-xs" />
            <span>{{ copied ? 'Copiado' : 'Copiar' }}</span>
          </button>
        </div>
      </template>
      <template v-else>
        <code class="secret-code muted-code">{{ item.key_prefix }}... (chave antiga sem cópia salva)</code>
        <button type="button" class="mini-btn" @click="emit('regenerate')">
          <RefreshCw class="icon-xs" />
          <span>Regerar para visualizar</span>
        </button>
      </template>
    </div>

    <div class="key-footer">
      <div class="scope-pills">
        <span v-for="label in scopeBadges" :key="label" class="scope-pill">{{ label }}</span>
      </div>
      <span class="key-meta">
        empresa #{{ item.account_id }} · criada {{ formatDate(item.created_at) }}
        <span v-if="item.last_used_at"> · último uso {{ formatDate(item.last_used_at) }}</span>
        <span v-else> · nunca usada</span>
      </span>
    </div>
  </div>
</template>

<style scoped>
.key-row { display: flex; flex-direction: column; gap: 0.65rem; padding: 0.95rem 1.05rem; border: 1px solid var(--border); border-radius: 11px; background: var(--surface-soft-2); }
.key-row.is-revoked { opacity: 0.6; }
.key-top { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem; }
.key-title-group { display: flex; align-items: center; flex-wrap: wrap; gap: 0.45rem; }
.key-name { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); }
.key-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 0.25rem; }
.text-danger { color: var(--danger); }
.icon-xs { width: 0.85rem; height: 0.85rem; }
.secret-bar { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem; padding: 0.55rem 0.75rem; background: var(--surface-deep); border: 1px solid var(--border); border-radius: 8px; }
.secret-code { font-family: var(--font-mono); font-size: var(--fs-base); color: var(--brand); word-break: break-all; }
.muted-code { color: var(--text-3); }
.secret-btns { display: flex; align-items: center; gap: 0.4rem; }
.mini-btn { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.28rem 0.6rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface-soft); color: var(--text-1); font-size: var(--fs-xs); font-weight: var(--fw-medium); cursor: pointer; }
.mini-btn:hover { border-color: var(--brand-border); }
.mini-btn-brand { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); }
.key-footer { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem; }
.scope-pills { display: flex; flex-wrap: wrap; gap: 0.3rem; }
.scope-pill { font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-deep); border: 1px solid var(--border); padding: 0.12rem 0.48rem; border-radius: 6px; }
.key-meta { font-size: var(--fs-xs); color: var(--text-3); }
</style>
