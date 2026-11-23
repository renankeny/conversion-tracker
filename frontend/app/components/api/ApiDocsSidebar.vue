<script setup lang="ts">
import { Search, Sparkles, ShieldCheck, X } from 'lucide-vue-next'
import type { ApiModuleGroup } from '~/utils/apiCatalog'

defineProps<{
  modules: ApiModuleGroup[]
  activeSection: string
  searchQuery: string
}>()

const emit = defineEmits<{
  'update:activeSection': [id: string]
  'update:searchQuery': [q: string]
}>()

const selectSection = (id: string) => {
  if (id) emit('update:searchQuery', '')
  emit('update:activeSection', id)
}
</script>

<template>
  <aside class="api-sidebar">
    <div class="search-box">
      <Search class="search-icon" />
      <input
        type="text"
        class="search-input"
        :value="searchQuery"
        placeholder="Buscar rota ou recurso..."
        @input="emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
      />
      <button
        v-if="searchQuery"
        type="button"
        class="clear-btn"
        aria-label="Limpar busca"
        @click="emit('update:searchQuery', '')"
      >
        <X class="clear-icon" />
      </button>
    </div>

    <nav class="nav-list" aria-label="Módulos da API">
      <button
        type="button"
        class="nav-item"
        :class="{ active: !searchQuery && activeSection === 'mcp' }"
        @click="selectSection('mcp')"
      >
        <span class="nav-left">
          <Sparkles class="nav-icon" />
          <span>Conectar IA (MCP)</span>
        </span>
        <BaseBadge variant="success">Novo</BaseBadge>
      </button>

      <button
        type="button"
        class="nav-item"
        :class="{ active: !searchQuery && activeSection === 'auth' }"
        @click="selectSection('auth')"
      >
        <span class="nav-left">
          <ShieldCheck class="nav-icon" />
          <span>Autenticação e Escopo</span>
        </span>
      </button>

      <div class="nav-group-label">Módulos REST</div>

      <button
        v-for="mod in modules"
        :key="mod.id"
        type="button"
        class="nav-item"
        :class="{ active: !searchQuery && activeSection === mod.id }"
        @click="selectSection(mod.id)"
      >
        <span class="nav-left">
          <span class="mod-num">{{ mod.badge }}</span>
          <span class="mod-label">{{ mod.title }}</span>
        </span>
        <span class="mod-count">{{ mod.endpoints.length }}</span>
      </button>
    </nav>
  </aside>
</template>

<style scoped>
.api-sidebar { width: 268px; flex-shrink: 0; display: flex; flex-direction: column; gap: 0.875rem; }
.search-box { position: relative; display: flex; align-items: center; background: var(--surface-deep); border: 1px solid var(--border); border-radius: 10px; padding: 0 0.75rem; }
.search-box:focus-within { border-color: var(--brand-border); }
.search-icon { width: 0.9rem; height: 0.9rem; color: var(--text-3); flex-shrink: 0; }
.search-input { width: 100%; padding: 0.6rem 0.5rem; background: transparent; border: none; outline: none; color: var(--text-1); font-family: var(--font-sans); font-size: var(--fs-md); }
.search-input::placeholder { color: var(--text-3); }
.clear-btn { background: transparent; border: none; color: var(--text-3); cursor: pointer; display: inline-flex; padding: 0.2rem; }
.clear-icon { width: 0.85rem; height: 0.85rem; }
.nav-list { display: flex; flex-direction: column; gap: 0.25rem; }
.nav-group-label { margin: 0.625rem 0.5rem 0.25rem; font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.06em; color: var(--text-3); }
.nav-item { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; width: 100%; padding: 0.55rem 0.75rem; border-radius: 9px; border: 1px solid transparent; background: transparent; color: var(--text-2); font-family: var(--font-sans); font-size: var(--fs-md); font-weight: var(--fw-medium); cursor: pointer; text-align: left; transition: all 0.15s ease; }
.nav-item:hover { background: var(--surface-soft); color: var(--text-1); }
.nav-item.active { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); font-weight: var(--fw-semibold); }
.nav-left { display: flex; align-items: center; gap: 0.55rem; min-width: 0; }
.nav-icon { width: 0.95rem; height: 0.95rem; flex-shrink: 0; }
.mod-num { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); flex-shrink: 0; }
.nav-item.active .mod-num { color: var(--brand); }
.mod-label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mod-count { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); background: var(--surface-soft); padding: 0.1rem 0.4rem; border-radius: 6px; }
@media (max-width: 1023px) {
  .api-sidebar { width: 100%; }
  .nav-list { flex-direction: row; overflow-x: auto; padding-bottom: 0.25rem; }
  .nav-group-label { display: none; }
  .nav-item { width: auto; flex-shrink: 0; }
}
</style>
