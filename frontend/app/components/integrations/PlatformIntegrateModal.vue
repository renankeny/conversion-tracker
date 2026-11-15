<template>
  <BaseModal
    :model-value="modelValue"
    :title="platform ? `Integrar ${platform.name}` : 'Integrar Plataforma'"
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="platform" class="modal-stack">
      <div class="platform-banner">
        <div class="banner-logo">
          <img :src="platform.logoUrl" :alt="platform.name" />
        </div>
        <div class="banner-meta">
          <div class="banner-events">
            <BaseBadge v-for="ev in platform.events" :key="ev" variant="info">
              {{ ev }}
            </BaseBadge>
            <BaseBadge v-if="hasSavedToken" variant="success">Token ativo</BaseBadge>
          </div>
          <p class="banner-desc">{{ platform.description }}</p>
        </div>
        <BaseButton size="sm" variant="ghost" @click="goToLogs">
          <template #iconLeft><ExternalLink :size="13" /></template>
          Ver logs
        </BaseButton>
      </div>

      <div v-if="isGlobalTenantMode && accountOptions.length" class="section-block">
        <span class="section-label">Empresa de destino</span>
        <BaseDropdown
          :model-value="accountId"
          :options="accountOptions"
          placeholder="Selecione a empresa"
          @update:model-value="emit('update:accountId', String($event))"
        />
      </div>

      <div class="section-block">
        <span class="section-label">1. Destino e URL do Webhook ({{ platform.name }})</span>

        <div v-if="pixelOptions.length > 1" class="pixel-select-wrap">
          <BaseDropdown
            :model-value="selectedPixel"
            :options="pixelOptions"
            placeholder="Automático da empresa"
            @update:model-value="emit('update:selectedPixel', String($event))"
          />
          <p v-if="hasMultiplePixels && !selectedPixel" class="multi-pixel-hint">
            Esta empresa possui múltiplos Pixels. Se este checkout for exclusivo de um produto, selecione o Pixel acima para garantir atribuição exata mesmo em compras diretas.
          </p>
        </div>

        <div class="webhook-row">
          <code class="webhook-url">{{ webhookUrl }}</code>
          <BaseButton size="sm" variant="ghost" @click="emit('copy')">
            {{ copied ? '✓ Copiado' : 'Copiar' }}
          </BaseButton>
        </div>
      </div>

      <div v-if="platform.setupSteps.length" class="section-block">
        <span class="section-label">2. Passo a passo no painel {{ platform.name }}</span>
        <ol class="steps-list">
          <li v-for="(step, idx) in platform.setupSteps" :key="idx" class="step-item">
            <span class="step-num">{{ idx + 1 }}</span>
            <span class="step-text">{{ step }}</span>
          </li>
        </ol>
        <p v-if="platform.notes" class="notes-box">{{ platform.notes }}</p>
      </div>

      <div class="section-block">
        <div class="token-header">
          <span class="section-label">3. Token de Segurança / Secret (Opcional)</span>
          <span v-if="hasSavedToken && maskedToken" class="saved-mask">
            Salvo: {{ maskedToken }}
          </span>
        </div>

        <p class="token-hint">
          Se você cadastrou um Token, Hottok, Chave Única ou Secret HMAC na {{ platform.name }}, salve-o abaixo para validar a autenticidade de cada venda.
        </p>

        <template v-if="canManage">
          <div class="token-row">
            <div class="secret-input-wrap">
              <BaseInput
                :model-value="tokenValue"
                :type="showSecret ? 'text' : 'password'"
                :placeholder="hasSavedToken ? 'Digite um novo token para substituir o atual...' : 'Deixe em branco para aceitar sem validação de token...'"
                :disabled="loadingToken || savingToken"
                @update:model-value="emit('update:tokenValue', String($event))"
              />
              <button
                type="button"
                class="eye-btn"
                :title="showSecret ? 'Ocultar token' : 'Mostrar token'"
                @click="showSecret = !showSecret"
              >
                <EyeOff v-if="showSecret" :size="15" />
                <Eye v-else :size="15" />
              </button>
            </div>

            <BaseButton
              :loading="savingToken"
              :disabled="loadingToken || (!tokenValue.trim() && hasSavedToken)"
              @click="emit('saveToken')"
            >
              Salvar
            </BaseButton>

            <BaseButton
              v-if="hasSavedToken"
              variant="danger"
              size="sm"
              :disabled="savingToken || loadingToken"
              @click="emit('clearToken')"
            >
              Remover
            </BaseButton>
          </div>
        </template>

        <p v-else class="readonly-note">
          Apenas administradores ou gestores podem alterar o token de segurança desta plataforma.
        </p>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ExternalLink, Eye, EyeOff } from 'lucide-vue-next'
import type { PlatformDef } from '#shared/constants/platforms'

interface DropdownOption {
  value: string
  label: string
}

interface Props {
  modelValue: boolean
  platform: PlatformDef | null
  webhookUrl: string
  copied: boolean
  tokenValue: string
  hasSavedToken: boolean
  maskedToken: string
  loadingToken: boolean
  savingToken: boolean
  canManage: boolean
  pixelOptions: DropdownOption[]
  selectedPixel: string
  hasMultiplePixels: boolean
  isGlobalTenantMode: boolean
  accountId: string
  accountOptions: DropdownOption[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'update:tokenValue': [value: string]
  'update:selectedPixel': [value: string]
  'update:accountId': [value: string]
  copy: []
  saveToken: []
  clearToken: []
}>()

const showSecret = ref(false)

watch(() => props.modelValue, (open) => {
  if (!open) showSecret.value = false
})

const goToLogs = () => {
  if (!props.platform) return
  emit('update:modelValue', false)
  navigateTo(`/settings/webhooks?source=${encodeURIComponent(props.platform.id)}`)
}
</script>

<style scoped>
.modal-stack { display: flex; flex-direction: column; gap: 1.15rem; }
.platform-banner { display: flex; align-items: center; gap: 0.875rem; padding: 0.875rem; background: var(--surface-soft); border: 1px solid var(--border); border-radius: 10px; }
.banner-logo { width: 46px; height: 46px; border-radius: 11px; overflow: hidden; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: var(--surface-1); border: 1px solid var(--border); }
.banner-logo img { width: 100%; height: 100%; object-fit: cover; }
.banner-meta { flex: 1; display: flex; flex-direction: column; gap: 0.3rem; min-width: 0; }
.banner-events { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.banner-desc { font-family: var(--font-sans); font-size: var(--fs-base); color: var(--text-2); line-height: 1.4; }
.section-block { display: flex; flex-direction: column; gap: 0.45rem; }
.section-label { font-family: var(--font-sans); font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); }
.pixel-select-wrap { display: flex; flex-direction: column; gap: 0.35rem; }
.multi-pixel-hint { font-size: var(--fs-sm); color: var(--warning); line-height: 1.35; }
.webhook-row { display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem 0.625rem; background: var(--surface-deep); border: 1px solid var(--brand-border); border-radius: 8px; }
.webhook-url { flex: 1; font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--brand); word-break: break-all; user-select: all; }
.steps-list { display: flex; flex-direction: column; gap: 0.4rem; padding: 0.75rem; background: var(--surface-soft-2); border: 1px solid var(--border); border-radius: 8px; list-style: none; margin: 0; }
.step-item { display: flex; align-items: flex-start; gap: 0.5rem; font-family: var(--font-sans); font-size: var(--fs-base); color: var(--text-1); line-height: 1.4; }
.step-num { width: 18px; height: 18px; border-radius: 999px; background: var(--brand-soft); color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-semibold); display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; margin-top: 0.05rem; }
.step-text { flex: 1; }
.notes-box { font-family: var(--font-sans); font-size: var(--fs-sm); color: var(--text-3); line-height: 1.4; }
.token-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; flex-wrap: wrap; }
.saved-mask { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--success); background: var(--surface-soft); padding: 0.15rem 0.45rem; border-radius: 6px; }
.token-hint { font-family: var(--font-sans); font-size: var(--fs-base); color: var(--text-3); line-height: 1.4; }
.token-row { display: flex; gap: 0.5rem; align-items: center; flex-wrap: wrap; }
.secret-input-wrap { position: relative; flex: 1; min-width: 200px; }
.secret-input-wrap :deep(input) { padding-right: 2.25rem; }
.eye-btn { position: absolute; right: 0.55rem; top: 50%; transform: translateY(-50%); background: transparent; border: none; color: var(--text-3); cursor: pointer; display: inline-flex; align-items: center; justify-content: center; padding: 0.2rem; }
.eye-btn:hover { color: var(--text-1); }
.readonly-note { font-size: var(--fs-sm); color: var(--text-3); background: var(--surface-soft-2); border: 1px solid var(--border-soft); padding: 0.55rem 0.75rem; border-radius: 8px; }
</style>
