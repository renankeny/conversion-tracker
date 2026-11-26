<template>
  <BaseModal
    v-model="modalOpen"
    title="Atualização disponível"
    description="Uma nova versão da plataforma já está pronta para instalação."
    size="sm"
  >
    <div class="update-body">
      <div class="version-strip">
        <div class="version-col">
          <span class="version-label">Versão atual</span>
          <code class="version-code">{{ currentLabel }}</code>
        </div>
        <ArrowRight :size="15" class="version-arrow" />
        <div class="version-col">
          <span class="version-label">Nova versão</span>
          <code class="version-code version-code-new">{{ latestLabel }}</code>
        </div>
        <div v-if="publishedLabel" class="version-date">
          <span class="version-label">Publicada em</span>
          <span class="version-date-val">{{ publishedLabel }}</span>
        </div>
      </div>

      <div v-if="isUpdatingNow" class="update-progress">
        <RefreshCw :size="15" class="animate-spin progress-icon" />
        <div class="progress-text">
          <p class="progress-title">Atualização em andamento</p>
          <p class="progress-desc">
            {{ status?.update_message || 'Os serviços serão reiniciados automaticamente em alguns instantes. Esta página recarregará assim que concluir.' }}
          </p>
        </div>
      </div>

      <p v-else-if="status?.can_auto_update" class="update-info">
        A atualização preserva seus dados, pixels, conversas e webhooks. Durante o processo o painel pode ficar indisponível por cerca de 1 a 2 minutos.
      </p>

      <div v-else class="manual-box">
        <p class="manual-title">Atualização manual na VPS</p>
        <p class="manual-desc">
          Como esta instalação não utiliza atualização automática vinculada, execute o redeploy dos containers no gerenciador da sua VPS para aplicar a versão <strong>{{ latestLabel }}</strong>.
        </p>
      </div>

      <div v-if="errorMsg" class="update-error">
        {{ errorMsg }}
      </div>
    </div>

    <template #footer>
      <div class="update-footer">
        <BaseButton type="button" variant="ghost" size="sm" @click="closeModal">
          {{ isUpdatingNow ? 'Minimizar' : 'Depois' }}
        </BaseButton>

        <BaseButton
          v-if="status?.can_auto_update"
          type="button"
          variant="primary"
          size="sm"
          :loading="updating || isUpdatingNow"
          :disabled="updating || isUpdatingNow"
          @click="triggerUpdate"
        >
          {{ isUpdatingNow ? 'Atualizando...' : 'Atualizar agora' }}
        </BaseButton>

        <BaseButton
          v-else
          type="button"
          variant="outline"
          size="sm"
          :loading="checking"
          @click="onRecheck"
        >
          Verificar novamente
        </BaseButton>
      </div>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, RefreshCw } from 'lucide-vue-next'
import { useSystemUpdate } from '~/composables/useSystemUpdate'
import { useToast } from '~/composables/useToast'

const {
  status,
  checking,
  updating,
  modalOpen,
  errorMsg,
  isUpdatingNow,
  checkForUpdates,
  triggerUpdate,
  closeModal,
} = useSystemUpdate()
const { add: addToast } = useToast()

const currentLabel = computed(() => status.value?.current_version || '—')
const latestLabel = computed(() => status.value?.latest_version || '—')

const publishedLabel = computed(() => {
  const raw = status.value?.published_at
  if (!raw) return ''
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) return ''
  return new Intl.DateTimeFormat('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(d)
})

const onRecheck = async () => {
  await checkForUpdates(true)
  if (!status.value?.update_available) {
    closeModal()
    addToast({
      type: 'success',
      title: 'Sistema atualizado',
      message: 'Sua instalação já está rodando na versão mais recente.',
    })
  }
}
</script>

<style scoped>
.update-body { display: flex; flex-direction: column; gap: 1rem; }
.version-strip {
  display: flex; align-items: center; flex-wrap: wrap; gap: 0.75rem;
  padding: 0.85rem 1rem; border-radius: 10px; background: var(--surface-deep); border: 1px solid var(--border-soft);
}
.version-col { display: flex; flex-direction: column; gap: 0.2rem; }
.version-label { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-3); }
.version-code { font-family: var(--font-mono); font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); }
.version-code-new { color: var(--brand); }
.version-arrow { color: var(--text-3); flex-shrink: 0; }
.version-date { margin-left: auto; display: flex; flex-direction: column; align-items: flex-end; gap: 0.2rem; }
.version-date-val { font-size: var(--fs-sm); font-weight: var(--fw-medium); color: var(--text-2); }
.update-info { font-size: var(--fs-sm); color: var(--text-2); line-height: 1.55; margin: 0; }
.update-progress {
  display: flex; align-items: flex-start; gap: 0.75rem; padding: 0.85rem 1rem;
  border-radius: 10px; background: var(--brand-soft); border: 1px solid var(--brand-border);
}
.progress-icon { color: var(--brand); flex-shrink: 0; margin-top: 0.1rem; }
.progress-text { display: flex; flex-direction: column; gap: 0.2rem; }
.progress-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.progress-desc { font-size: var(--fs-xs); color: var(--text-2); line-height: 1.45; margin: 0; }
.manual-box { padding: 0.85rem 1rem; border-radius: 10px; background: var(--surface-soft); border: 1px solid var(--border-soft); }
.manual-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0 0 0.3rem; }
.manual-desc { font-size: var(--fs-xs); color: var(--text-2); line-height: 1.5; margin: 0; }
.manual-desc strong { color: var(--text-1); font-weight: var(--fw-semibold); }
.update-error { padding: 0.65rem 0.85rem; border-radius: 8px; font-size: var(--fs-xs); background: var(--danger-soft); border: 1px solid var(--danger); color: var(--danger); }
.update-footer { display: flex; align-items: center; justify-content: flex-end; gap: 0.625rem; width: 100%; }
</style>
