<template>
  <form class="step-form" @submit.prevent="onNext">
    <div class="step-intro">
      <h2 class="step-title">Criar Conta Super Admin</h2>
      <p class="step-desc">
        Este será o usuário mestre da plataforma, com controle total sobre empresas, usuários e personalização Whitelabel.
      </p>
    </div>

    <div class="purchase-email-notice">
      <AlertCircle :size="18" class="notice-icon" />
      <div class="notice-content">
        <strong class="notice-title">Atenção ao e-mail de cadastro</strong>
        <p class="notice-text">
          É muito importante utilizar <strong>exatamente o mesmo e-mail usado na compra do sistema</strong> para validar sua licença e liberar seu acesso.
        </p>
      </div>
    </div>

    <div class="fields-stack">
      <BaseInput
        v-model="model.admin_name"
        label="Seu Nome Completo"
        placeholder="Ex: João Silva"
        required
      />

      <BaseInput
        v-model="model.admin_email"
        type="email"
        label="E-mail de Acesso (mesmo utilizado na compra)"
        placeholder="seuemail@compradosistema.com.br"
        hint="Utilize obrigatoriamente o mesmo endereço de e-mail da compra do sistema."
        required
      />

      <BaseInput
        v-model="model.admin_phone"
        label="WhatsApp / Telefone (Opcional)"
        placeholder="11999999999"
      />

      <div class="password-grid">
        <BaseInput
          v-model="model.admin_password"
          type="password"
          label="Senha (mín. 6 caracteres)"
          placeholder="••••••••"
          required
        />
        <BaseInput
          v-model="confirmPassword"
          type="password"
          label="Confirmar Senha"
          placeholder="••••••••"
          required
        />
      </div>
    </div>

    <p v-if="errorMsg" class="error-banner">{{ errorMsg }}</p>

    <div class="step-actions">
      <BaseButton type="submit" variant="primary" class="w-full" :loading="verifyingEmail">
        Continuar para Operação →
      </BaseButton>
    </div>
  </form>
</template>

<script setup lang="ts">
import { AlertCircle } from 'lucide-vue-next'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  model: {
    admin_name: string
    admin_email: string
    admin_phone: string
    admin_password: string
  }
}>()

const emit = defineEmits<{
  next: []
}>()

const { verifySetupEmail } = useSetup()
const confirmPassword = ref(props.model.admin_password || '')
const errorMsg = ref('')
const verifyingEmail = ref(false)

const onNext = async () => {
  errorMsg.value = ''
  if (!props.model.admin_name.trim()) {
    errorMsg.value = 'Informe seu nome completo.'
    return
  }
  if (!props.model.admin_email.trim() || !props.model.admin_email.includes('@')) {
    errorMsg.value = 'Informe um endereço de e-mail válido (o mesmo utilizado na compra).'
    return
  }
  if (props.model.admin_password.length < 6) {
    errorMsg.value = 'A senha deve ter pelo menos 6 caracteres.'
    return
  }
  if (props.model.admin_password !== confirmPassword.value) {
    errorMsg.value = 'As senhas digitadas não coincidem.'
    return
  }

  verifyingEmail.value = true
  try {
    await verifySetupEmail(props.model.admin_email.trim())
    emit('next')
  } catch (err: unknown) {
    errorMsg.value = errorMessage(err, 'Este e-mail não possui uma compra ativa do sistema.')
  } finally {
    verifyingEmail.value = false
  }
}
</script>

<style scoped>
.step-form { display: flex; flex-direction: column; gap: 1.125rem; }
.step-intro { display: flex; flex-direction: column; gap: 0.375rem; }
.step-title { font-family: var(--font-heading); font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.step-desc { font-size: var(--fs-md); color: var(--text-2); line-height: 1.5; }
.purchase-email-notice {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  border-radius: 10px;
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
}
.notice-icon {
  color: var(--brand);
  flex-shrink: 0;
  margin-top: 0.125rem;
}
.notice-content {
  display: flex;
  flex-direction: column;
  gap: 0.1875rem;
}
.notice-title {
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}
.notice-text {
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.45;
}
.notice-text strong {
  color: var(--text-strong);
  font-weight: var(--fw-semibold);
}
.fields-stack { display: flex; flex-direction: column; gap: 0.875rem; }
.password-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
@media (max-width: 520px) { .password-grid { grid-template-columns: 1fr; } }
.error-banner { font-size: var(--fs-base); color: var(--danger); background: var(--surface-soft-2); border: 1px solid var(--danger); padding: 0.625rem 0.875rem; border-radius: 8px; }
.step-actions { display: flex; justify-content: flex-end; padding-top: 0.25rem; }
</style>
