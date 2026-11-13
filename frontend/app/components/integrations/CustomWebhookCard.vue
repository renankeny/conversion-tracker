<template>
  <div class="webhook-card">
    <div class="card-header">
      <h3 class="card-title">{{ item.name }}</h3>
      <BaseBadge v-if="item.account_id === null" variant="info">Global</BaseBadge>
    </div>

    <div class="webhook-url-block">
      <span class="url-label">URL do Webhook:</span>
      <code class="url-code">{{ webhookUrl }}</code>
    </div>

    <div class="card-actions">
      <BaseButton size="sm" variant="ghost" @click="$emit('logs', item)">📋 Logs</BaseButton>
      <BaseButton size="sm" variant="ghost" @click="$emit('edit', item)">Editar</BaseButton>
      <BaseButton size="sm" variant="danger" ghost @click="$emit('delete', String(item.id))">Excluir</BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CustomWebhook } from '#shared/types'

const props = defineProps<{
  item: CustomWebhook
}>()

defineEmits<{
  logs: [item: CustomWebhook]
  edit: [item: CustomWebhook]
  delete: [id: string]
}>()

const config = useRuntimeConfig()
const webhookUrl = computed(() => {
  const base = String(config.public.apiUrl || 'http://localhost:8080/api/v1')
  return `${base}/webhooks/custom/${props.item.id}`
})
</script>

<style scoped>
.webhook-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  padding: 1.25rem;
  border-radius: 12px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}
.card-title {
  font-size: var(--fs-xl);
  font-weight: var(--fw-semibold);
  margin: 0;
}
.webhook-url-block {
  background: var(--surface-1);
  padding: 0.75rem;
  border-radius: 8px;
  font-size: var(--fs-base);
  word-break: break-all;
  margin-bottom: 1rem;
}
.url-label {
  color: var(--text-3);
  display: block;
  margin-bottom: 0.25rem;
  font-weight: var(--fw-semibold);
}
.url-code {
  color: var(--brand);
}
.card-actions {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
}
</style>
