<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <h1 class="font-heading text-3xl font-bold" style="color: var(--text-strong)">
          TrackFind
        </h1>
        <p style="color: var(--text-2)">
          Faça login para acessar o painel
        </p>
      </div>

      <form @submit.prevent="handleLogin" class="login-form">
        <BaseInput
          v-model="credentials.email"
          type="email"
          label="E-mail"
          placeholder="seu@email.com"
          :error="errors.email"
          required
        >
          <template #icon>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M16 12a4 4 0 10-8 0 4 4 0 008 0zm0 0v1.5a2.5 2.5 0 005 0V12a9 9 0 10-9 9m4.5-1.206a8.959 8.959 0 01-4.5 1.207"
              />
            </svg>
          </template>
        </BaseInput>

        <BaseInput
          v-model="credentials.password"
          type="password"
          label="Senha"
          placeholder="••••••••"
          :error="errors.password"
          required
        >
          <template #icon>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
              />
            </svg>
          </template>
        </BaseInput>

        <BaseButton
          type="submit"
          variant="primary"
          size="lg"
          :loading="loading"
          class="w-full"
        >
          Entrar
        </BaseButton>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'default',
  middleware: 'guest',
})

const { login } = useAuth()
const toast = useToast()

const credentials = reactive({
  email: '',
  password: '',
})

const errors = reactive({
  email: '',
  password: '',
})

const loading = ref(false)

const validateForm = () => {
  errors.email = ''
  errors.password = ''

  let isValid = true

  if (!credentials.email) {
    errors.email = 'E-mail é obrigatório'
    isValid = false
  }

  if (!credentials.password) {
    errors.password = 'Senha é obrigatória'
    isValid = false
  }

  return isValid
}

const handleLogin = async () => {
  if (!validateForm()) {
    return
  }

  loading.value = true

  try {
    await login(credentials)
    toast.success('Bem-vindo!', 'Login realizado com sucesso')
    navigateTo('/')
  } catch (error: any) {
    toast.error('Erro no login', error.message || 'Credenciais inválidas')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: var(--surface-1);
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 2.5rem;
  box-shadow: var(--shadow-card);
}

.login-header {
  text-align: center;
  margin-bottom: 2rem;
}

.login-header h1 {
  margin-bottom: 0.5rem;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
</style>
