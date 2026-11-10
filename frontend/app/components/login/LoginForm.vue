<template>
  <form @submit.prevent="handleSubmit" class="login-form">
    <div v-if="errorMessage" class="error-alert">
      <AlertCircle :size="16" class="text-danger flex-shrink-0" />
      <span class="error-text">{{ errorMessage }}</span>
    </div>

    <div class="form-group">
      <label for="login-email" class="form-label">E-mail</label>
      <div class="input-container" :class="{ 'input-focused': focusedField === 'email' }">
        <Mail :size="16" class="input-icon" />
        <input
          id="login-email"
          v-model="credentials.email"
          type="email"
          placeholder="admin@exemplo.com"
          required
          autocomplete="email"
          class="form-input"
          @focus="focusedField = 'email'"
          @blur="focusedField = null"
        />
      </div>
    </div>

    <div class="form-group">
      <label for="login-password" class="form-label">Senha</label>
      <div class="input-container" :class="{ 'input-focused': focusedField === 'password' }">
        <Lock :size="16" class="input-icon" />
        <input
          id="login-password"
          v-model="credentials.password"
          :type="showPassword ? 'text' : 'password'"
          placeholder="••••••••"
          required
          autocomplete="current-password"
          class="form-input pr-10"
          @focus="focusedField = 'password'"
          @blur="focusedField = null"
        />
        <button
          type="button"
          class="password-toggle-btn"
          tabindex="-1"
          @click="showPassword = !showPassword"
        >
          <EyeOff v-if="showPassword" :size="16" />
          <Eye v-else :size="16" />
        </button>
      </div>
    </div>

    <div class="remember-row">
      <BaseCheckbox v-model="credentials.remember_me" size="sm">
        <span class="checkbox-text">
          Lembrar de mim <span class="remember-tag">(30 dias)</span>
        </span>
      </BaseCheckbox>
    </div>

    <button type="submit" class="submit-button" :disabled="loading">
      <span v-if="loading" class="spinner"></span>
      <span v-else class="flex items-center justify-center gap-2">
        <span>Entrar na Plataforma</span>
        <ArrowRight :size="16" />
      </span>
    </button>
  </form>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Mail, Lock, Eye, EyeOff, AlertCircle, ArrowRight } from 'lucide-vue-next'
import { useAuth } from '~/composables/useAuth'

const { login } = useAuth()

const credentials = reactive({
  email: '',
  password: '',
  remember_me: true,
})

const showPassword = ref(false)
const loading = ref(false)
const errorMessage = ref('')
const focusedField = ref<string | null>(null)

onMounted(() => {
  if (import.meta.client) {
    const savedEmail = localStorage.getItem('remember_email')
    if (savedEmail) {
      credentials.email = savedEmail
      credentials.remember_me = true
    }
  }
})

const handleSubmit = async () => {
  errorMessage.value = ''
  const cleanEmail = credentials.email.trim().toLowerCase()
  if (!cleanEmail || !credentials.password) {
    errorMessage.value = 'Por favor, preencha seu e-mail e senha.'
    return
  }

  loading.value = true
  try {
    await login({
      email: cleanEmail,
      password: credentials.password,
      remember_me: credentials.remember_me,
    })
    if (import.meta.client) {
      window.location.href = '/'
      return
    }
    await navigateTo('/', { replace: true })
  } catch (err: any) {
    errorMessage.value = err.message || 'Credenciais inválidas ou erro no servidor.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-form { display: flex; flex-direction: column; gap: 1.125rem; }
.error-alert {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.625rem 0.875rem;
  border-radius: 0.5rem;
  background: var(--danger-soft);
  border: 1px solid var(--danger);
  font-size: var(--fs-base);
  color: var(--danger);
}
.text-danger { color: var(--danger); }
.error-text { font-weight: var(--fw-medium); line-height: 1.3; }
.form-group { display: flex; flex-direction: column; gap: 0.375rem; }
.form-label { font-size: var(--fs-base); font-weight: var(--fw-medium); color: var(--text-2); }
.input-container {
  position: relative;
  display: flex;
  align-items: center;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  transition: all 0.15s ease;
}
.input-focused {
  border-color: var(--brand) !important;
  box-shadow: 0 0 0 2px var(--brand-soft);
  background: var(--surface-1);
}
.input-icon { position: absolute; left: 0.875rem; color: var(--text-3); pointer-events: none; }
.form-input {
  width: 100%;
  height: 42px;
  background: transparent;
  border: none;
  padding: 0 0.875rem 0 2.5rem;
  color: var(--text-1);
  font-size: var(--fs-md);
  outline: none;
}
.form-input::placeholder { color: var(--text-3); font-size: var(--fs-base); }
.password-toggle-btn {
  position: absolute;
  right: 0.75rem;
  background: none;
  border: none;
  color: var(--text-3);
  cursor: pointer;
  padding: 0.25rem;
  display: flex;
  align-items: center;
}
.password-toggle-btn:hover { color: var(--text-1); }
.remember-row { display: flex; align-items: center; justify-content: space-between; }
.checkbox-text { font-size: var(--fs-base); color: var(--text-2); font-weight: var(--fw-medium); }
.remember-tag { color: var(--brand); font-weight: var(--fw-semibold); }
.submit-button {
  width: 100%;
  height: 44px;
  border-radius: 0.5rem;
  background: var(--brand);
  color: var(--brand-text);
  font-weight: var(--fw-semibold);
  font-size: var(--fs-md);
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
  box-shadow: 0 0 16px var(--brand-soft);
  margin-top: 0.5rem;
}
.submit-button:hover:not(:disabled) {
  background: var(--brand-strong);
  box-shadow: 0 0 20px var(--brand-border);
  transform: translateY(-1px);
}
.submit-button:disabled { opacity: 0.6; cursor: not-allowed; }
.spinner {
  width: 1.125rem;
  height: 1.125rem;
  border: 2px solid var(--brand-text);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
