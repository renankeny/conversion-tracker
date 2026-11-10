<template>
  <form class="step-form" @submit.prevent="onNext">
    <div class="step-intro">
      <h2 class="step-title">Sua Operação Principal</h2>
      <p class="step-desc">
        A plataforma é multi-empresa. Crie agora a sua conta principal para organizar seus primeiros Pixels, Funis CRM e Webhooks.
      </p>
    </div>

    <div class="fields-stack">
      <BaseInput
        v-model="model.account_name"
        label="Nome da Empresa ou Operação"
        placeholder="Ex: Minha Agência / Operação Principal"
        required
      />

      <div class="info-box">
        <span class="info-badge">Multi-Tenant Nativo</span>
        <p class="info-text">
          Você poderá adicionar quantas empresas ou clientes quiser depois em <strong>Configurações → Empresas</strong>, cada uma com seus próprios pixels e equipe isolada.
        </p>
      </div>
    </div>

    <p v-if="errorMsg" class="error-banner">{{ errorMsg }}</p>

    <div class="step-actions">
      <BaseButton type="button" variant="ghost" @click="emit('back')">
        ← Voltar
      </BaseButton>
      <BaseButton type="submit" variant="primary">
        Personalizar Marca →
      </BaseButton>
    </div>
  </form>
</template>

<script setup lang="ts">
const props = defineProps<{
  model: {
    account_name: string
  }
}>()

const emit = defineEmits<{
  back: []
  next: []
}>()

const errorMsg = ref('')

const onNext = () => {
  errorMsg.value = ''
  if (!props.model.account_name.trim()) {
    errorMsg.value = 'Informe o nome da sua empresa ou operação principal.'
    return
  }
  emit('next')
}
</script>

<style scoped>
.step-form { display: flex; flex-direction: column; gap: 1.25rem; }
.step-intro { display: flex; flex-direction: column; gap: 0.375rem; }
.step-title { font-family: 'Instrument Sans', sans-serif; font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); }
.step-desc { font-size: var(--fs-md); color: var(--text-2); line-height: 1.5; }
.fields-stack { display: flex; flex-direction: column; gap: 1rem; }
.info-box { background: var(--surface-soft-2); border: 1px solid var(--border-soft); border-radius: 10px; padding: 0.875rem 1rem; display: flex; flex-direction: column; gap: 0.375rem; }
.info-badge { align-self: flex-start; font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--brand); background: var(--brand-soft); padding: 0.125rem 0.5rem; border-radius: 999px; }
.info-text { font-size: var(--fs-base); color: var(--text-2); line-height: 1.45; }
.error-banner { font-size: var(--fs-base); color: var(--danger); background: var(--surface-soft-2); border: 1px solid var(--danger); padding: 0.625rem 0.875rem; border-radius: 8px; }
.step-actions { display: flex; justify-content: space-between; align-items: center; gap: 0.75rem; padding-top: 0.25rem; }
</style>

