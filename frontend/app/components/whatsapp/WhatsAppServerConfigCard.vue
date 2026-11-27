<template>
  <section class="server-card">
    <div class="server-header">
      <div class="server-title-wrap">
        <div class="server-icon">
          <Server :size="18" />
        </div>
        <div>
          <div class="title-row">
            <h2 class="section-title">Servidor {{ providerMeta.label }}</h2>
            <BaseBadge :variant="activeServer?.has_api_key ? 'success' : 'warning'">
              {{ activeServer?.has_api_key ? 'Configurado' : 'Pendente' }}
            </BaseBadge>
            <BaseBadge v-if="activeServer?.has_api_key" variant="info">
              Global · Todas as empresas
            </BaseBadge>
          </div>
          <p class="section-desc">
            {{ providerMeta.desc }}
          </p>
        </div>
      </div>

      <div class="header-right">
        <div class="provider-tabs" role="tablist" aria-label="Provedor de Conexão Direta">
          <button
            type="button"
            role="tab"
            :aria-selected="activeProvider === 'evolution_go'"
            class="provider-tab"
            :class="{ active: activeProvider === 'evolution_go' }"
            @click="switchProvider('evolution_go')"
          >
            <span class="tab-dot" :class="{ ok: Boolean(evoServer?.has_api_key) }" />
            <span>Evolution Go</span>
          </button>
          <button
            type="button"
            role="tab"
            :aria-selected="activeProvider === 'waha'"
            class="provider-tab"
            :class="{ active: activeProvider === 'waha' }"
            @click="switchProvider('waha')"
          >
            <span class="tab-dot" :class="{ ok: Boolean(wahaServer?.has_api_key) }" />
            <span>WAHA API</span>
          </button>
        </div>

        <button
          v-if="activeServer && !expanded && canEditServer"
          type="button"
          class="toggle-edit-btn"
          @click="expanded = true"
        >
          Alterar credenciais
        </button>
      </div>
    </div>

    <div v-if="(!activeServer || expanded) && canEditServer" class="server-form">
      <div class="form-grid">
        <BaseInput
          v-model="form.url"
          :label="providerMeta.urlLabel"
          :placeholder="providerMeta.urlPlaceholder"
          required
        />
        <BaseInput
          v-model="form.apiKey"
          type="password"
          :label="providerMeta.keyLabel"
          :placeholder="activeServer?.has_api_key ? '•••••••••••••••• (deixe vazio para manter)' : providerMeta.keyPlaceholder"
          :required="!activeServer?.has_api_key"
        />
      </div>

      <div v-if="healthInfo" class="health-strip">
        <CheckCircle2 :size="15" class="health-ok" />
        <span>
          Conexão validada — {{ providerMeta.label }} <strong>v{{ healthInfo.version || 'OK' }}</strong>
          <template v-if="healthInfo.engine"> · Engine <strong>{{ healthInfo.engine }}</strong></template>
          ({{ healthInfo.instance_count }} {{ healthInfo.instance_count === 1 ? 'sessão' : 'sessões' }} no servidor)
        </span>
      </div>

      <div class="server-actions">
        <BaseButton v-if="activeServer && expanded" type="button" variant="ghost" size="sm" @click="expanded = false">
          Cancelar
        </BaseButton>
        <BaseButton type="button" variant="outline" size="sm" :loading="testing" @click="onTest">
          Testar conexão
        </BaseButton>
        <BaseButton type="button" variant="primary" size="sm" :loading="saving" @click="onSave">
          Salvar servidor
        </BaseButton>
      </div>
    </div>

    <div v-else-if="activeServer" class="server-summary-bar">
      <span class="summary-url">{{ activeServer.url }}</span>
      <BaseButton type="button" variant="ghost" size="sm" :loading="testing" @click="onTest">
        Verificar saúde
      </BaseButton>
    </div>
    <p v-else class="section-desc">
      Aguardando um administrador configurar o servidor global {{ providerMeta.label }}.
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { Server, CheckCircle2 } from 'lucide-vue-next'
import type { WhatsAppServer, WhatsAppServerHealth, WhatsAppServerProvider } from '#shared/types'
import { useAuthStore } from '~/stores/useAuthStore'
import { errorMessage } from '~/utils/errorMessage'

const props = defineProps<{
  server?: WhatsAppServer | null
  evolutionServer?: WhatsAppServer | null
  wahaServer?: WhatsAppServer | null
}>()

const emit = defineEmits<{ saved: [] }>()

const authStore = useAuthStore()
const canEditServer = computed(() => authStore.isAdmin)
const wa = useWhatsApp()
const toast = useToast()

const evoServer = computed(() => props.evolutionServer ?? props.server ?? null)
const activeProvider = ref<WhatsAppServerProvider>('evolution_go')
const expanded = ref(false)
const saving = ref(false)
const testing = ref(false)
const healthInfo = ref<WhatsAppServerHealth | null>(null)

const activeServer = computed(() => (activeProvider.value === 'waha' ? (props.wahaServer ?? null) : evoServer.value))

const providerMeta = computed(() => {
  if (activeProvider.value === 'waha') {
    return {
      label: 'WAHA API',
      desc: 'Servidor global WAHA compartilhado por todas as empresas (recomendado: engine GOWS). Cria sessões, injeta webhooks, baixa mídias e exibe QR Code automaticamente.',
      urlLabel: 'URL da WAHA API',
      urlPlaceholder: 'https://waha.seudominio.com.br',
      keyLabel: 'WAHA API Key (X-Api-Key)',
      keyPlaceholder: 'Sua WAHA_API_KEY',
    }
  }
  return {
    label: 'Evolution Go',
    desc: 'Configuração global compartilhada por todas as empresas deste painel. Configure uma única vez para conectar WhatsApp via QR Code em qualquer cliente.',
    urlLabel: 'URL da Evolution Go',
    urlPlaceholder: 'https://evo.seudominio.com.br',
    keyLabel: 'Global API Key',
    keyPlaceholder: 'Sua GLOBAL_API_KEY',
  }
})

const form = reactive({ name: 'Evolution Go', url: '', apiKey: '' })

const syncForm = () => {
  const srv = activeServer.value
  form.name = srv?.name || providerMeta.value.label
  form.url = srv?.url || ''
  form.apiKey = ''
  healthInfo.value = null
}

watch([activeProvider, evoServer, () => props.wahaServer], syncForm, { immediate: true })

const switchProvider = (p: WhatsAppServerProvider) => {
  activeProvider.value = p
  expanded.value = false
}

const onTest = async () => {
  testing.value = true
  healthInfo.value = null
  try {
    const res = await wa.testServer({
      provider: activeProvider.value,
      url: form.url.trim() || undefined,
      api_key: form.apiKey.trim() || undefined,
    })
    healthInfo.value = res
    toast.success('Servidor online', `${providerMeta.value.label} v${res.version || 'OK'} respondeu com sucesso.`)
  } catch (e: unknown) {
    toast.error('Falha na conexão', errorMessage(e, `Verifique a URL e a API Key da ${providerMeta.value.label}.`))
  } finally {
    testing.value = false
  }
}

const onSave = async () => {
  if (!form.url.trim()) {
    toast.error('URL obrigatória', `Informe o endereço http:// ou https:// da ${providerMeta.value.label}.`)
    return
  }
  saving.value = true
  try {
    await wa.saveServer({
      provider: activeProvider.value,
      name: form.name,
      url: form.url.trim(),
      api_key: form.apiKey.trim(),
    })
    form.apiKey = ''
    expanded.value = false
    toast.success('Servidor salvo', `Credenciais da ${providerMeta.value.label} criptografadas e salvas.`)
    emit('saved')
  } catch (e: unknown) {
    toast.error('Erro ao salvar', errorMessage(e, 'Não foi possível salvar o servidor.'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.server-card { background: var(--surface-2); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem; }
.server-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; flex-wrap: wrap; }
.server-title-wrap { display: flex; align-items: flex-start; gap: 0.875rem; flex: 1; min-width: 240px; }
.server-icon { width: 38px; height: 38px; border-radius: 9px; background: var(--brand-soft); color: var(--brand); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.title-row { display: flex; align-items: center; gap: 0.625rem; flex-wrap: wrap; }
.section-title { font-family: var(--font-heading); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-desc { font-size: var(--fs-base); color: var(--text-2); margin: 0.25rem 0 0; line-height: 1.5; }
.header-right { display: flex; align-items: center; gap: 0.625rem; flex-wrap: wrap; }
.provider-tabs { display: inline-flex; padding: 0.2rem; border-radius: 9px; background: var(--surface-deep); border: 1px solid var(--border-soft); gap: 0.2rem; }
.provider-tab { display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.35rem 0.7rem; border: none; border-radius: 7px; background: transparent; color: var(--text-2); font-size: var(--fs-xs); font-weight: var(--fw-medium); cursor: pointer; transition: all 0.15s ease; }
.provider-tab:hover { color: var(--text-1); }
.provider-tab.active { background: var(--surface-2); color: var(--text-strong); font-weight: var(--fw-semibold); box-shadow: inset 0 0 0 1px var(--border); }
.tab-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--text-3); opacity: 0.45; }
.tab-dot.ok { background: var(--success); opacity: 1; }
.toggle-edit-btn { background: transparent; border: 1px solid var(--border); color: var(--text-2); padding: 0.375rem 0.75rem; border-radius: 7px; font-size: var(--fs-sm); font-weight: var(--fw-medium); cursor: pointer; }
.toggle-edit-btn:hover { color: var(--text-1); border-color: var(--brand-border); }
.server-form { display: flex; flex-direction: column; gap: 0.875rem; padding-top: 0.5rem; border-top: 1px solid var(--border-faint); }
.form-grid { display: grid; grid-template-columns: 1.2fr 1fr; gap: 1rem; }
.health-strip { display: flex; align-items: center; gap: 0.5rem; padding: 0.625rem 0.875rem; background: var(--success-soft); color: var(--success); border-radius: 8px; font-size: var(--fs-sm); }
.health-ok { flex-shrink: 0; }
.server-actions { display: flex; justify-content: flex-end; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.server-summary-bar { display: flex; justify-content: space-between; align-items: center; gap: 0.75rem; padding: 0.625rem 0.875rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 8px; }
.summary-url { font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--text-2); overflow: hidden; text-overflow: ellipsis; }
@media (max-width: 720px) { .form-grid { grid-template-columns: 1fr; } }
</style>
