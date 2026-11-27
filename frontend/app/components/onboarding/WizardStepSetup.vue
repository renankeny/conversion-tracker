<template>
  <div class="step-setup">
    <section class="block">
      <div class="block-head">
        <span class="block-label">1. Empresa do cliente</span>
        <div v-if="existingAccounts.length > 0 && !createdAccount" class="mode-toggle">
          <button
            type="button"
            class="mode-btn"
            :class="{ active: accountMode === 'new' }"
            @click="accountMode = 'new'"
          >
            Nova empresa
          </button>
          <button
            type="button"
            class="mode-btn"
            :class="{ active: accountMode === 'existing' }"
            @click="accountMode = 'existing'"
          >
            Empresa existente
          </button>
        </div>
      </div>

      <div v-if="accountMode === 'new' || createdAccount" class="field-wrap">
        <BaseInput
          v-model="clientName"
          label="Nome do cliente / empresa"
          placeholder="Ex: Clínica Odonto Prime"
          :disabled="Boolean(createdAccount)"
          :error="nameError"
          required
        />
        <span v-if="slugPreview && !createdAccount" class="slug-hint">
          Identificador automático: <code>{{ slugPreview }}</code>
        </span>
      </div>

      <BaseDropdown
        v-else
        v-model="selectedExistingAccountId"
        label="Selecione a empresa"
        placeholder="Escolha uma empresa já cadastrada..."
        :options="existingAccountOptions"
        :error="nameError"
      />
    </section>

    <section class="block">
      <span class="block-label">2. Onde o cliente anuncia?</span>
      <div class="choice-grid">
        <button
          v-for="opt in platformOptions"
          :key="opt.value"
          type="button"
          class="choice-card"
          :class="{ selected: platform === opt.value }"
          @click="platform = opt.value"
        >
          <div class="choice-top">
            <div class="choice-icons">
              <MetaAdsIcon v-if="opt.value === 'meta' || opt.value === 'both'" :size="18" />
              <GoogleAdsIcon v-if="opt.value === 'google' || opt.value === 'both'" :size="18" />
            </div>
            <span class="choice-check" :class="{ active: platform === opt.value }" />
          </div>
          <span class="choice-title">{{ opt.title }}</span>
          <span class="choice-desc">{{ opt.desc }}</span>
        </button>
      </div>
    </section>

    <section class="block">
      <span class="block-label">3. Qual o tipo de campanha?</span>
      <div class="choice-grid">
        <button
          v-for="opt in campaignOptions"
          :key="opt.value"
          type="button"
          class="choice-card"
          :class="{ selected: campaignType === opt.value }"
          @click="campaignType = opt.value"
        >
          <div class="choice-top">
            <component :is="opt.icon" :size="18" class="choice-icon" />
            <span class="choice-check" :class="{ active: campaignType === opt.value }" />
          </div>
          <span class="choice-title">{{ opt.title }}</span>
          <span class="choice-desc">{{ opt.desc }}</span>
        </button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Globe, Layers, MessageCircle } from 'lucide-vue-next'
import { MetaAdsIcon, GoogleAdsIcon } from '~/utils/adIcons'
import { useTenantStore } from '~/stores/useTenantStore'
import { errorMessage } from '~/utils/errorMessage'
import {
  slugifyClientName,
  useClientOnboardingWizard,
  type OnboardingCampaignType,
  type OnboardingPlatform,
} from '~/composables/useClientOnboardingWizard'

const emit = defineEmits<{ done: [] }>()

const accountsApi = useAccounts()
const tenantStore = useTenantStore()
const toast = useToast()

const {
  accountMode,
  selectedExistingAccountId,
  clientName,
  platform,
  campaignType,
  createdAccount,
} = useClientOnboardingWizard()

const nameError = ref('')
const saving = ref(false)

const existingAccounts = computed(() => tenantStore.accounts || [])
const existingAccountOptions = computed(() =>
  existingAccounts.value.map((a) => ({ value: a.id, label: a.name })),
)
const slugPreview = computed(() => slugifyClientName(clientName.value))

const platformOptions: Array<{ value: OnboardingPlatform; title: string; desc: string }> = [
  { value: 'meta', title: 'Meta Ads', desc: 'Facebook e Instagram Ads (CAPI e CTWA)' },
  { value: 'google', title: 'Google Ads', desc: 'Pesquisa, YouTube, PMax e conversões offline' },
  { value: 'both', title: 'Ambos', desc: 'Conecta primeiro Meta Ads e depois Google Ads' },
]

const campaignOptions: Array<{ value: OnboardingCampaignType; title: string; desc: string; icon: unknown }> = [
  {
    value: 'messages',
    title: 'Mensagens (WhatsApp)',
    desc: 'Sem domínio nem script de site. Pixel + WhatsApp + gatilhos.',
    icon: MessageCircle,
  },
  {
    value: 'site',
    title: 'Site / Landing Page',
    desc: 'Domínio, Pixel, Tracker e script pronto para colar no site.',
    icon: Globe,
  },
  {
    value: 'both',
    title: 'Site + Mensagens',
    desc: 'Funil completo com script no site, WhatsApp e gatilhos.',
    icon: Layers,
  },
]

const submit = async () => {
  nameError.value = ''
  if (createdAccount.value) {
    if (String(tenantStore.activeAccountId) !== String(createdAccount.value.id)) {
      tenantStore.switchAccount(createdAccount.value.id)
    }
    emit('done')
    return
  }

  if (accountMode.value === 'existing') {
    const found = existingAccounts.value.find((a) => a.id === Number(selectedExistingAccountId.value))
    if (!found) {
      nameError.value = 'Selecione uma empresa para continuar'
      return
    }
    createdAccount.value = found
    clientName.value = found.name
    tenantStore.switchAccount(found.id)
    emit('done')
    return
  }

  const trimmed = clientName.value.trim()
  if (trimmed.length < 2) {
    nameError.value = 'Informe o nome do cliente (mínimo 2 caracteres)'
    return
  }

  saving.value = true
  try {
    const created = await accountsApi.create({
      name: trimmed,
      slug: slugifyClientName(trimmed) || undefined,
      enabled: true,
    })
    createdAccount.value = created
    tenantStore.switchAccount(created.id)
    toast.success('Empresa criada', `"${created.name}" selecionada para configuração.`)
    emit('done')
  } catch (err: unknown) {
    nameError.value = errorMessage(err, 'Não foi possível criar a empresa')
  } finally {
    saving.value = false
  }
}

defineExpose({ submit, saving })
</script>

<style scoped>
.step-setup { display: flex; flex-direction: column; gap: 1.35rem; }
.block { display: flex; flex-direction: column; gap: 0.65rem; }
.block-head { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.block-label { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); }
.mode-toggle { display: inline-flex; padding: 0.18rem; border-radius: 8px; background: var(--surface-deep); border: 1px solid var(--border-soft); gap: 0.18rem; }
.mode-btn { padding: 0.3rem 0.65rem; border-radius: 6px; border: none; background: transparent; color: var(--text-3); font-size: var(--fs-xs); font-weight: var(--fw-medium); cursor: pointer; transition: all 0.15s; }
.mode-btn.active { background: var(--brand-soft); color: var(--brand); font-weight: var(--fw-semibold); }
.field-wrap { display: flex; flex-direction: column; gap: 0.35rem; }
.slug-hint { font-size: var(--fs-xs); color: var(--text-3); }
.slug-hint code { font-family: var(--font-mono); color: var(--text-2); background: var(--surface-soft); padding: 0.08rem 0.35rem; border-radius: 4px; }
.choice-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; }
.choice-card {
  display: flex; flex-direction: column; align-items: flex-start; gap: 0.45rem;
  padding: 0.9rem; border-radius: 11px; text-align: left; cursor: pointer;
  background: var(--surface-deep); border: 1px solid var(--border-soft); transition: all 0.15s ease;
}
.choice-card:hover { border-color: var(--brand-border); background: var(--surface-soft); }
.choice-card.selected { border-color: var(--brand); background: var(--brand-soft); }
.choice-top { display: flex; align-items: center; justify-content: space-between; width: 100%; }
.choice-icons { display: inline-flex; align-items: center; gap: 0.4rem; color: var(--text-1); }
.choice-icon { color: var(--brand); }
.choice-check { width: 14px; height: 14px; border-radius: 50%; border: 1.5px solid var(--border); flex-shrink: 0; transition: all 0.15s; }
.choice-check.active { border-color: var(--brand); background: var(--brand); box-shadow: inset 0 0 0 3px var(--surface-2); }
.choice-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); }
.choice-desc { font-size: var(--fs-xs); color: var(--text-2); line-height: 1.4; }
@media (max-width: 640px) { .choice-grid { grid-template-columns: 1fr; } }
</style>
