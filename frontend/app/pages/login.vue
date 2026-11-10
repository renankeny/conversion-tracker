<template>
  <!-- Loading state while checking active session -->
  <div v-if="isCheckingSession" class="session-loading-screen">
    <div class="flex flex-col items-center gap-3">
      <div class="spinner-brand"></div>
      <p class="loading-text">Verificando sessão segura...</p>
    </div>
  </div>

  <!-- Main Login Screen: Split Desktop/Tablet, Single Column Mobile -->
  <div v-else class="login-page-wrapper">
    <!-- Left Column: Hero Panel (Desktop & Tablet: hidden on mobile) -->
    <div class="hero-column">
      <LoginHero />
    </div>

    <!-- Right Column: Login Form Section (Mobile: full width, Desktop/Tablet: right panel) -->
    <div class="form-column">
      <div class="form-card-wrapper">
        <LoginHeader />
        <LoginForm />
        <LoginFooter class="mt-6" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuth } from '~/composables/useAuth'
import LoginHero from '~/components/login/LoginHero.vue'
import LoginHeader from '~/components/login/LoginHeader.vue'
import LoginForm from '~/components/login/LoginForm.vue'
import LoginFooter from '~/components/login/LoginFooter.vue'

definePageMeta({
  layout: false,
})

const { formatTitle } = useBranding()
useHead({
  title: computed(() => formatTitle('Login')),
})

const { token } = useAuth()
const { checkSetupStatus } = useSetup()
const isCheckingSession = ref(true)

onMounted(async () => {
  // Se ainda não existe nenhum Super Admin no banco, redireciona para o Wizard de 1º acesso
  const mustSetup = await checkSetupStatus()
  if (mustSetup) {
    await navigateTo('/setup')
    return
  }

  // Se o usuário já estiver logado, redireciona imediatamente
  if (token.value) {
    await navigateTo('/')
    return
  }

  isCheckingSession.value = false
})
</script>

<style scoped>
.session-loading-screen {
  height: 100vh;
  width: 100vw;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-1);
}

.spinner-brand {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border);
  border-top-color: var(--brand);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-text {
  font-size: var(--fs-base);
  color: var(--text-3);
  font-weight: var(--fw-medium);
}

.login-page-wrapper {
  min-height: 100vh;
  width: 100vw;
  display: flex;
  background: var(--surface-1);
  color: var(--text-1);
  overflow-x: hidden;
}

/* Left Hero Column: hidden on mobile, visible on tablet (md) and desktop (lg/xl) */
.hero-column {
  display: none;
}

@media (min-width: 768px) {
  .hero-column {
    display: block;
    width: 50%;
    height: 100vh;
    overflow: hidden;
  }
}

@media (min-width: 1024px) {
  .hero-column {
    width: 58%;
  }
}

/* Right Form Column */
.form-column {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem 1.5rem;
  min-height: 100vh;
  background: var(--surface-1);
  position: relative;
}

@media (min-width: 768px) {
  .form-column {
    width: 50%;
    padding: 3rem 2.5rem;
    overflow-y: auto;
  }
}

@media (min-width: 1024px) {
  .form-column {
    width: 42%;
    padding: 4rem 3.5rem;
  }
}

.form-card-wrapper {
  width: 100%;
  max-width: 400px;
  margin: auto 0;
  display: flex;
  flex-direction: column;
}
</style>
