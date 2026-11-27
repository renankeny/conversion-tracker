<template>
  <div class="step-ads">
    <div v-if="!metaAccount && showQuickReuse" class="reuse-banner">
      <div class="reuse-info">
        <span class="reuse-title">Usar login Facebook já conectado no painel</span>
        <span class="reuse-sub">Reaproveite o perfil da sua agência sem fazer login OAuth novamente e apenas escolha a conta de anúncios deste cliente.</span>
      </div>
      <div class="reuse-chips">
        <BaseButton
          v-for="src in reusableMetaSources"
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

    <MetaAdsAccountForm
      ref="formRef"
      :account="metaAccount"
      @submit="onFormSubmit"
      @connected="onConnected"
      @refresh="onRefresh"
    />

    <div class="skip-strip">
      <span class="skip-hint">
        Prefere informar o Pixel ID e Access Token manualmente sem conectar via OAuth?
      </span>
      <BaseButton variant="ghost" size="sm" @click="onSkip">
        Usar Pixel manual na próxima etapa
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import MetaAdsAccountForm from '~/components/meta-ads/MetaAdsAccountForm.vue'
import type { MetaAccount, MetaAccountInput } from '#shared/types'
import { useAuthStore } from '~/stores/useAuthStore'
import { useClientOnboardingWizard } from '~/composables/useClientOnboardingWizard'
import { errorMessage } from '~/utils/errorMessage'

const emit = defineEmits<{ done: [] }>()

const metaApi = useMetaAccounts()
const authStore = useAuthStore()
const toast = useToast()

const {
  clientName,
  metaAccount,
  skippedMetaOAuth,
  reusableMetaSources,
} = useClientOnboardingWizard()

const formRef = ref<{ submit: () => void } | null>(null)
const saving = ref(false)
const reusingId = ref<number | null>(null)

const showQuickReuse = computed(() => authStore.isOwner && reusableMetaSources.value.length > 0)

const reuseFromAgency = async (sourceId: number) => {
  reusingId.value = sourceId
  try {
    const created = await metaApi.create({
      name: clientName.value.trim() ? `Meta Ads — ${clientName.value.trim()}` : 'Meta Ads',
      reuse_oauth_from_id: sourceId,
      enabled: true,
    })
    metaAccount.value = created
    skippedMetaOAuth.value = false
    toast.success('Perfil Meta vinculado', 'Agora selecione a Conta de Anúncios e a Página padrão deste cliente.')
  } catch (err: unknown) {
    toast.error('Erro ao reutilizar login Meta', errorMessage(err, 'Tente conectar via OAuth normalmente.'))
  } finally {
    reusingId.value = null
  }
}

const onConnected = (acc: MetaAccount) => {
  metaAccount.value = acc
  skippedMetaOAuth.value = false
}

const onRefresh = async () => {
  if (!metaAccount.value) return
  try {
    metaAccount.value = await metaApi.get(metaAccount.value.id)
  } catch {}
}

const onFormSubmit = async (input: MetaAccountInput) => {
  if (!metaAccount.value) return
  saving.value = true
  try {
    const updated = await metaApi.update(metaAccount.value.id, input)
    metaAccount.value = updated
    skippedMetaOAuth.value = false
    toast.success('Meta Ads configurada', `"${updated.name}" salva com sucesso.`)
    emit('done')
  } catch (err: unknown) {
    toast.error('Erro ao salvar Meta Ads', errorMessage(err, 'Falha ao salvar conta Meta Ads.'))
  } finally {
    saving.value = false
  }
}

const onSkip = () => {
  skippedMetaOAuth.value = true
  emit('done')
}

const submit = () => {
  formRef.value?.submit()
}

const primaryLabel = computed(() => {
  if (metaAccount.value?.connected) return 'Salvar Meta Ads e continuar'
  return 'Conectar Meta Ads'
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
