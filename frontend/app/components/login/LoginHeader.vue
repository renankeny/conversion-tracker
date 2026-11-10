<template>
  <div class="login-header">
    <div class="header-brand">
      <BrandLogo size="md" />
    </div>

    <h2 class="header-title">Acessar Plataforma</h2>
    <p class="header-subtitle">
      Insira suas credenciais para gerenciar seu tráfego e atribuições
    </p>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.login-header {
  margin-bottom: 1.75rem;
  text-align: left;
}

@media (max-width: 767px) {
  .login-header {
    text-align: center;
  }
}

.header-brand {
  display: inline-flex;
  align-items: center;
  margin-bottom: 1.25rem;
}

.header-title {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  letter-spacing: -0.02em;
  margin-bottom: 0.375rem;
}

.header-subtitle {
  font-size: var(--fs-base);
  color: var(--text-3);
  line-height: 1.4;
}
</style>
