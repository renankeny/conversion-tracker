<template>
  <section class="ai-stack">
    <div v-if="loading" class="card loading-box">
      <div class="spinner" />
      <span>Carregando configurações da IA...</span>
    </div>

    <template v-else>
      <div class="card">
        <div class="card-top">
          <div>
            <h3 class="card-title">Analista Comercial de WhatsApp (OpenAI)</h3>
            <p class="card-hint">
              Analisa conversas a cada <strong>15 minutos</strong> (textos, áudios, imagens e comprovantes em PDF; ignora vídeos), move cards automaticamente no Kanban e dispara conversões sem duplicar eventos já enviados por gatilhos ou webhooks.
            </p>
          </div>

          <div class="top-actions">
            <span v-if="lastRunLabel" class="run-meta">Último ciclo: {{ lastRunLabel }}</span>
            <BaseButton variant="outline" size="sm" :loading="runningNow" @click="onRunNow">
              <Sparkles :size="14" />
              <span>Analisar agora</span>
            </BaseButton>
          </div>
        </div>

        <form class="ai-form" @submit.prevent="onSave">
          <div class="key-banner">
            <div class="key-banner-info">
              <span class="key-pill" :class="keyReady ? 'ok' : 'warn'">
                {{ keyReady ? 'Chave OpenAI ativa' : 'Chave OpenAI pendente' }}
              </span>
              <span class="key-banner-text">
                {{ keyReady ? 'Utilizando a chave da OpenAI conectada na aba Meu perfil.' : 'Adicione sua chave da OpenAI na aba Meu perfil para ativar a IA em toda a plataforma.' }}
              </span>
            </div>
            <BaseButton
              v-if="!keyReady"
              variant="outline"
              size="sm"
              type="button"
              @click="$emit('goProfile')"
            >
              Configurar em Meu perfil
            </BaseButton>
          </div>

          <div class="toggle-strip">
            <BaseCheckbox
              v-model="form.enabled"
              label="Ativar análise automática de conversas a cada 15 minutos"
              hint="Leads já qualificados não são reavaliados para qualificação; novas compras na mesma conversa continuam sendo identificadas."
            />
          </div>

          <div class="grid-2">
            <BaseDropdown
              :model-value="form.qualified_stage_id"
              label="Etapa de Lead Qualificado no Kanban"
              :options="qualStageOptions"
              hint="Para onde o card vai quando a IA qualificar o atendimento."
              @update:model-value="form.qualified_stage_id = Number($event || 0)"
            />
            <BaseDropdown
              :model-value="form.sale_stage_id"
              label="Etapa de Venda Realizada no Kanban"
              :options="saleStageOptions"
              hint="Para onde o card vai quando a IA confirmar pagamento/venda."
              @update:model-value="form.sale_stage_id = Number($event || 0)"
            />
          </div>

          <div class="prompt-block">
            <div class="prompt-head">
              <label class="field-label" for="ai-base-prompt">Prompt Base do Analista</label>
              <button type="button" class="reset-link" @click="restoreDefaultPrompt">
                <RotateCcw :size="12" />
                <span>Restaurar prompt padrão</span>
              </button>
            </div>
            <textarea
              id="ai-base-prompt"
              v-model="form.base_prompt"
              rows="11"
              class="prompt-textarea"
              placeholder="Instruções comerciais para qualificação e identificação de venda..."
            />
          </div>

          <div class="form-actions">
            <BaseButton variant="primary" type="submit" :loading="saving">
              Salvar configurações da IA
            </BaseButton>
          </div>
        </form>
      </div>

      <SettingsWhatsAppAiRulesCard
        :rules="form.learned_rules"
        @remove="removeRule"
        @add="addRule"
      />
    </template>
  </section>
</template>

<script setup lang="ts">
import { Sparkles, RotateCcw } from 'lucide-vue-next'
import SettingsWhatsAppAiRulesCard from '~/components/settings/SettingsWhatsAppAiRulesCard.vue'

defineEmits<{
  goProfile: []
}>()

const {
  loading, saving, runningNow, form,
  keyReady, lastRunLabel,
  qualStageOptions, saleStageOptions,
  restoreDefaultPrompt, removeRule, addRule, onSave, onRunNow,
} = useWhatsAppAiSettingsForm()
</script>

<style scoped>
.ai-stack { display: flex; flex-direction: column; gap: 1rem; }
.card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; box-shadow: var(--shadow-card); }
.loading-box { display: flex; align-items: center; gap: 0.75rem; color: var(--text-2); font-size: var(--fs-md); }
.card-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; flex-wrap: wrap; margin-bottom: 1.1rem; }
.card-title { font-family: var(--font-sans); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0 0 0.35rem; }
.card-hint { font-size: var(--fs-md); color: var(--text-2); margin: 0; line-height: 1.5; max-width: 680px; }
.top-actions { display: flex; align-items: center; gap: 0.75rem; flex-shrink: 0; }
.run-meta { font-size: var(--fs-xs); color: var(--text-3); }
.ai-form { display: flex; flex-direction: column; gap: 1rem; }
.key-banner { padding: 0.75rem 1rem; border-radius: 10px; background: var(--surface-deep); border: 1px solid var(--border-soft); display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.key-banner-info { display: flex; align-items: center; gap: 0.65rem; flex-wrap: wrap; }
.key-banner-text { font-size: var(--fs-sm); color: var(--text-2); }
.toggle-strip { padding: 0.9rem 1rem; border-radius: 10px; background: var(--surface-deep); border: 1px solid var(--border-soft); display: flex; flex-direction: column; }
.field-label { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-1); }
.key-pill { padding: 0.2rem 0.6rem; border-radius: 999px; font-size: var(--fs-2xs); font-weight: var(--fw-semibold); }
.key-pill.ok { background: var(--surface-soft); color: var(--success); border: 1px solid var(--border-soft); }
.key-pill.warn { background: var(--surface-soft); color: var(--warning); border: 1px solid var(--border-soft); }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0.875rem; }
.prompt-block { display: flex; flex-direction: column; gap: 0.45rem; }
.prompt-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.reset-link { display: inline-flex; align-items: center; gap: 0.35rem; background: transparent; border: none; color: var(--brand); font-size: var(--fs-xs); font-weight: var(--fw-semibold); cursor: pointer; padding: 0.2rem 0.4rem; border-radius: 6px; }
.reset-link:hover { background: var(--brand-soft); }
.prompt-textarea { width: 100%; padding: 0.85rem 0.95rem; border-radius: 10px; background: var(--surface-deep); border: 1px solid var(--border); color: var(--text-1); font-family: var(--font-sans); font-size: var(--fs-base); line-height: 1.55; resize: vertical; outline: none; transition: border-color 0.15s ease; }
.prompt-textarea:focus { border-color: var(--brand); }
.form-actions { display: flex; justify-content: flex-end; }
.spinner { width: 1.25rem; height: 1.25rem; border: 2px solid var(--border); border-top-color: var(--brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 680px) { .grid-2 { grid-template-columns: 1fr; } }
</style>
