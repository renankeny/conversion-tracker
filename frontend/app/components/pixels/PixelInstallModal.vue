<template>
  <BaseModal
    :model-value="modelValue"
    title="Instalar Pixel"
    :description="pixel ? `Configure e instale o pixel &quot;${pixel.label || pixel.name}&quot;` : ''"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="pixel" class="install-content">
      <div class="tabs">
        <button type="button" class="tab" :class="{ active: tab === 'script' }" @click="tab = 'script'">Script</button>
        <button type="button" class="tab" :class="{ active: tab === 'links' }" @click="tab = 'links'">
          Links Rastreáveis
          <span v-if="linksCount > 0" class="badge-pill">{{ linksCount }}</span>
        </button>
        <button type="button" class="tab" :class="{ active: tab === 'events' }" @click="tab = 'events'">Eventos &amp; Teste</button>
      </div>

      <div v-show="tab === 'script'" class="tab-pane">
        <div v-if="currentToken" class="section">
          <div class="section-header">
            <h4 class="section-title">Token público</h4>
            <div class="header-actions">
              <span class="badge-pill">Opaco · Seguro</span>
              <BaseButton v-if="canEdit" variant="ghost" size="sm" @click="showRegenConfirm = true">
                Regenerar
              </BaseButton>
            </div>
          </div>
          <p class="section-hint">Identificador usado no snippet. Não expõe o slug interno do pixel no código-fonte do site.</p>
          <div class="copy-row">
            <code class="copy-val">{{ currentToken }}</code>
            <button type="button" class="copy-mini" title="Copiar token público" @click="copy(currentToken, 'token')">
              <Check v-if="miniCopied === 'token'" :size="14" />
              <Copy v-else :size="14" />
            </button>
          </div>
        </div>

        <div class="section">
          <div class="section-header">
            <h4 class="section-title">Cole no &lt;head&gt; do site</h4>
            <BaseButton variant="outline" size="sm" @click="copySnippet">
              {{ copied ? 'Copiado!' : 'Copiar' }}
            </BaseButton>
          </div>
          <pre class="code-block"><code>{{ snippet }}</code></pre>
          <p class="section-hint">Funciona em WordPress, Shopify, páginas próprias e Google Tag Manager. Cole antes de <code>&lt;/head&gt;</code>.</p>
        </div>

        <div class="section">
          <h4 class="section-title">Disparar eventos manuais via JavaScript</h4>
          <pre class="code-block small"><code>{{ manualSnippet }}</code></pre>
        </div>

        <div class="section">
          <h4 class="section-title">Domínios autorizados</h4>
          <div class="domains-list">
            <span v-for="d in pixel.domains" :key="d" class="domain-pill">{{ d }}</span>
            <span v-if="!pixel.domains?.length" class="hint-warn">
              Nenhum domínio cadastrado. Edite o pixel para autorizar os domínios onde o script será instalado.
            </span>
          </div>
        </div>
      </div>

      <PixelTrackableLinksSection
        v-show="tab === 'links'"
        :pixel="pixel"
        :can-edit="canEdit"
        @count-updated="linksCount = $event"
      />

      <PixelTestEventSection
        v-show="tab === 'events'"
        :pixel="pixel"
        :can-edit="canEdit"
        @edit-config="$emit('edit-config', $event)"
      />
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">Fechar</BaseButton>
      <BaseButton v-if="tab === 'script'" variant="primary" @click="copySnippet">
        {{ copied ? 'Copiado!' : 'Copiar Script' }}
      </BaseButton>
    </template>
  </BaseModal>

  <ConfirmModal
    v-model="showRegenConfirm"
    title="Gerar novo token público?"
    message="O token atual deixará de funcionar imediatamente. Atualize o snippet instalado no site logo após gerar o novo token."
    confirm-label="Gerar novo token"
    variant="danger"
    :loading="regenerating"
    @confirm="onRegenerateToken"
  />
</template>

<script setup lang="ts">
import { Copy, Check } from 'lucide-vue-next'
import ConfirmModal from '~/components/ConfirmModal.vue'
import PixelTestEventSection from '~/components/pixels/PixelTestEventSection.vue'
import PixelTrackableLinksSection from '~/components/pixels/links/PixelTrackableLinksSection.vue'
import { buildPixelSnippet } from '~/utils/apiOrigin'
import { errorMessage } from '~/utils/errorMessage'
import type { Pixel } from '#shared/types'

const props = withDefaults(defineProps<{
  modelValue: boolean
  pixel: Pixel | null
  canEdit?: boolean
}>(), {
  canEdit: true,
})

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  'edit-config': [pixel: Pixel]
  'token-regenerated': [pixel: Pixel]
}>()

const { brandName } = useBranding()
const config = useRuntimeConfig()
const toast = useToast()
const pixelsApi = usePixels()

const tab = ref<'script' | 'links' | 'events'>('script')
const linksCount = ref(0)
const currentToken = computed(() => props.pixel?.public_token || '')
const snippet = computed(() =>
  props.pixel
    ? buildPixelSnippet(config.public.apiUrl as string, props.pixel.name, props.pixel.public_token || undefined, brandName.value)
    : '',
)
const manualSnippet = `// Lead manual (ex: botão ou formulário customizado)\ntf('send', 'Lead');\n\n// Compra com valor\ntf('send', 'Purchase', { value: 99.90, currency: 'BRL' });`

const miniCopied = ref('')
const showRegenConfirm = ref(false)
const regenerating = ref(false)

const onRegenerateToken = async () => {
  if (!props.pixel) return
  regenerating.value = true
  try {
    const r = await pixelsApi.regenerateToken(props.pixel.id)
    emit('token-regenerated', { ...props.pixel, public_token: r.public_token } as Pixel)
    toast.success('Novo token gerado', 'Atualize o snippet instalado no site.')
  } catch (e: unknown) {
    toast.error('Erro', errorMessage(e, 'Falha ao gerar novo token'))
  } finally {
    regenerating.value = false
    showRegenConfirm.value = false
  }
}

const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | null = null

const writeClipboard = async (t: string) => {
  if (navigator.clipboard) {
    await navigator.clipboard.writeText(t)
    return
  }
  const ta = document.createElement('textarea')
  ta.value = t
  document.body.appendChild(ta)
  ta.select()
  document.execCommand('copy')
  document.body.removeChild(ta)
}

const copy = async (text: string, tag: string) => {
  try {
    await writeClipboard(text)
    miniCopied.value = tag
    setTimeout(() => { miniCopied.value = '' }, 1500)
  } catch {
    toast.error('Erro', 'Não foi possível copiar')
  }
}

const copySnippet = async () => {
  if (!snippet.value) return
  try {
    await writeClipboard(snippet.value)
    copied.value = true
    toast.success('Copiado!', 'Script copiado para a área de transferência')
    if (copiedTimer) clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => { copied.value = false }, 2500)
  } catch {
    toast.error('Erro', 'Selecione e copie manualmente.')
  }
}

watch(() => props.modelValue, (open) => {
  if (!open) {
    copied.value = false
    miniCopied.value = ''
    tab.value = 'script'
  }
})
</script>

<style scoped>
.install-content { display: flex; flex-direction: column; }
.tabs { display: flex; border-bottom: 1px solid var(--border); margin-bottom: 1.5rem; overflow-x: auto; white-space: nowrap; }
.tab { padding: 0.625rem 1.125rem; background: transparent; border: none; color: var(--text-2); font-size: var(--fs-md); font-weight: var(--fw-medium); cursor: pointer; position: relative; display: inline-flex; align-items: center; gap: 0.375rem; flex-shrink: 0; }
.tab:hover { color: var(--text-1); }
.tab.active { color: var(--brand); }
.tab.active::after { content: ''; position: absolute; bottom: -1px; left: 0; right: 0; height: 2px; background: var(--brand); }
.tab-pane { display: flex; flex-direction: column; gap: 1.25rem; }
.section { display: flex; flex-direction: column; gap: 0.625rem; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; flex-wrap: wrap; }
.header-actions { display: flex; align-items: center; gap: 0.5rem; }
.section-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); display: inline-flex; align-items: center; gap: 0.5rem; }
.section-hint { font-size: var(--fs-base); color: var(--text-2); line-height: 1.5; }
.section-hint code, .copy-val, .code-block { font-family: var(--font-mono); }
.section-hint code { font-size: var(--fs-base); background: var(--surface-soft); padding: 0.0625rem 0.3125rem; border-radius: 4px; }
.badge-pill { font-size: var(--fs-xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; padding: 0.125rem 0.5rem; border-radius: 9999px; background: var(--brand-soft); color: var(--brand); }
.copy-row { display: flex; align-items: center; gap: 0.5rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 8px; padding: 0.5rem 0.75rem; }
.copy-val { font-size: var(--fs-base); color: var(--brand); flex: 1; word-break: break-all; letter-spacing: 0.04em; }
.copy-mini { display: inline-flex; align-items: center; justify-content: center; background: transparent; border: 1px solid var(--border); color: var(--text-2); padding: 0.3125rem 0.5rem; border-radius: 6px; cursor: pointer; transition: all 0.15s; }
.copy-mini:hover { border-color: var(--brand-border); color: var(--brand); }
.code-block { background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 10px; padding: 1rem; font-size: var(--fs-base); color: var(--text-1); overflow-x: auto; line-height: 1.6; white-space: pre; }
.code-block.small { font-size: var(--fs-sm); }
.domains-list { display: flex; flex-wrap: wrap; gap: 0.375rem; }
.domain-pill { font-size: var(--fs-sm); font-weight: var(--fw-medium); background: var(--success-soft); color: var(--success); padding: 0.1875rem 0.625rem; border-radius: 9999px; }
.hint-warn { font-size: var(--fs-base); color: var(--warning); }
</style>
