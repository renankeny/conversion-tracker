<template>
  <div class="pixel-card">
    <div class="card-header">
      <div class="card-title-row">
        <h3 class="card-title">{{ pixel.name }}</h3>
        <BaseBadge :variant="pixel.enabled ? 'success' : 'default'">
          {{ pixel.enabled ? 'Ativo' : 'Desativado' }}
        </BaseBadge>
      </div>
      <p v-if="pixel.label" class="card-label">{{ pixel.label }}</p>
    </div>

    <div class="card-body">
      <div class="card-row">
        <span class="row-label">Pixels Meta</span>
        <span class="row-value">
          <code v-for="id in pixel.pixel_ids" :key="id" class="pixel-id">{{ id }}</code>
        </span>
      </div>

      <div class="card-row">
        <span class="row-label">Domínios</span>
        <span class="row-value">
          <span v-for="d in pixel.domains" :key="d" class="domain-pill">{{ d }}</span>
          <span v-if="!pixel.domains?.length" class="muted">nenhum</span>
        </span>
      </div>

      <div class="card-row">
        <span class="row-label">Test Codes</span>
        <span class="row-value">
          <code v-for="(tc, i) in pixel.test_codes" :key="i" class="test-code">
            {{ tc || '—' }}
          </code>
          <span v-if="!pixel.test_codes?.length" class="muted">nenhum</span>
        </span>
      </div>
    </div>

    <div class="card-footer">
      <span class="updated-at">Atualizado {{ formatDate(pixel.updated_at) }}</span>
      <div class="card-actions">
        <BaseButton variant="primary" size="sm" @click="$emit('install', pixel)">
          <template #iconLeft>
            <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
              <path d="M8 2a1 1 0 000 2h2a1 1 0 100-2H8z" />
              <path fill-rule="evenodd" d="M3 5a2 2 0 012-2h1a2 2 0 002 2h4a2 2 0 002-2h1a2 2 0 012 2v11a2 2 0 01-2 2H5a2 2 0 01-2-2V5zm5.5 7.5l1.5 1.5 4-4-1.06-1.06L10 11.94l-.94-.94L8.5 12.5z" clip-rule="evenodd" />
            </svg>
          </template>
          Instalar
        </BaseButton>
        <BaseButton variant="ghost" size="sm" @click="$emit('edit', pixel)">
          Editar
        </BaseButton>
        <BaseButton variant="ghost" size="sm" @click="$emit('delete', pixel)">
          <span class="text-danger">Excluir</span>
        </BaseButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Pixel } from '#shared/types'

defineProps<{
  pixel: Pixel
}>()

defineEmits<{
  edit: [pixel: Pixel]
  delete: [pixel: Pixel]
  install: [pixel: Pixel]
}>()

const formatDate = (iso: string) => {
  if (!iso) return ''
  return new Date(iso).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<style scoped>
.pixel-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: var(--shadow-card);
  transition: all 0.2s ease;
  display: flex;
  flex-direction: column;
}

.pixel-card:hover {
  box-shadow: var(--shadow-elevated);
  border-color: var(--brand-border);
}

.card-header {
  padding: 1.25rem 1.25rem 0.75rem;
}

.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.card-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-strong);
}

.card-label {
  font-size: 0.875rem;
  color: var(--text-2);
  margin-top: 0.25rem;
}

.card-body {
  padding: 0.75rem 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
  flex: 1;
}

.card-row {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.row-label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-3);
  font-weight: 600;
}

.row-value {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
  font-size: 0.875rem;
}

.pixel-id,
.test-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  background: var(--surface-soft);
  border: 1px solid var(--border-soft);
  padding: 0.125rem 0.5rem;
  border-radius: 6px;
  color: var(--text-1);
}

.domain-pill {
  font-size: 0.75rem;
  background: var(--brand-soft);
  color: var(--brand);
  padding: 0.125rem 0.625rem;
  border-radius: 9999px;
  font-weight: 500;
}

.muted {
  color: var(--text-3);
  font-size: 0.8125rem;
  font-style: italic;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-top: 1px solid var(--border-soft);
  margin-top: auto;
}

.updated-at {
  font-size: 0.75rem;
  color: var(--text-3);
}

.card-actions {
  display: flex;
  gap: 0.25rem;
}

.text-danger {
  color: var(--danger);
}
</style>
