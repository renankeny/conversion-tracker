<template>
  <div class="preview-card">
    <div class="preview-header">
      <div>
        <h3 class="preview-title">Pré-visualização em Tempo Real</h3>
        <p class="preview-subtitle">
          As alterações ao lado já estão refletidas no seu painel atual. Clique em <strong>Salvar personalização</strong> para publicar para todos os usuários.
        </p>
      </div>
      <div class="preview-actions">
        <BaseButton variant="primary" :loading="saving" @click="emit('save')">
          Salvar personalização
        </BaseButton>
      </div>
    </div>

    <div class="preview-Showcase">
      <div class="mini-panel">
        <div class="mini-sidebar">
          <BrandLogo size="sm" :show-tagline="true" />
          <div class="mini-nav">
            <div class="mini-nav-item active">Visão Geral</div>
            <div class="mini-nav-item">Eventos CAPI</div>
            <div class="mini-nav-item">Pixels & Canais</div>
          </div>
        </div>
        <div class="mini-content">
          <div class="mini-kpi-row">
            <div class="mini-kpi">
              <span class="mini-kpi-label">Eventos Processados</span>
              <span class="mini-kpi-val">128.490</span>
            </div>
            <div class="mini-kpi">
              <span class="mini-kpi-label">Match Quality</span>
              <span class="mini-kpi-val accent">9.4 / 10</span>
            </div>
          </div>
          <div class="mini-cta-bar">
            <span class="mini-footer-preview">
              {{ draft.footer_text || `© ${new Date().getFullYear()} ${draft.platform_name || 'Sua Plataforma'}` }}
            </span>
            <span class="mini-btn">Botão Primário</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PlatformBranding } from '#shared/types'
import BrandLogo from '~/components/BrandLogo.vue'

defineProps<{
  draft: PlatformBranding
  saving: boolean
}>()

const emit = defineEmits<{
  save: []
}>()
</script>

<style scoped>
.preview-card {
  background: var(--surface-2);
  border: 1px solid var(--brand-border);
  border-radius: 12px;
  padding: 1.25rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.preview-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}

.preview-subtitle {
  font-size: var(--fs-base);
  color: var(--text-2);
  margin-top: 0.125rem;
}

.mini-panel {
  display: grid;
  grid-template-columns: 200px 1fr;
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  background: var(--surface-1);
}

@media (max-width: 640px) {
  .mini-panel {
    grid-template-columns: 1fr;
  }
}

.mini-sidebar {
  background: var(--surface-2);
  border-right: 1px solid var(--border);
  padding: 0.875rem;
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

.mini-nav {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.mini-nav-item {
  font-size: var(--fs-sm);
  padding: 0.375rem 0.625rem;
  border-radius: 6px;
  color: var(--text-2);
}

.mini-nav-item.active {
  background: var(--brand-soft);
  color: var(--brand);
  font-weight: var(--fw-semibold);
}

.mini-content {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 1rem;
}

.mini-kpi-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 0.75rem;
}

.mini-kpi {
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 0.625rem 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.mini-kpi-label {
  font-size: var(--fs-xs);
  color: var(--text-3);
  text-transform: uppercase;
}

.mini-kpi-val {
  font-family: 'Instrument Sans', sans-serif;
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
}

.mini-kpi-val.accent {
  color: var(--brand);
}

.mini-cta-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  padding-top: 0.625rem;
  border-top: 1px solid var(--border-soft);
}

.mini-footer-preview {
  font-size: var(--fs-sm);
  color: var(--text-3);
}

.mini-btn {
  padding: 0.375rem 0.75rem;
  border-radius: 6px;
  background: var(--brand);
  color: var(--brand-text);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}
</style>

