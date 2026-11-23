<script setup lang="ts">
import { ShieldCheck, Lock, CheckCircle2, KeyRound } from 'lucide-vue-next'

defineProps<{
  apiUrl: string
}>()

const emit = defineEmits<{
  'open-keys': []
}>()
</script>

<template>
  <div class="auth-section">
    <div class="auth-card">
      <div class="auth-top">
        <div>
          <h2 class="section-title">Uma Única Chave para API e MCP</h2>
          <p class="section-desc">
            Você pode visualizar sua chave quando quiser e editar quais módulos ela pode acessar (inclusive em modo somente leitura).
          </p>
        </div>
        <BaseButton variant="primary" size="sm" @click="emit('open-keys')">
          <KeyRound class="icon-sm" />
          Ver e Editar Chaves
        </BaseButton>
      </div>

      <div class="header-box">
        <div class="header-row">
          <span class="header-tag">Padrão Oficial</span>
          <code class="header-code">X-API-Key: trk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx</code>
        </div>
        <p class="compat-note">
          Compatibilidade automática: integrações que enviam
          <code>Authorization: Bearer trk_...</code> também continuam aceitas em 100% das rotas.
        </p>
      </div>

      <div class="base-url-row">
        <span class="base-label">URL Base desta instalação:</span>
        <code class="base-code">{{ apiUrl }}</code>
      </div>
    </div>

    <div class="security-grid">
      <div class="sec-card">
        <div class="sec-header">
          <CheckCircle2 class="sec-icon text-brand" />
          <h3 class="sec-title">Isolamento Estrito por Empresa</h3>
        </div>
        <p class="sec-desc">
          Cada chave <code>trk_...</code> nasce vinculada exclusivamente a uma empresa (<code>account_id</code>).
          Ela acessa apenas os pixels, conversas de WhatsApp, funil CRM e métricas daquela empresa, sem possibilidade
          de visualizar ou alterar outras empresas do seu painel.
        </p>
      </div>

      <div class="sec-card">
        <div class="sec-header">
          <Lock class="sec-icon text-danger" />
          <h3 class="sec-title">Rotas Bloqueadas para API Keys (403)</h3>
        </div>
        <p class="sec-desc">
          Por segurança, ações administrativas sensíveis exigem login humano no painel: criar ou excluir empresas,
          gerenciar usuários/senhas, criar ou revogar API Keys, alterar personalização Whitelabel e rodar limpeza
          destrutiva de banco.
        </p>
      </div>
    </div>

    <div class="status-strip">
      <ShieldCheck class="icon-sm text-brand" />
      <span>
        <strong>Códigos HTTP:</strong> <code>200 / 201 / 202</code> sucesso ·
        <code>400</code> parâmetro inválido · <code>401</code> chave ausente ou revogada ·
        <code>403</code> rota exclusiva humana ou empresa diferente · <code>429</code> limite de segurança por minuto.
      </span>
    </div>
  </div>
</template>

<style scoped>
.auth-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.auth-card,
.sec-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.35rem;
}

.auth-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.section-title {
  font-family: var(--font-sans);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.section-desc {
  font-size: var(--fs-md);
  color: var(--text-2);
}

.icon-sm {
  width: 0.95rem;
  height: 0.95rem;
  flex-shrink: 0;
}

.header-box {
  background: var(--surface-deep);
  border: 1px solid var(--brand-border);
  border-radius: 10px;
  padding: 0.9rem 1rem;
  margin-bottom: 0.875rem;
}

.header-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.65rem;
  margin-bottom: 0.45rem;
}

.header-tag {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--brand-soft);
  color: var(--brand);
  padding: 0.15rem 0.5rem;
  border-radius: 6px;
}

.header-code {
  font-family: var(--font-mono);
  font-size: var(--fs-lg);
  color: var(--brand);
  word-break: break-all;
}

.compat-note {
  font-size: var(--fs-sm);
  color: var(--text-3);
}

.compat-note code,
.sec-desc code,
.status-strip code {
  font-family: var(--font-mono);
  color: var(--text-1);
  background: var(--surface-soft);
  padding: 0.08rem 0.35rem;
  border-radius: 4px;
}

.base-url-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  font-size: var(--fs-sm);
  color: var(--text-2);
}

.base-code {
  font-family: var(--font-mono);
  color: var(--text-1);
  background: var(--surface-deep);
  padding: 0.25rem 0.6rem;
  border-radius: 6px;
  border: 1px solid var(--border);
}

.security-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.sec-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.45rem;
}

.sec-icon {
  width: 1.05rem;
  height: 1.05rem;
}

.text-brand {
  color: var(--brand);
}

.text-danger {
  color: var(--danger);
}

.sec-title {
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}

.sec-desc {
  font-size: var(--fs-md);
  color: var(--text-2);
  line-height: 1.55;
}

.status-strip {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.8rem 1rem;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 10px;
  font-size: var(--fs-sm);
  color: var(--text-2);
}

@media (max-width: 768px) {
  .security-grid {
    grid-template-columns: 1fr;
  }
}
</style>
