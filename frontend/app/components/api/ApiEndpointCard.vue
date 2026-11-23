<script setup lang="ts">
import { ChevronDown, Copy, Check } from 'lucide-vue-next'
import { buildEndpointCodeTabs, type ApiEndpointSpec } from '~/utils/apiCatalog'

const props = withDefaults(
  defineProps<{
    endpoint: ApiEndpointSpec
    apiUrl: string
    defaultOpen?: boolean
  }>(),
  { defaultOpen: false }
)

const isOpen = ref(props.defaultOpen)
const copiedPath = ref(false)

watch(
  () => props.defaultOpen,
  (val) => {
    isOpen.value = val
  }
)

const fullEndpointUrl = computed(() => `${props.apiUrl.replace(/\/+$/, '')}${props.endpoint.path}`)
const codeTabs = computed(() => buildEndpointCodeTabs(props.apiUrl, props.endpoint))

const methodClass = computed(() => {
  switch (props.endpoint.method) {
    case 'POST':
      return 'method-post'
    case 'PUT':
      return 'method-put'
    case 'DELETE':
      return 'method-delete'
    default:
      return 'method-get'
  }
})

const copyEndpoint = async (e: Event) => {
  e.stopPropagation()
  await navigator.clipboard.writeText(fullEndpointUrl.value)
  copiedPath.value = true
  setTimeout(() => {
    copiedPath.value = false
  }, 1800)
}
</script>

<template>
  <article class="ep-card" :class="{ open: isOpen }">
    <button type="button" class="ep-summary" @click="isOpen = !isOpen">
      <div class="ep-main">
        <span class="method-pill" :class="methodClass">{{ endpoint.method }}</span>
        <code class="ep-path">{{ endpoint.path }}</code>
        <span class="ep-title">{{ endpoint.title }}</span>
      </div>

      <div class="ep-actions">
        <span class="copy-url-btn" role="button" tabindex="0" title="Copiar URL completa" @click="copyEndpoint">
          <Check v-if="copiedPath" class="icon-xs text-brand" />
          <Copy v-else class="icon-xs" />
        </span>
        <ChevronDown class="chevron-icon" :class="{ rotated: isOpen }" />
      </div>
    </button>

    <div v-if="isOpen" class="ep-details">
      <p class="ep-desc">{{ endpoint.summary }}</p>

      <div v-if="endpoint.note" class="ep-note">
        {{ endpoint.note }}
      </div>

      <div v-if="endpoint.params?.length" class="params-block">
        <h4 class="sub-label">Parâmetros Principais</h4>
        <div class="params-table">
          <div v-for="p in endpoint.params" :key="p.name" class="param-row">
            <div class="param-meta">
              <code class="param-name">{{ p.name }}</code>
              <span class="param-loc">{{ p.location }} · {{ p.type }}</span>
              <span v-if="p.required" class="param-req">obrigatório</span>
            </div>
            <span class="param-desc">{{ p.description }}</span>
          </div>
        </div>
      </div>

      <div class="snippets-grid">
        <div>
          <h4 class="sub-label">Exemplo de Chamada</h4>
          <CodeTabs :tabs="codeTabs" />
        </div>
        <div>
          <h4 class="sub-label">Resposta JSON</h4>
          <pre class="resp-box"><code>{{ endpoint.sampleResponse }}</code></pre>
        </div>
      </div>
    </div>
  </article>
</template>

<style scoped>
.ep-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; transition: border-color 0.15s ease; }
.ep-card.open { border-color: var(--brand-border); }
.ep-summary { width: 100%; display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 0.9rem 1.1rem; background: transparent; border: none; cursor: pointer; text-align: left; }
.ep-summary:hover { background: var(--surface-soft-2); }
.ep-main { display: flex; align-items: center; flex-wrap: wrap; gap: 0.65rem; min-width: 0; }
.method-pill { font-family: var(--font-mono); font-size: var(--fs-xs); font-weight: var(--fw-bold); padding: 0.2rem 0.55rem; border-radius: 6px; letter-spacing: 0.04em; flex-shrink: 0; }
.method-get { background: var(--accent-sky-soft); color: var(--accent-sky); border: 1px solid var(--accent-sky-border); }
.method-post { background: var(--brand-soft); color: var(--brand); border: 1px solid var(--brand-border); }
.method-put { background: var(--warning-soft); color: var(--warning-light); border: 1px solid var(--warning-border); }
.method-delete { background: var(--danger-soft); color: var(--danger-light); border: 1px solid var(--danger-border); }
.ep-path { font-family: var(--font-mono); font-size: var(--fs-base); color: var(--text-strong); }
.ep-title { font-size: var(--fs-md); color: var(--text-2); }
.ep-actions { display: flex; align-items: center; gap: 0.5rem; flex-shrink: 0; }
.copy-url-btn { display: inline-flex; align-items: center; justify-content: center; padding: 0.3rem; border-radius: 6px; color: var(--text-3); }
.copy-url-btn:hover { background: var(--surface-soft); color: var(--text-1); }
.icon-xs { width: 0.85rem; height: 0.85rem; }
.text-brand { color: var(--brand); }
.chevron-icon { width: 1rem; height: 1rem; color: var(--text-3); transition: transform 0.18s ease; }
.chevron-icon.rotated { transform: rotate(180deg); }
.ep-details { padding: 1rem 1.1rem 1.15rem; border-top: 1px solid var(--border); display: flex; flex-direction: column; gap: 1rem; }
.ep-desc { font-size: var(--fs-md); color: var(--text-2); line-height: 1.5; }
.ep-note { font-size: var(--fs-sm); color: var(--text-1); background: var(--brand-soft-2); border: 1px solid var(--brand-border); border-radius: 8px; padding: 0.65rem 0.85rem; }
.sub-label { font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); margin-bottom: 0.5rem; }
.params-table { display: flex; flex-direction: column; border: 1px solid var(--border); border-radius: 9px; background: var(--surface-deep); }
.param-row { display: flex; align-items: baseline; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem; padding: 0.55rem 0.8rem; border-bottom: 1px solid var(--border); }
.param-row:last-child { border-bottom: none; }
.param-meta { display: flex; align-items: center; gap: 0.5rem; }
.param-name { font-family: var(--font-mono); font-size: var(--fs-base); color: var(--brand); }
.param-loc { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-3); }
.param-req { font-size: var(--fs-2xs); text-transform: uppercase; color: var(--warning-light); background: var(--warning-soft); padding: 0.08rem 0.35rem; border-radius: 4px; }
.param-desc { font-size: var(--fs-sm); color: var(--text-2); }
.snippets-grid { display: grid; grid-template-columns: 1.35fr 0.85fr; gap: 0.875rem; align-items: start; }
.resp-box { background: var(--surface-code); border: 1px solid var(--border); border-radius: 12px; padding: 1rem; font-family: var(--font-mono); font-size: var(--fs-base); color: var(--text-2); overflow-x: auto; line-height: 1.55; }
@media (max-width: 900px) {
  .snippets-grid { grid-template-columns: 1fr; }
}
</style>
