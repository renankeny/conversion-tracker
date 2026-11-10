<template>
  <section class="assets-section">
    <header class="section-header">
      <div>
        <h4>2. Conta de Anúncios & Página Padrão</h4>
        <p class="section-hint">
          Selecione a Conta de Anúncios (`act_...`) para Spend/Criativos e a Página do Facebook vinculada aos anúncios Click-to-WhatsApp.
        </p>
      </div>
      <BaseButton
        v-if="oauthConnected"
        type="button"
        variant="ghost"
        size="sm"
        :loading="loadingAssets"
        @click="$emit('refresh-assets')"
      >
        Atualizar lista
      </BaseButton>
    </header>

    <BaseInput
      v-model="form.name"
      label="Nome interno da conexão"
      placeholder="Ex: BM Principal — Meta Ads"
      :error="nameError"
      required
    />

    <div class="field-grid">
      <BaseAdAccountSelect
        :model-value="form.ad_account_id"
        label="Conta de Anúncios (Ad Account)"
        :options="adAccountOptions"
        placeholder="Selecione uma Conta de Anúncios..."
        :disabled="!oauthConnected"
        :loading="loadingAssets"
        @update:model-value="onSelectAdAccount"
      />

      <BaseAdAccountSelect
        :model-value="form.default_page_id"
        label="Página Padrão (CTWA `page_id`)"
        :options="pageOptions"
        placeholder="Automático pelo anúncio"
        :disabled="!oauthConnected"
        :loading="loadingAssets"
        @update:model-value="onSelectPage"
      />
    </div>

    <div v-if="remotePixels.length" class="pixels-box">
      <span class="pixels-title">Pixels encontrados nesta Conta de Anúncios</span>
      <div class="pixels-list">
        <div v-for="px in remotePixels" :key="px.id" class="pixel-chip">
          <span class="pixel-chip-name">{{ px.name }}</span>
          <code class="pixel-chip-id">{{ px.id }}</code>
          <button type="button" class="copy-btn" @click="copyId(px.id)">Copiar ID</button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { AdAccountSelectOption } from '~/components/BaseAdAccountSelect.vue'
import type { MetaRemoteAdAccount, MetaRemotePage, MetaRemotePixel } from '#shared/types'

interface FormState {
  name: string
  ad_account_id: string
  ad_account_name: string
  ad_account_picture_url: string
  default_page_id: string
  default_page_name: string
  default_page_picture_url: string
}

const props = defineProps<{
  form: FormState
  nameError: string
  oauthConnected: boolean
  oauthPictureUrl?: string
  remoteAdAccounts: MetaRemoteAdAccount[]
  remotePages: MetaRemotePage[]
  remotePixels: MetaRemotePixel[]
  loadingAssets: boolean
}>()

const emit = defineEmits<{
  'refresh-assets': []
  'ad-account-selected': [adAccountId: string]
}>()

const toast = useToast()

const normalizeAct = (a: MetaRemoteAdAccount) => (a.id.startsWith('act_') ? a.id : `act_${a.account_id || a.id}`)

const adAccountOptions = computed<AdAccountSelectOption[]>(() => {
  const opts: AdAccountSelectOption[] = [
    {
      value: '',
      label: props.oauthConnected ? 'Selecione uma conta...' : 'Conecte o OAuth primeiro',
      channel: 'none',
    },
  ]
  for (const a of props.remoteAdAccounts) {
    const id = normalizeAct(a)
    const sub = a.business_name ? `${id} · ${a.business_name}` : id
    opts.push({
      value: id,
      label: a.name || id,
      sublabel: sub,
      avatarUrl: a.picture_url || props.oauthPictureUrl || '',
      channel: 'meta',
    })
  }
  if (props.form.ad_account_id && !opts.some((o) => o.value === props.form.ad_account_id)) {
    opts.push({
      value: props.form.ad_account_id,
      label: props.form.ad_account_name || props.form.ad_account_id,
      sublabel: props.form.ad_account_id,
      avatarUrl: props.form.ad_account_picture_url || props.oauthPictureUrl || '',
      channel: 'meta',
    })
  }
  return opts
})

const pageOptions = computed<AdAccountSelectOption[]>(() => {
  const opts: AdAccountSelectOption[] = [
    { value: '', label: 'Automático pelo anúncio (ou selecione)', channel: 'none' },
  ]
  for (const p of props.remotePages) {
    opts.push({
      value: p.id,
      label: p.name,
      sublabel: p.category ? `${p.id} · ${p.category}` : p.id,
      avatarUrl: p.picture_url || '',
      channel: 'meta',
    })
  }
  if (props.form.default_page_id && !opts.some((o) => o.value === props.form.default_page_id)) {
    opts.push({
      value: props.form.default_page_id,
      label: props.form.default_page_name || props.form.default_page_id,
      sublabel: props.form.default_page_id,
      avatarUrl: props.form.default_page_picture_url || '',
      channel: 'meta',
    })
  }
  return opts
})

const onSelectAdAccount = (val: string) => {
  props.form.ad_account_id = String(val || '')
  const found = props.remoteAdAccounts.find((a) => normalizeAct(a) === props.form.ad_account_id)
  props.form.ad_account_name = found?.name || ''
  props.form.ad_account_picture_url = found?.picture_url || props.oauthPictureUrl || ''
  if (found?.name && (!props.form.name.trim() || props.form.name === 'Meta Ads')) {
    props.form.name = found.name
  }
  emit('ad-account-selected', props.form.ad_account_id)
}

const onSelectPage = (val: string) => {
  props.form.default_page_id = String(val || '')
  const found = props.remotePages.find((p) => p.id === props.form.default_page_id)
  props.form.default_page_name = found?.name || ''
  props.form.default_page_picture_url = found?.picture_url || ''
}

const copyId = async (id: string) => {
  try {
    await navigator.clipboard.writeText(id)
    toast.success('Copiado', `Pixel ID ${id} copiado.`)
  } catch {
    // fallback silencioso
  }
}
</script>

<style scoped>
.assets-section { display: flex; flex-direction: column; gap: 1rem; padding-bottom: 1.25rem; border-bottom: 1px solid var(--border-faint); }
.section-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 0.75rem; flex-wrap: wrap; }
.section-header h4 { font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); margin: 0.25rem 0 0; line-height: 1.5; max-width: 560px; }
.field-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.875rem; }
@media (max-width: 768px) { .field-grid { grid-template-columns: 1fr; } }
.pixels-box { display: flex; flex-direction: column; gap: 0.5rem; padding: 0.875rem; background: var(--surface-soft); border: 1px solid var(--border-soft); border-radius: 8px; }
.pixels-title { font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); }
.pixels-list { display: flex; flex-wrap: wrap; gap: 0.5rem; }
.pixel-chip { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.35rem 0.65rem; background: var(--surface-2); border: 1px solid var(--border); border-radius: 8px; font-size: var(--fs-sm); }
.pixel-chip-name { color: var(--text-1); font-weight: var(--fw-medium); }
.pixel-chip-id { font-family: var(--font-mono); color: var(--text-3); font-size: var(--fs-xs); }
.copy-btn { background: transparent; border: none; color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-semibold); cursor: pointer; padding: 0.1rem 0.25rem; }
.copy-btn:hover { text-decoration: underline; }
</style>
