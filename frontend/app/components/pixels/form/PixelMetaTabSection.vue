<template>
  <div class="tab-content">
    <div class="oauth-link-box">
      <BaseAdAccountSelect
        v-model="metaAccountSelectValue"
        label="Conta de Anúncios Meta Ads (OAuth)"
        :options="metaAccountOptions"
        :loading="loadingAccounts"
      />
      <p class="field-help">
        Vincule uma conta em <NuxtLink to="/integracoes/meta-ads" class="link">Integrações &gt; Meta Ads</NuxtLink> para carregar os Pixels da conta automaticamente, enriquecer anúncios CTWA, exibir investimento ao vivo e disparar CAPI sem token manual.
      </p>
    </div>

    <div class="section-header">
      <h4 class="section-title">
        <BrandIcon channel="meta" size="sm" :colored="true" />
        Meta Conversions API (Pixels)
      </h4>
      <BaseButton type="button" variant="ghost" size="sm" @click="addPixelRow">
        + Adicionar pixel
      </BaseButton>
    </div>
    <p class="section-hint">
      {{ remotePixels.length
        ? 'O Pixel da sua Conta de Anúncios já vem selecionado e configurado automaticamente para envio CAPI via OAuth.'
        : 'Espelha eventos em tempo real para todos os Pixel IDs abaixo. Se uma Conta Meta OAuth estiver selecionada acima, não é necessário informar token manual.' }}
    </p>

    <div v-if="loadingPixels" class="empty-channel-box">
      Carregando Pixels da Conta de Anúncios...
    </div>
    <div v-else-if="!form.pixel_ids.length" class="empty-channel-box">
      Nenhum pixel da Meta configurado para este tracker.
      <span v-if="pixelIdsError" class="inline-error block-err">{{ pixelIdsError }}</span>
    </div>

    <div v-for="(_, idx) in form.pixel_ids" :key="idx" class="pixel-row">
      <div class="pixel-row-grid">
        <div v-if="remotePixels.length && !manualRows[idx]" class="pixel-select-col">
          <BaseDropdown
            :model-value="pixelDropdownValue(idx)"
            label="Pixel da Meta"
            :options="pixelDropdownOptions"
            placeholder="Escolha um Pixel..."
            @update:model-value="(v) => onSelectPixel(idx, String(v))"
          />
          <span v-if="idx === 0 && pixelIdsError" class="inline-error">{{ pixelIdsError }}</span>
        </div>

        <div v-else class="pixel-select-col">
          <BaseInput
            v-model="form.pixel_ids[idx]"
            label="Pixel ID"
            placeholder="123456789012345"
            :error="idx === 0 ? pixelIdsError : ''"
          />
          <button
            v-if="remotePixels.length"
            type="button"
            class="switch-mode-btn"
            @click="switchToList(idx)"
          >
            Escolher Pixel da conta conectada
          </button>
        </div>

        <div v-if="isOAuthAutoTokenRow(idx)" class="pixel-select-col">
          <span class="auto-token-label">Autenticação CAPI</span>
          <div class="oauth-auto-pill">
            <span class="oauth-auto-dot" />
            <span>Automático via OAuth</span>
          </div>
          <button type="button" class="switch-mode-btn" @click="enableManualTokenRow(idx)">
            Informar token manual (opcional)
          </button>
        </div>

        <div v-else class="pixel-select-col">
          <BaseInput
            v-model="form.access_tokens[idx]"
            :label="form.meta_account_id ? 'Access Token (opcional)' : 'Access Token'"
            type="password"
            autocomplete="new-password"
            :placeholder="tokenPlaceholder(idx)"
          />
          <button
            v-if="form.meta_account_id"
            type="button"
            class="switch-mode-btn"
            @click="resetRowToOAuth(idx)"
          >
            {{ hasSavedManualMetaToken(form.pixel_ids[idx] || '') ? 'Remover token manual e usar OAuth' : 'Usar envio automático via OAuth' }}
          </button>
        </div>
      </div>
      <button type="button" class="row-remove" aria-label="Remover pixel Meta" @click="removePixelRow(idx)">×</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import BrandIcon from '~/components/events/BrandIcon.vue'
import { usePixelMetaTab, type MetaTabFormState } from '~/composables/usePixelMetaTab'

const props = defineProps<{
  form: MetaTabFormState
  pixelIdsError: string
  isEditing: boolean
  hasSavedManualMetaToken: (pixelId: string) => boolean
}>()

const emit = defineEmits<{
  clearManualMetaToken: [idx: number]
}>()

const {
  remotePixels, loadingAccounts, loadingPixels, manualRows,
  metaAccountOptions, metaAccountSelectValue,
  pixelDropdownOptions, pixelDropdownValue,
  onSelectPixel, switchToList, addPixelRow, removePixelRow,
} = usePixelMetaTab(props)

const manualTokenRows = reactive<Record<number, boolean>>({})

const isOAuthAutoTokenRow = (idx: number): boolean => {
  if (!props.form.meta_account_id || manualTokenRows[idx]) return false
  const pid = props.form.pixel_ids[idx] || ''
  return !props.hasSavedManualMetaToken(pid) && !(props.form.access_tokens[idx] || '').trim()
}

const enableManualTokenRow = (idx: number) => {
  manualTokenRows[idx] = true
}

const resetRowToOAuth = (idx: number) => {
  manualTokenRows[idx] = false
  emit('clearManualMetaToken', idx)
}

const tokenPlaceholder = (idx: number): string => {
  const pid = props.form.pixel_ids[idx] || ''
  if (props.hasSavedManualMetaToken(pid)) return 'Token manual salvo (vazio mantém)'
  if (props.form.meta_account_id) return 'EAAxxxxxxxxxxxx... (opcional)'
  return props.isEditing ? 'Informe o Access Token' : 'EAAxxxxxxxxxxxx...'
}
</script>

<style scoped>
.tab-content { display: flex; flex-direction: column; gap: 1rem; }
.oauth-link-box { display: flex; flex-direction: column; gap: 0.5rem; padding: 0.875rem; background: var(--surface-soft); border: 1px solid var(--border-soft); border-radius: 10px; }
.field-help { font-size: var(--fs-sm); color: var(--text-3); margin: 0; line-height: 1.45; }
.link { color: var(--brand); text-decoration: underline; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); display: inline-flex; align-items: center; gap: 0.5rem; }
.section-hint { font-size: var(--fs-base); color: var(--text-3); margin: 0; }
.empty-channel-box { padding: 1.25rem; text-align: center; border: 1px dashed var(--border-soft); border-radius: 8px; font-size: var(--fs-md); color: var(--text-3); }
.block-err { display: block; margin-top: 0.375rem; }
.pixel-row { position: relative; padding: 0.75rem 2.25rem 0.75rem 0.75rem; background: var(--surface-soft); border: 1px solid var(--border-soft); border-radius: 10px; }
.pixel-row-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; align-items: start; }
@media (max-width: 768px) { .pixel-row-grid { grid-template-columns: 1fr; } }
.pixel-select-col { display: flex; flex-direction: column; gap: 0.25rem; min-width: 0; }
.auto-token-label { font-size: var(--fs-base); font-weight: var(--fw-medium); color: var(--text-2); margin-bottom: 0.125rem; }
.oauth-auto-pill { display: flex; align-items: center; gap: 0.5rem; height: 36px; padding: 0 0.75rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface-2); color: var(--success); font-size: var(--fs-sm); font-weight: var(--fw-semibold); }
.oauth-auto-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--success); flex-shrink: 0; }
.switch-mode-btn { align-self: flex-start; background: transparent; border: none; padding: 0; color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-medium); cursor: pointer; }
.switch-mode-btn:hover { text-decoration: underline; }
.inline-error { font-size: var(--fs-xs); color: var(--danger); }
.row-remove { position: absolute; top: 0.25rem; right: 0.5rem; background: transparent; border: none; color: var(--text-3); font-size: var(--fs-2xl); line-height: 1; cursor: pointer; padding: 0.25rem 0.5rem; border-radius: 6px; transition: all 0.15s; }
.row-remove:hover { color: var(--danger); background: var(--danger-soft); }
</style>
