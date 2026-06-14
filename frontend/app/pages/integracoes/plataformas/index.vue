<template>
  <div class="platforms-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Plataformas</h1>
        <p class="page-subtitle">
          Gateways de checkout suportados. Cole a URL do webhook na configuração
          de "compra aprovada" da plataforma — o resto é automático.
        </p>
      </div>
    </div>

    <div class="platforms-grid">
      <div v-for="p in PLATFORMS" :key="p.id" class="platform-card">
        <div class="platform-header">
          <div class="platform-logo" :style="{ background: p.brandColor }">
            {{ p.name.charAt(0) }}
          </div>
          <div class="platform-info">
            <div class="platform-name">
              {{ p.name }}
              <BaseBadge v-if="p.needsHmac" variant="info">HMAC</BaseBadge>
            </div>
            <p class="platform-description">{{ p.description }}</p>
          </div>
        </div>

        <div class="webhook-block">
          <span class="webhook-label">URL do webhook</span>
          <div class="webhook-row">
            <code class="webhook-url">{{ buildURL(p.webhookPath) }}</code>
            <BaseButton size="sm" variant="ghost" @click="copy(p)">
              {{ copiedId === p.id ? '✓ Copiado' : 'Copiar' }}
            </BaseButton>
          </div>
        </div>

        <div class="platform-meta">
          <div class="meta-item">
            <span class="meta-label">Eventos cobertos</span>
            <ul class="meta-list">
              <li v-for="e in p.events" :key="e">{{ e }}</li>
            </ul>
          </div>

          <div class="meta-item">
            <span class="meta-label">Como configurar</span>
            <ol class="setup-steps">
              <li v-for="(step, i) in p.setupSteps" :key="i">{{ step }}</li>
            </ol>
          </div>

          <p v-if="p.notes" class="platform-notes">
            <strong>Nota:</strong> {{ p.notes }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { PLATFORMS, type PlatformDef } from '#shared/constants/platforms'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

useHead({ title: 'Plataformas - TrackFind' })

const toast = useToast()
const config = useRuntimeConfig()

// API_URL com fallback. Em produção vem de NUXT_PUBLIC_API_URL; em dev,
// localhost. Removo /api/v1 do final pra não duplicar com o webhookPath.
const apiBase = computed(() => {
  const raw = String(config.public.apiUrl || '')
  return raw || 'http://localhost:8080/api/v1'
})

const buildURL = (path: string): string => {
  return apiBase.value.replace(/\/+$/, '') + path
}

const copiedId = ref<string | null>(null)

const copy = async (p: PlatformDef) => {
  const url = buildURL(p.webhookPath)
  try {
    await navigator.clipboard.writeText(url)
    copiedId.value = p.id
    toast.success('URL copiada', `Cole no painel da ${p.name}`)
    setTimeout(() => {
      if (copiedId.value === p.id) copiedId.value = null
    }, 2000)
  } catch {
    toast.error('Erro', 'Não foi possível copiar — selecione manualmente.')
  }
}
</script>

<style scoped>
.platforms-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.page-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.page-subtitle {
  color: var(--text-2);
  font-size: 0.9375rem;
  max-width: 720px;
}

.platforms-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
  gap: 1rem;
}

.platform-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  box-shadow: var(--shadow-card);
}

.platform-header {
  display: flex;
  gap: 0.875rem;
  align-items: flex-start;
}

.platform-logo {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.5rem;
  font-weight: 700;
  color: white;
}

.platform-info {
  flex: 1;
  min-width: 0;
}

.platform-name {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.platform-description {
  font-size: 0.8125rem;
  color: var(--text-2);
  line-height: 1.4;
}

.webhook-block {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.webhook-label {
  font-size: 0.6875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-3);
  font-weight: 600;
}

.webhook-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.625rem;
  background: var(--surface-deep);
  border: 1px solid var(--brand-border);
  border-radius: 8px;
}

.webhook-url {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  color: var(--brand);
  word-break: break-all;
  user-select: all;
}

.platform-meta {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

.meta-item {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.meta-label {
  font-size: 0.6875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-3);
  font-weight: 600;
}

.meta-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.meta-list li {
  font-size: 0.8125rem;
  color: var(--text-1);
  padding-left: 1rem;
  position: relative;
}

.meta-list li::before {
  content: '→';
  color: var(--brand);
  position: absolute;
  left: 0;
}

.setup-steps {
  margin: 0;
  padding-left: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.setup-steps li {
  font-size: 0.8125rem;
  color: var(--text-1);
  line-height: 1.4;
}

.platform-notes {
  font-size: 0.75rem;
  color: var(--text-2);
  background: var(--surface-soft);
  padding: 0.625rem 0.75rem;
  border-radius: 8px;
  border-left: 3px solid var(--brand);
  line-height: 1.5;
}

.platform-notes strong {
  color: var(--text-strong);
}
</style>
