<template>
  <BaseModal
    v-model="isOpen"
    :title="modalTitle"
    :description="modalDescription"
    size="lg"
    :close-on-overlay="allowOverlayClose"
  >
    <div class="wizard-shell">
      <ol class="stepper" aria-label="Etapas do cadastro">
        <li
          v-for="(step, idx) in steps"
          :key="step.key"
          class="step-item"
          :class="{ done: idx < stepIndex, active: idx === stepIndex }"
        >
          <span class="step-badge">
            <Check v-if="idx < stepIndex" :size="12" />
            <span v-else>{{ idx + 1 }}</span>
          </span>
          <span class="step-label">{{ step.label }}</span>
        </li>
      </ol>

      <div class="step-viewport">
        <WizardStepSetup
          v-if="currentStep.key === 'setup'"
          ref="activeStepRef"
          @done="nextStep"
        />
        <WizardStepMetaAds
          v-else-if="currentStep.key === 'meta'"
          ref="activeStepRef"
          @done="nextStep"
        />
        <WizardStepGoogleAds
          v-else-if="currentStep.key === 'google'"
          ref="activeStepRef"
          @done="nextStep"
        />
        <WizardStepPixel
          v-else-if="currentStep.key === 'pixel'"
          ref="activeStepRef"
          @done="nextStep"
        />
        <WizardStepWhatsApp
          v-else-if="currentStep.key === 'whatsapp'"
          ref="activeStepRef"
          @done="nextStep"
        />
        <WizardStepDone
          v-else
          ref="activeStepRef"
          @finish="onFinish"
        />
      </div>
    </div>

    <template #footer>
      <div class="wizard-footer">
        <BaseButton
          v-if="stepIndex === 0"
          variant="ghost"
          @click="closeWizard"
        >
          Cancelar
        </BaseButton>
        <BaseButton
          v-else-if="currentStep.key !== 'done'"
          variant="ghost"
          :disabled="isStepSaving"
          @click="prevStep"
        >
          Voltar
        </BaseButton>
        <BaseButton
          v-else
          variant="ghost"
          @click="closeWizard"
        >
          Fechar
        </BaseButton>

        <BaseButton
          variant="primary"
          :loading="isStepSaving"
          @click="onPrimaryClick"
        >
          {{ primaryButtonLabel }}
        </BaseButton>
      </div>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check } from 'lucide-vue-next'
import WizardStepSetup from '~/components/onboarding/WizardStepSetup.vue'
import WizardStepMetaAds from '~/components/onboarding/WizardStepMetaAds.vue'
import WizardStepGoogleAds from '~/components/onboarding/WizardStepGoogleAds.vue'
import WizardStepPixel from '~/components/onboarding/WizardStepPixel.vue'
import WizardStepWhatsApp from '~/components/onboarding/WizardStepWhatsApp.vue'
import WizardStepDone from '~/components/onboarding/WizardStepDone.vue'
import { useClientOnboardingWizard } from '~/composables/useClientOnboardingWizard'

interface StepExpose {
  submit: () => void | Promise<void>
  saving?: boolean
  primaryLabel?: string
}

const {
  isOpen,
  stepIndex,
  steps,
  currentStep,
  clientName,
  createdAccount,
  isMessagesOnly,
  closeWizard,
  nextStep,
  prevStep,
} = useClientOnboardingWizard()

const activeStepRef = ref<StepExpose | null>(null)

const allowOverlayClose = computed(
  () => currentStep.value.key === 'setup' || currentStep.value.key === 'done',
)

const isStepSaving = computed(() => Boolean(activeStepRef.value?.saving))

const activeClientLabel = computed(() => createdAccount.value?.name || clientName.value.trim())

const modalTitle = computed(() => {
  switch (currentStep.value.key) {
    case 'setup': return 'Novo cliente'
    case 'meta': return `Meta Ads · ${activeClientLabel.value || 'Novo cliente'}`
    case 'google': return `Google Ads · ${activeClientLabel.value || 'Novo cliente'}`
    case 'pixel': return `Configurar Pixel · ${activeClientLabel.value || 'Novo cliente'}`
    case 'whatsapp': return `WhatsApp & Gatilhos · ${activeClientLabel.value || 'Novo cliente'}`
    case 'done': return 'Configuração concluída'
    default: return 'Novo cliente'
  }
})

const modalDescription = computed(() => {
  switch (currentStep.value.key) {
    case 'setup': return 'Configure um novo cliente passo a passo em poucos minutos.'
    case 'meta': return 'Conecte a conta Meta Ads e selecione a Conta de Anúncios e a Página padrão.'
    case 'google': return 'Conecte a conta Google Ads, selecione o Customer ID e as Ações de Conversão.'
    case 'pixel': return isMessagesOnly.value
      ? 'Escolha o Pixel da conta ou informe manualmente para receber as conversões do WhatsApp.'
      : 'Informe o domínio do site, configure o Pixel e ajuste o rastreamento automático.'
    case 'whatsapp': return 'Configure a conexão do WhatsApp, gatilhos de conversão e escaneie o QR Code.'
    case 'done': return 'Revise o resumo abaixo e copie os links ou script de instalação.'
    default: return ''
  }
})

const primaryButtonLabel = computed(() => {
  if (activeStepRef.value?.primaryLabel) return activeStepRef.value.primaryLabel
  switch (currentStep.value.key) {
    case 'setup': return 'Continuar'
    case 'done': return 'Ir para o Dashboard'
    default: return 'Salvar e continuar'
  }
})

const onPrimaryClick = () => {
  activeStepRef.value?.submit()
}

const onFinish = () => {
  closeWizard()
}
</script>

<style scoped>
.wizard-shell { display: flex; flex-direction: column; gap: 1.25rem; }
.stepper {
  display: flex; align-items: center; gap: 0.4rem; list-style: none; margin: 0; padding: 0 0 0.9rem;
  border-bottom: 1px solid var(--border-soft); overflow-x: auto; white-space: nowrap;
}
.step-item {
  display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.3rem 0.65rem;
  border-radius: 999px; background: var(--surface-soft); border: 1px solid transparent;
  color: var(--text-3); font-size: var(--fs-xs); font-weight: var(--fw-medium); flex-shrink: 0;
}
.step-item.active { background: var(--brand-soft); border-color: var(--brand-border); color: var(--brand); font-weight: var(--fw-semibold); }
.step-item.done { color: var(--success); background: var(--success-soft); }
.step-badge {
  width: 18px; height: 18px; border-radius: 50%; display: inline-flex; align-items: center; justify-content: center;
  background: var(--surface-2); font-size: var(--fs-2xs); font-weight: var(--fw-bold); color: inherit;
}
.step-item.active .step-badge { background: var(--brand); color: var(--brand-text); }
.step-item.done .step-badge { background: var(--success); color: var(--surface-deep); }
.step-viewport { min-height: 220px; }
.wizard-footer { display: flex; align-items: center; justify-content: space-between; width: 100%; gap: 0.75rem; }
</style>
