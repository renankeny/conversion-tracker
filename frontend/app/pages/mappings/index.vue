<template>
  <div class="mappings-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">URL Mappings</h1>
        <p class="page-subtitle">
          Reescrita de URLs antes do envio pra Meta CAPI. Use para anunciar uma página black
          (vendas) e mostrar ao Facebook uma página white (institucional/segura), protegendo
          o pixel contra reprovação e limitando entrega.
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
        <div class="info-box">
          <div class="info-box-title">Como funciona</div>
          <p class="info-box-text">
            Toda vez que um evento (PageView, Purchase, etc) for enviado pra Meta CAPI,
            o sistema procura a URL real (de onde o usuário veio) na tabela abaixo. Se houver
            match, a URL é trocada antes do envio. O Meta passa a "ver" a página white
            no lugar da black.
          </p>
          <p class="info-box-text">
            <strong>3 níveis de match</strong>, do mais específico pro mais genérico:
          </p>
          <ul class="info-list">
            <li>
              <code>https://black.com/oferta</code> → substitui a URL inteira pelo target
            </li>
            <li>
              <code>black.com/oferta</code> → idem, ignorando scheme
            </li>
            <li>
              <code>black.com</code> → troca só o domínio, preservando path/query/fragment original
            </li>
          </ul>
        </div>

        <BaseInput
          v-model="form.source"
          label="Source — URL black (que o usuário acessa de verdade)"
          placeholder="https://meublack.com.br/oferta-emagrecedor"
          hint="A URL real onde o tracker dispara. Pode ser domínio inteiro (ex.: meublack.com.br) ou URL específica."
          required
        />
        <BaseInput
          v-model="form.target"
          label="Target — URL white (que o Meta vai ver)"
          placeholder="https://meuwhite.com.br/produto"
          hint="A URL inofensiva que vai como event_source_url pro Facebook. Costuma ser uma página institucional/about/blog do mesmo nicho."
          required
        />

        <details class="examples">
          <summary>Ver exemplos práticos</summary>
          <div class="example-block">
            <div class="example-label">Domínio inteiro (mais comum)</div>
            <div class="example-row">
              <span class="example-tag source">source</span>
              <code>blackpage.com.br</code>
            </div>
            <div class="example-row">
              <span class="example-tag target">target</span>
              <code>https://whitepage.com.br</code>
            </div>
            <div class="example-note">
              Qualquer URL de blackpage.com.br vira whitepage.com.br mantendo o path original.
            </div>
          </div>

          <div class="example-block">
            <div class="example-label">Override de página específica</div>
            <div class="example-row">
              <span class="example-tag source">source</span>
              <code>https://blackpage.com.br/oferta-vip</code>
            </div>
            <div class="example-row">
              <span class="example-tag target">target</span>
              <code>https://whitepage.com.br/blog/saude</code>
            </div>
            <div class="example-note">
              Substitui inteira. Útil quando o domínio já tem regra mais ampla mas você quer
              direcionar uma URL específica pra outro destino white.
            </div>
          </div>
        </details>

        <label class="enable-toggle">
          <input v-model="form.enabled" type="checkbox" />
          <span>Ativo (regra começa a valer imediatamente após salvar)</span>
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

.info-box {
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  border-radius: 8px;
  padding: 0.875rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.info-box-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--brand);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.info-box-text {
  font-size: 0.8125rem;
  line-height: 1.5;
  color: var(--text-1);
  margin: 0;
}

.info-list {
  margin: 0;
  padding-left: 1.125rem;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  font-size: 0.8125rem;
  color: var(--text-1);
  line-height: 1.5;
}

.info-list code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  background: var(--surface-soft);
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  color: var(--text-strong);
}

.examples {
  background: var(--surface-soft);
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  padding: 0.625rem 0.875rem;
  font-size: 0.8125rem;
}

.examples summary {
  cursor: pointer;
  font-weight: 500;
  color: var(--text-1);
  user-select: none;
  padding: 0.125rem 0;
}

.examples summary:hover {
  color: var(--brand);
}

.example-block {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border-faint);
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.example-block:first-of-type {
  border-top: none;
  padding-top: 0;
}

.example-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-strong);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.example-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.example-tag {
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  flex-shrink: 0;
}

.example-tag.source {
  background: var(--danger-soft);
  color: var(--danger);
}

.example-tag.target {
  background: var(--success-soft);
  color: var(--success);
}

.example-row code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  background: var(--surface-2);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  color: var(--text-1);
  word-break: break-all;
}

.example-note {
  font-size: 0.75rem;
  color: var(--text-2);
  line-height: 1.4;
  font-style: italic;
}

.enable-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  cursor: pointer;
  user-select: none;
  color: var(--text-1);
}

.enable-toggle input[type='checkbox'] {
  width: 1rem;
  height: 1rem;
  accent-color: var(--brand);
}
</style>
