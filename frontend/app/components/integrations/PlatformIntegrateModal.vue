<template>
  <BaseModal
    :model-value="modelValue"
    :title="platform ? `Integrar ${platform.name}` : 'Integrar Plataforma'"
    size="md"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="platform" class="modal-stack">
      <div class="platform-banner">
        <div class="banner-logo" :style="{ background: platform.brandColor }">
          <img :src="platform.logoUrl" :alt="platform.name" />
        </div>
        <div class="banner-meta">
          <div class="banner-events">
            <BaseBadge v-for="ev in platform.events" :key="ev" variant="info">
              {{ ev }}
            </BaseBadge>
          </div>
          <p class="banner-desc">{{ platform.description }}</p>
        </div>
      </div>

      <div class="section-block">
        <span class="section-label">1. URL do Webhook para {{ platform.name }}</span>
        <div class="webhook-row">
          <code class="webhook-url">{{ webhookUrl }}</code>
          <BaseButton size="sm" variant="ghost" @click="emit('copy', platform)">
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
        <span class="section-label">3. Token de Segurança / Secret (Opcional)</span>
        <p class="token-hint">
          Se você cadastrou um Token, Hottok, Chave Única ou Secret HMAC na {{ platform.name }}, salve-o abaixo para validar a autenticidade de cada venda.
        </p>
        <div class="token-row">
          <BaseInput
            :model-value="tokenValue"
            placeholder="Deixe em branco para aceitar sem validação de token..."
            :disabled="loadingToken"
            @update:model-value="emit('update:tokenValue', String($event))"
          />
          <BaseButton :loading="savingToken" @click="emit('saveToken')">
            Salvar
          </BaseButton>
        </div>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import type { PlatformDef } from '#shared/constants/platforms'

interface Props {
  modelValue: boolean
  platform: PlatformDef | null
  webhookUrl: string
  copied: boolean
  tokenValue: string
  loadingToken: boolean
  savingToken: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'update:tokenValue': [value: string]
  copy: [platform: PlatformDef]
  saveToken: []
}>()
</script>

<style scoped>
.modal-stack { display: flex; flex-direction: column; gap: 1.25rem; }
.platform-banner { display: flex; align-items: center; gap: 0.875rem; padding: 0.875rem; background: var(--surface-soft); border: 1px solid var(--border); border-radius: 10px; }
.banner-logo { width: 48px; height: 48px; border-radius: 12px; overflow: hidden; flex-shrink: 0; display: flex; align-items: center; justify-content: center; border: 1px solid var(--border); }
.banner-logo img { width: 100%; height: 100%; object-fit: cover; background: var(--surface-1); }
.banner-meta { display: flex; flex-direction: column; gap: 0.35rem; min-width: 0; }
.banner-events { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.banner-desc { font-family: var(--font-sans); font-size: var(--fs-base); color: var(--text-2); line-height: 1.4; }
.section-block { display: flex; flex-direction: column; gap: 0.45rem; }
.section-label { font-family: var(--font-sans); font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); }
.webhook-row { display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem 0.625rem; background: var(--surface-deep); border: 1px solid var(--brand-border); border-radius: 8px; }
.webhook-url { flex: 1; font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--brand); word-break: break-all; user-select: all; }
.steps-list { display: flex; flex-direction: column; gap: 0.4rem; padding: 0.75rem; background: var(--surface-soft-2); border: 1px solid var(--border); border-radius: 8px; list-style: none; margin: 0; }
.step-item { display: flex; align-items: flex-start; gap: 0.5rem; font-family: var(--font-sans); font-size: var(--fs-base); color: var(--text-1); line-height: 1.4; }
.step-num { width: 18px; height: 18px; border-radius: 999px; background: var(--brand-soft); color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-semibold); display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; margin-top: 0.05rem; }
.step-text { flex: 1; }
.notes-box { font-family: var(--font-sans); font-size: var(--fs-sm); color: var(--text-3); line-height: 1.4; margin-top: 0.15rem; }
.token-hint { font-family: var(--font-sans); font-size: var(--fs-base); color: var(--text-3); line-height: 1.4; }
.token-row { display: flex; gap: 0.5rem; align-items: center; }
.token-row > *:first-child { flex: 1; }
</style>

