<template>
  <section class="server-card">
    <div class="server-header">
      <div class="server-title-wrap">
        <div class="server-icon">
          <Server :size="18" />
        </div>
        <div>
          <div class="title-row">
            <h2 class="section-title">Servidor Evolution Go</h2>
            <BaseBadge :variant="server?.has_api_key ? 'success' : 'warning'">
              {{ server?.has_api_key ? 'Configurado' : 'Pendente' }}
            </BaseBadge>
          </div>
          <p class="section-desc">
            Informe a URL base e a Global API Key da sua instância Evolution Go para provisionar conexões e receber webhooks CTWA.
          </p>
        </div>
      </div>

      <button
        v-if="server && !expanded"
        type="button"
        class="toggle-edit-btn"
        @click="expanded = true"
      >
        Alterar credenciais
      </button>
    </div>

    <div v-if="!server || expanded" class="server-form">
      <div class="form-grid">
        <BaseInput
          v-model="form.url"
          label="URL da Evolution Go"
          placeholder="https://evo.seudominio.com.br"
          required
        />

        <BaseInput
          v-model="form.apiKey"
          type="password"
          label="Global API Key"
          :placeholder="server?.has_api_key ? '•••••••••••••••• (deixe vazio para manter)' : 'Sua GLOBAL_API_KEY'"
          :required="!server?.has_api_key"
        />
      </div>

      <div v-if="healthInfo" class="health-strip">
        <CheckCircle2 :size="15" class="health-ok" />
        <span>
          Conexão validada — Evolution Go <strong>v{{ healthInfo.version || '0.7' }}</strong>
          ({{ healthInfo.instance_count }} {{ healthInfo.instance_count === 1 ? 'instância' : 'instâncias' }} no servidor)
        </span>
      </div>

      <div class="server-actions">
        <BaseButton
          v-if="server && expanded"
          type="button"
          variant="ghost"
          size="sm"
          @click="expanded = false"
        >
          Cancelar
        </BaseButton>
        <BaseButton
          type="button"
          variant="outline"
          size="sm"
          :loading="testing"
          @click="onTest"
        >
          Testar conexão
        </BaseButton>
        <BaseButton
          type="button"
          variant="primary"
          size="sm"
          :loading="saving"
          @click="onSave"
        >
          Salvar servidor
        </BaseButton>
      </div>
    </div>

    <div v-else class="server-summary-bar">
      <span class="summary-url">{{ server?.url }}</span>
      <BaseButton type="button" variant="ghost" size="sm" :loading="testing" @click="onTest">
        Verificar saúde
      </BaseButton>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Server, CheckCircle2 } from 'lucide-vue-next'
import type { WhatsAppServer, WhatsAppServerHealth } from '#shared/types'

const props = defineProps<{
  server: WhatsAppServer | null
}>()

const emit = defineEmits<{
  saved: []
}>()

const wa = useWhatsApp()
const toast = useToast()

const expanded = ref(false)
const saving = ref(false)
const testing = ref(false)
const healthInfo = ref<WhatsAppServerHealth | null>(null)

const form = reactive({
  name: 'Evolution Go',
  url: '',
  apiKey: '',
})

watch(
  () => props.server,
  (srv) => {
    if (srv) {
      form.name = srv.name || 'Evolution Go'
      form.url = srv.url || ''
      form.apiKey = ''
    }
  },
  { immediate: true },
)

const onTest = async () => {
  testing.value = true
  healthInfo.value = null
  try {
    const res = await wa.testServer({
      url: form.url.trim() || undefined,
      api_key: form.apiKey.trim() || undefined,
    })
    healthInfo.value = res
    toast.success('Servidor online', `Evolution Go v${res.version || '0.7'} respondeu com sucesso.`)
  } catch (e: any) {
    toast.error('Falha na conexão', e.message || 'Verifique a URL e a Global API Key.')
  } finally {
    testing.value = false
  }
}

const onSave = async () => {
  if (!form.url.trim()) {
    toast.error('URL obrigatória', 'Informe o endereço http:// ou https:// da Evolution Go.')
    return
  }
  saving.value = true
  try {
    await wa.saveServer({
      name: form.name,
      url: form.url.trim(),
      api_key: form.apiKey.trim(),
    })
    form.apiKey = ''
    expanded.value = false
    toast.success('Servidor salvo', 'Credenciais da Evolution Go criptografadas e salvas.')
    emit('saved')
  } catch (e: any) {
    toast.error('Erro ao salvar', e.message || 'Não foi possível salvar o servidor.')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.server-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.server-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: wrap;
}
.server-title-wrap { display: flex; align-items: flex-start; gap: 0.875rem; flex: 1; min-width: 240px; }
.server-icon {
  width: 38px; height: 38px; border-radius: 9px;
  background: var(--brand-soft); color: var(--brand);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.title-row { display: flex; align-items: center; gap: 0.625rem; flex-wrap: wrap; }
.section-title { font-family: var(--font-heading); font-size: var(--fs-xl); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.section-desc { font-size: var(--fs-base); color: var(--text-2); margin: 0.25rem 0 0; line-height: 1.5; }
.toggle-edit-btn {
  background: transparent; border: 1px solid var(--border); color: var(--text-2);
  padding: 0.375rem 0.75rem; border-radius: 7px; font-size: var(--fs-sm); font-weight: var(--fw-medium); cursor: pointer;
}
.toggle-edit-btn:hover { color: var(--text-1); border-color: var(--brand-border); }
.server-form { display: flex; flex-direction: column; gap: 0.875rem; padding-top: 0.5rem; border-top: 1px solid var(--border-faint); }
.form-grid { display: grid; grid-template-columns: 1.2fr 1fr; gap: 1rem; }
.health-strip {
  display: flex; align-items: center; gap: 0.5rem; padding: 0.625rem 0.875rem;
  background: var(--success-soft); color: var(--success); border-radius: 8px; font-size: var(--fs-sm);
}
.health-ok { flex-shrink: 0; }
.server-actions { display: flex; justify-content: flex-end; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.server-summary-bar {
  display: flex; justify-content: space-between; align-items: center; gap: 0.75rem;
  padding: 0.625rem 0.875rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 8px;
}
.summary-url { font-family: var(--font-mono); font-size: var(--fs-sm); color: var(--text-2); overflow: hidden; text-overflow: ellipsis; }
@media (max-width: 720px) { .form-grid { grid-template-columns: 1fr; } }
</style>

