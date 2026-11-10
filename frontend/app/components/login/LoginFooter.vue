<template>
  <div class="login-footer">
    <div class="security-badge">
      <ShieldCheck :size="14" class="text-brand" />
      <span>Ambiente Criptografado & Sessão Segura</span>
    </div>
    <p class="copyright-text">
      {{ footerText }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { ShieldCheck } from 'lucide-vue-next'

const { footerText } = useBranding()
</script>

<style scoped>
.login-footer {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.625rem;
  padding-top: 1.25rem;
  border-top: 1px solid var(--border-soft);
  text-align: center;
}

.security-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: var(--fs-sm);
  color: var(--text-3);
  font-weight: var(--fw-medium);
}

.text-brand {
  color: var(--brand);
}

.copyright-text {
  font-size: var(--fs-xs);
  color: var(--text-3);
  opacity: 0.8;
}
</style>
