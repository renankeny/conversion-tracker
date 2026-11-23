<script setup lang="ts">
import { Sparkles, Copy, Check, KeyRound } from 'lucide-vue-next'
import { buildMcpSetupTabs, MCP_TOOL_CATEGORIES } from '~/utils/apiCatalog'

const props = defineProps<{
  apiUrl: string
}>()

const emit = defineEmits<{
  'open-keys': []
}>()

const mcpUrl = computed(() => `${props.apiUrl.replace(/\/+$/, '')}/mcp`)
const setupTabs = computed(() => buildMcpSetupTabs(props.apiUrl))
const copiedUrl = ref(false)

const copyMcpUrl = async () => {
  await navigator.clipboard.writeText(mcpUrl.value)
  copiedUrl.value = true
  setTimeout(() => {
    copiedUrl.value = false
  }, 2000)
}
</script>

<template>
  <div class="mcp-section">
    <div class="mcp-card">
      <div class="mcp-header">
        <div>
          <div class="mcp-kicker">
            <Sparkles class="kicker-icon" />
            <span>SERVIDOR MCP NATIVO EMBUTIDO</span>
          </div>
          <h2 class="mcp-title">Plugar Agentes de IA em 1 Clique</h2>
          <p class="mcp-desc">
            Seu sistema já possui um servidor <strong>Model Context Protocol (MCP)</strong> nativo pronto para uso.
            <strong>A mesma API Key funciona na API REST e no MCP</strong>, respeitando exatamente os módulos e o modo
            somente leitura que você escolher ao criar ou editar a chave.
          </p>
        </div>
        <BaseButton variant="primary" size="sm" @click="emit('open-keys')">
          <KeyRound class="btn-icon" />
          Ver / Criar Chave
        </BaseButton>
      </div>

      <div class="endpoint-strip">
        <div class="strip-item">
          <span class="strip-label">URL do Servidor MCP</span>
          <div class="strip-code-row">
            <code class="strip-code">{{ mcpUrl }}</code>
            <button type="button" class="copy-pill" @click="copyMcpUrl">
              <Check v-if="copiedUrl" class="pill-icon text-brand" />
              <Copy v-else class="pill-icon" />
              <span>{{ copiedUrl ? 'Copiado' : 'Copiar URL' }}</span>
            </button>
          </div>
        </div>
        <div class="strip-item">
          <span class="strip-label">Header Único de Autenticação</span>
          <div class="strip-code-row">
            <code class="strip-code">X-API-Key: trk_sua_chave_aqui</code>
          </div>
        </div>
      </div>

      <div class="setup-block">
        <h3 class="block-subtitle">Configuração rápida no seu agente ou automação</h3>
        <CodeTabs :tabs="setupTabs" />
      </div>
    </div>

    <div class="tools-section">
      <h3 class="block-subtitle">O que sua IA consegue executar automaticamente (39 ferramentas nativas)</h3>
      <div class="tools-grid">
        <div v-for="cat in MCP_TOOL_CATEGORIES" :key="cat.title" class="tool-cat-card">
          <h4 class="cat-title">{{ cat.title }}</h4>
          <p class="cat-desc">{{ cat.description }}</p>
          <div class="tool-chips">
            <code v-for="t in cat.tools" :key="t" class="tool-chip">{{ t }}</code>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mcp-section { display: flex; flex-direction: column; gap: 1.5rem; }
.mcp-card { background: var(--surface-2); border: 1px solid var(--brand-border); border-radius: 14px; padding: 1.5rem; display: flex; flex-direction: column; gap: 1.25rem; }
.mcp-header { display: flex; align-items: flex-start; justify-content: space-between; flex-wrap: wrap; gap: 1rem; }
.mcp-kicker { display: inline-flex; align-items: center; gap: 0.4rem; font-size: var(--fs-xs); font-weight: var(--fw-semibold); color: var(--brand); letter-spacing: 0.06em; margin-bottom: 0.35rem; }
.kicker-icon, .btn-icon, .pill-icon { width: 0.9rem; height: 0.9rem; }
.text-brand { color: var(--brand); }
.mcp-title { font-family: var(--font-sans); font-size: var(--fs-2xl); font-weight: var(--fw-bold); color: var(--text-strong); margin-bottom: 0.35rem; }
.mcp-desc { font-size: var(--fs-md); color: var(--text-2); line-height: 1.55; max-width: 68ch; }
.endpoint-strip { display: grid; grid-template-columns: 1.3fr 1fr; gap: 0.75rem; }
.strip-item { background: var(--surface-deep); border: 1px solid var(--border); border-radius: 10px; padding: 0.75rem 0.9rem; display: flex; flex-direction: column; gap: 0.35rem; }
.strip-label { font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); }
.strip-code-row { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.strip-code { font-family: var(--font-mono); font-size: var(--fs-base); color: var(--brand); word-break: break-all; }
.copy-pill { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.25rem 0.6rem; border-radius: 6px; border: 1px solid var(--border); background: var(--surface-soft); color: var(--text-1); font-size: var(--fs-xs); font-weight: var(--fw-medium); cursor: pointer; flex-shrink: 0; }
.copy-pill:hover { border-color: var(--brand-border); }
.block-subtitle { font-family: var(--font-sans); font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.75rem; }
.tools-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.875rem; }
.tool-cat-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.1rem; }
.cat-title { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--text-strong); margin-bottom: 0.25rem; }
.cat-desc { font-size: var(--fs-sm); color: var(--text-2); margin-bottom: 0.65rem; line-height: 1.45; }
.tool-chips { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.tool-chip { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-1); background: var(--surface-deep); border: 1px solid var(--border); padding: 0.15rem 0.45rem; border-radius: 6px; }
@media (max-width: 768px) {
  .endpoint-strip, .tools-grid { grid-template-columns: 1fr; }
}
</style>
