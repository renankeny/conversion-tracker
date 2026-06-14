<template>
  <div class="mappings-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">URL Mappings</h1>
        <p class="page-subtitle">
          Mapeie URLs externas (campanhas, encurtadores) para URLs canônicas.
          O backend reescreve antes de enviar pra Meta CAPI, padronizando atribuição.
        </p>
      </div>
      <BaseButton variant="primary" @click="openCreateModal">
        + Novo mapeamento
      </BaseButton>
    </div>

    <div v-if="loading && !mappings.length" class="state-row">
      <div class="spinner" />
    </div>

    <div v-else-if="!mappings.length" class="empty-state">
      <p>Nenhum mapeamento cadastrado.</p>
      <BaseButton variant="primary" @click="openCreateModal">Criar primeiro mapeamento</BaseButton>
    </div>

    <div v-else class="mappings-list">
      <div v-for="m in mappings" :key="m.id" class="mapping-row">
        <div class="mapping-flow">
          <div class="mapping-side">
            <span class="side-label">Source</span>
            <code class="side-url">{{ m.source }}</code>
          </div>
          <svg class="arrow" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M10.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L12.586 11H5a1 1 0 110-2h7.586l-2.293-2.293a1 1 0 010-1.414z" clip-rule="evenodd" />
          </svg>
          <div class="mapping-side">
            <span class="side-label">Target</span>
            <code class="side-url">{{ m.target }}</code>
          </div>
        </div>

        <div class="mapping-actions">
          <BaseBadge :variant="m.enabled ? 'success' : 'default'">
            {{ m.enabled ? 'Ativo' : 'Inativo' }}
          </BaseBadge>
          <BaseButton variant="ghost" size="sm" @click="openEditModal(m)">Editar</BaseButton>
          <BaseButton variant="ghost" size="sm" @click="askDelete(m)">
            <span class="text-danger">Excluir</span>
          </BaseButton>
        </div>
      </div>
    </div>

    <!-- Modal create/edit -->
    <BaseModal
      v-model="formModalOpen"
      :title="editing ? 'Editar mapeamento' : 'Novo mapeamento'"
      size="md"
    >
      <div class="form">
        <BaseInput
          v-model="form.source"
          label="URL de origem (source)"
          placeholder="https://encurtador.com/abc123"
          hint="A URL que chega via UTM/campanha"
          required
        />
        <BaseInput
          v-model="form.target"
          label="URL canônica (target)"
          placeholder="https://meusite.com.br/produto-x"
          hint="Para qual URL o source será reescrito antes de enviar pra Meta"
          required
        />
        <label class="enable-toggle">
          <input v-model="form.enabled" type="checkbox" />
          <span>Ativo</span>
        </label>
      </div>
      <template #footer>
        <BaseButton variant="ghost" @click="formModalOpen = false">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="saving" @click="onSave">
          {{ editing ? 'Salvar' : 'Criar' }}
        </BaseButton>
      </template>
    </BaseModal>

    <ConfirmModal
      v-model="deleteOpen"
      title="Excluir mapeamento?"
      :message="deletingItem ? `Source: ${deletingItem.source}` : ''"
      confirm-label="Excluir"
      variant="danger"
      :loading="deleting"
      @confirm="onConfirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import type { UrlMapping } from '#shared/types'

definePageMeta({
  layout: 'app',
  middleware: 'auth',
})

useHead({ title: 'URL Mappings - TrackFind' })

const urlMappings = useUrlMappings()
const toast = useToast()

const mappings = ref<UrlMapping[]>([])
const loading = ref(false)

const formModalOpen = ref(false)
const editing = ref<UrlMapping | null>(null)
const saving = ref(false)
const form = reactive({
  source: '',
  target: '',
  enabled: true,
})

const deleteOpen = ref(false)
const deletingItem = ref<UrlMapping | null>(null)
const deleting = ref(false)

const fetchMappings = async () => {
  loading.value = true
  try {
    const res = await urlMappings.list()
    mappings.value = res.data || []
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao carregar mapeamentos')
  } finally {
    loading.value = false
  }
}

const openCreateModal = () => {
  editing.value = null
  form.source = ''
  form.target = ''
  form.enabled = true
  formModalOpen.value = true
}

const openEditModal = (m: UrlMapping) => {
  editing.value = m
  form.source = m.source
  form.target = m.target
  form.enabled = m.enabled
  formModalOpen.value = true
}

const onSave = async () => {
  if (!form.source.trim() || !form.target.trim()) {
    toast.error('Erro', 'Preencha source e target')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      const updated = await urlMappings.update(editing.value.id, {
        source: form.source.trim(),
        target: form.target.trim(),
        enabled: form.enabled,
      })
      const idx = mappings.value.findIndex((m) => m.id === updated.id)
      if (idx !== -1) mappings.value[idx] = updated
      toast.success('Atualizado')
    } else {
      const created = await urlMappings.create({
        source: form.source.trim(),
        target: form.target.trim(),
        enabled: form.enabled,
      })
      mappings.value = [created, ...mappings.value]
      toast.success('Criado')
    }
    formModalOpen.value = false
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao salvar')
  } finally {
    saving.value = false
  }
}

const askDelete = (m: UrlMapping) => {
  deletingItem.value = m
  deleteOpen.value = true
}

const onConfirmDelete = async () => {
  if (!deletingItem.value) return
  deleting.value = true
  try {
    await urlMappings.remove(deletingItem.value.id)
    mappings.value = mappings.value.filter((m) => m.id !== deletingItem.value!.id)
    toast.success('Removido')
    deleteOpen.value = false
    deletingItem.value = null
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao remover')
  } finally {
    deleting.value = false
  }
}

onMounted(fetchMappings)
</script>

<style scoped>
.mappings-page {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: wrap;
}

.page-header > div:first-child {
  flex: 1;
  min-width: 280px;
}

.page-title {
  font-family: 'Instrument Sans', sans-serif;
  font-size: 1.875rem;
  font-weight: 700;
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.page-subtitle {
  color: var(--text-2);
  font-size: 0.9375rem;
  max-width: 640px;
}

.state-row,
.empty-state {
  background: var(--surface-2);
  border: 1px dashed var(--border);
  border-radius: 12px;
  padding: 3rem 1rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  color: var(--text-3);
}

.spinner {
  width: 1.5rem;
  height: 1.5rem;
  border: 3px solid var(--border);
  border-top-color: var(--brand);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.mappings-list {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}

.mapping-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.25rem;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 10px;
  flex-wrap: wrap;
}

.mapping-flow {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex: 1;
  min-width: 280px;
}

.mapping-side {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
}

.side-label {
  font-size: 0.6875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-3);
  font-weight: 600;
}

.side-url {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8125rem;
  background: var(--surface-soft);
  padding: 0.125rem 0.5rem;
  border-radius: 6px;
  word-break: break-all;
  color: var(--text-1);
}

.arrow {
  width: 1.25rem;
  height: 1.25rem;
  color: var(--brand);
  flex-shrink: 0;
}

.mapping-actions {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  flex-shrink: 0;
}

.text-danger {
  color: var(--danger);
}

.form {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

.enable-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  cursor: pointer;
  user-select: none;
}

.enable-toggle input[type='checkbox'] {
  width: 1rem;
  height: 1rem;
  accent-color: var(--brand);
}
</style>
