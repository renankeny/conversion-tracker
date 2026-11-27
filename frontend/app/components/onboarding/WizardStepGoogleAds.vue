<template>
  <div class="step-ads">
    <div v-if="!googleAccount && showQuickReuse" class="reuse-banner">
      <div class="reuse-info">
        <span class="reuse-title">Usar login Google já conectado no painel</span>
        <span class="reuse-sub">Reaproveite sua MCC/conta Google da agência sem novo OAuth e selecione o Customer ID deste cliente.</span>
      </div>
      <div class="reuse-chips">
        <BaseButton
          v-for="src in reusableGoogleSources"
          :key="src.id"
          variant="outline"
          size="sm"
          :loading="reusingId === src.id"
          @click="reuseFromAgency(src.id)"
        >
          {{ src.label }}
        </BaseButton>
      </div>
    </div>

    <GoogleAdsAccountForm
      ref="formRef"
      :account="googleAccount"
      @submit="onFormSubmit"
      @connected="onConnected"
      @refresh="onRefresh"
    />

    <div class="skip-strip">
      <span class="skip-hint">
        Ainda não tem acesso à conta Google Ads deste cliente?
      </span>
      <BaseButton variant="ghost" size="sm" @click="onSkip">
        Conectar Google Ads depois
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import GoogleAdsAccountForm from '~/components/google-ads/GoogleAdsAccountForm.vue'
import type { GoogleAdsAccount, GoogleAdsAccountInput } from '#shared/types'
import { useAuthStore } from '~/stores/useAuthStore'
import { useClientOnboardingWizard } from '~/composables/useClientOnboardingWizard'
import { errorMessage } from '~/utils/errorMessage'

const emit = defineEmits<{ done: [] }>()

const gadsApi = useGoogleAdsAccounts()
const authStore = useAuthStore()
const toast = useToast()

const {
  clientName,
  googleAccount,
  skippedGoogleOAuth,
  reusableGoogleSources,
} = useClientOnboardingWizard()

const formRef = ref<{ submit: () => void } | null>(null)
const saving = ref(false)
const reusingId = ref<number | null>(null)

const showQuickReuse = computed(() => authStore.isOwner && reusableGoogleSources.value.length > 0)

const reuseFromAgency = async (sourceId: number) => {
  reusingId.value = sourceId
  try {
    const created = await gadsApi.create({
      name: clientName.value.trim() ? `Google Ads — ${clientName.value.trim()}` : 'Google Ads',
      reuse_oauth_from_id: sourceId,
      enabled: true,
    })
    googleAccount.value = created
    skippedGoogleOAuth.value = false
    toast.success('Login Google vinculado', 'Agora selecione o Customer ID e as Ações de Conversão deste cliente.')
  } catch (err: unknown) {
    toast.error('Erro ao reutilizar login Google', errorMessage(err, 'Tente conectar via OAuth normalmente.'))
  } finally {
    reusingId.value = null
  }
}

const onConnected = (acc: GoogleAdsAccount) => {
  googleAccount.value = acc
  skippedGoogleOAuth.value = false
}

const onRefresh = async () => {
  if (!googleAccount.value) return
  try {
    googleAccount.value = await gadsApi.get(googleAccount.value.id)
  } catch {}
}

const onFormSubmit = async (input: GoogleAdsAccountInput) => {
  if (!googleAccount.value) return
  saving.value = true
  try {
    const updated = await gadsApi.update(googleAccount.value.id, input)
    googleAccount.value = updated
    skippedGoogleOAuth.value = false
    toast.success('Google Ads conectado', `"${updated.name}" salvo com sucesso.`)
    emit('done')
  } catch (err: unknown) {
    toast.error('Erro ao salvar Google Ads', errorMessage(err, 'Falha ao salvar conta Google Ads.'))
  } finally {
    saving.value = false
  }
}

const onSkip = () => {
  skippedGoogleOAuth.value = true
  emit('done')
}

const submit = () => {
  formRef.value?.submit()
}

const primaryLabel = computed(() => {
  if (googleAccount.value?.connected || googleAccount.value?.oauth_connected) {
    return 'Salvar Google Ads e continuar'
  }
  return 'Conectar Google Ads'
})

defineExpose({ submit, saving, primaryLabel })
</script>

<style scoped>
.step-ads { display: flex; flex-direction: column; gap: 1.15rem; }
.reuse-banner {
  display: flex; flex-direction: column; gap: 0.65rem; padding: 0.85rem 1rem;
  border-radius: 10px; background: var(--brand-soft); border: 1px solid var(--brand-border);
}
.reuse-info { display: flex; flex-direction: column; gap: 0.15rem; }
.reuse-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); }
.reuse-sub { font-size: var(--fs-xs); color: var(--text-2); line-height: 1.4; }
.reuse-chips { display: flex; flex-wrap: wrap; gap: 0.5rem; }
.skip-strip {
  display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap;
  padding-top: 0.75rem; border-top: 1px solid var(--border-soft);
}
.skip-hint { font-size: var(--fs-xs); color: var(--text-3); }
</style>
