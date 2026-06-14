<template>
  <BaseModal
    :model-value="modelValue"
    :title="user ? userTitle : ''"
    :description="user ? user.external_id : ''"
    size="lg"
    @update:model-value="handleClose"
  >
    <div v-if="user" class="user-details">
      <section class="section">
        <h4 class="section-title">Identificação</h4>
        <div class="grid">
          <Field label="External ID" :value="user.external_id" mono :readonly="true" />
          <EditField
            label="ID WhatsApp"
            v-model="form.id_wpp"
            :editing="editing"
            :original="user.id_wpp"
            mono
          />
          <EditField
            label="Pixel"
            v-model="form.pixel_name"
            :editing="editing"
            :original="user.pixel_name"
          />
          <EditField
            label="Primeiro nome"
            v-model="form.first_name"
            :editing="editing"
            :original="user.first_name"
          />
          <EditField
            label="Sobrenome"
            v-model="form.last_name"
            :editing="editing"
            :original="user.last_name"
          />
          <EditField
            label="E-mail"
            v-model="form.email"
            :editing="editing"
            :original="user.email"
            mono
          />
          <EditField
            label="Telefone"
            v-model="form.phone"
            :editing="editing"
            :original="user.phone"
            mono
          />
        </div>
      </section>

      <section class="section">
        <h4 class="section-title">Cookies do Meta</h4>
        <div class="grid">
          <EditField
            label="FBP"
            v-model="form.fbp"
            :editing="editing"
            :original="user.fbp"
            mono
            small
          />
          <EditField
            label="FBC"
            v-model="form.fbc"
            :editing="editing"
            :original="user.fbc"
            mono
            small
          />
          <EditField
            label="GCLID"
            v-model="form.gclid"
            :editing="editing"
            :original="user.gclid"
            mono
            small
          />
        </div>
      </section>

      <section class="section">
        <h4 class="section-title">Geolocalização</h4>
        <div class="grid">
          <EditField
            label="País"
            v-model="form.country"
            :editing="editing"
            :original="user.country"
          />
          <EditField
            label="Estado"
            v-model="form.state"
            :editing="editing"
            :original="user.state"
          />
          <EditField
            label="Cidade"
            v-model="form.city"
            :editing="editing"
            :original="user.city"
          />
          <EditField
            label="CEP"
            v-model="form.postal_code"
            :editing="editing"
            :original="user.postal_code"
          />
          <EditField
            label="IP"
            v-model="form.client_ip"
            :editing="editing"
            :original="user.client_ip"
            mono
          />
        </div>
      </section>

      <section v-if="editing || user.client_user_agent" class="section">
        <h4 class="section-title">User-Agent</h4>
        <textarea
          v-if="editing"
          v-model="form.client_user_agent"
          class="ua-edit"
          rows="2"
        />
        <code v-else class="ua">{{ user.client_user_agent }}</code>
      </section>

      <section class="section">
        <h4 class="section-title">UTM da primeira visita</h4>
        <div class="grid">
          <EditField label="Source" v-model="form.first_utm_source" :editing="editing" :original="user.first_utm_source" mono small />
          <EditField label="Medium" v-model="form.first_utm_medium" :editing="editing" :original="user.first_utm_medium" mono small />
          <EditField label="Campaign" v-model="form.first_utm_campaign" :editing="editing" :original="user.first_utm_campaign" mono small />
          <EditField label="Content" v-model="form.first_utm_content" :editing="editing" :original="user.first_utm_content" mono small />
          <EditField label="Term" v-model="form.first_utm_term" :editing="editing" :original="user.first_utm_term" mono small />
        </div>
      </section>

      <section class="section">
        <h4 class="section-title">UTM da última visita</h4>
        <div class="grid">
          <EditField label="Source" v-model="form.last_utm_source" :editing="editing" :original="user.last_utm_source" mono small />
          <EditField label="Medium" v-model="form.last_utm_medium" :editing="editing" :original="user.last_utm_medium" mono small />
          <EditField label="Campaign" v-model="form.last_utm_campaign" :editing="editing" :original="user.last_utm_campaign" mono small />
          <EditField label="Content" v-model="form.last_utm_content" :editing="editing" :original="user.last_utm_content" mono small />
          <EditField label="Term" v-model="form.last_utm_term" :editing="editing" :original="user.last_utm_term" mono small />
        </div>
      </section>

      <section class="section">
        <h4 class="section-title">Última URL</h4>
        <input
          v-if="editing"
          v-model="form.url"
          type="url"
          class="url-edit"
          placeholder="https://..."
        />
        <a
          v-else-if="user.url"
          :href="user.url"
          target="_blank"
          rel="noopener"
          class="url-link"
        >{{ user.url }}</a>
        <span v-else class="muted">—</span>
      </section>

      <section class="section">
        <h4 class="section-title">Timestamps</h4>
        <div class="grid">
          <Field label="Criado em" :value="formatDate(user.created_at)" :readonly="true" />
          <Field label="Atualizado em" :value="formatDate(user.updated_at)" :readonly="true" />
        </div>
      </section>
    </div>

    <template #footer>
      <template v-if="editing">
        <BaseButton variant="ghost" @click="cancelEdit">Cancelar</BaseButton>
        <BaseButton variant="primary" :loading="saving" @click="onSave">
          Salvar alterações
        </BaseButton>
      </template>
      <template v-else>
        <BaseButton variant="ghost" @click="handleClose(false)">Fechar</BaseButton>
        <BaseButton variant="primary" @click="startEdit">Editar</BaseButton>
      </template>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { TrackedUser, TrackedUserUpdateInput } from '#shared/types'

interface Props {
  modelValue: boolean
  user: TrackedUser | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [user: TrackedUser]
}>()

const trackedUsers = useTrackedUsers()
const toast = useToast()

const editing = ref(false)
const saving = ref(false)

// form contém uma cópia dos campos editáveis. Inicializa pela função
// resetForm sempre que o user prop mudar ou ao começar/cancelar edição.
const form = reactive<Required<TrackedUserUpdateInput>>({
  id_wpp: '',
  pixel_name: '',
  fbp: '',
  fbc: '',
  gclid: '',
  client_ip: '',
  client_user_agent: '',
  country: '',
  state: '',
  city: '',
  postal_code: '',
  first_name: '',
  last_name: '',
  email: '',
  phone: '',
  url: '',
  first_utm_source: '',
  first_utm_medium: '',
  first_utm_campaign: '',
  first_utm_content: '',
  first_utm_term: '',
  last_utm_source: '',
  last_utm_medium: '',
  last_utm_campaign: '',
  last_utm_content: '',
  last_utm_term: '',
})

const resetForm = (u: TrackedUser | null) => {
  if (!u) return
  const fields: (keyof TrackedUserUpdateInput)[] = [
    'id_wpp', 'pixel_name', 'fbp', 'fbc', 'gclid',
    'client_ip', 'client_user_agent',
    'country', 'state', 'city', 'postal_code',
    'first_name', 'last_name', 'email', 'phone', 'url',
    'first_utm_source', 'first_utm_medium', 'first_utm_campaign',
    'first_utm_content', 'first_utm_term',
    'last_utm_source', 'last_utm_medium', 'last_utm_campaign',
    'last_utm_content', 'last_utm_term',
  ]
  for (const f of fields) {
    ;(form as any)[f] = (u as any)[f] ?? ''
  }
}

watch(
  () => props.user,
  (u) => {
    editing.value = false
    resetForm(u)
  },
  { immediate: true }
)

const startEdit = () => {
  resetForm(props.user)
  editing.value = true
}

const cancelEdit = () => {
  resetForm(props.user)
  editing.value = false
}

const onSave = async () => {
  if (!props.user) return
  saving.value = true
  try {
    const updated = await trackedUsers.update(props.user.id, { ...form })
    toast.success('Usuário atualizado')
    editing.value = false
    emit('saved', updated)
  } catch (error: any) {
    toast.error('Erro', error.message || 'Falha ao salvar')
  } finally {
    saving.value = false
  }
}

const handleClose = (v: boolean) => {
  if (!v && editing.value) {
    cancelEdit()
  }
  emit('update:modelValue', v)
}

const fullName = (u: TrackedUser): string => {
  const parts = [u.first_name, u.last_name].filter(Boolean)
  return parts.join(' ').trim() || '—'
}

const userTitle = computed(() => {
  if (!props.user) return ''
  const name = fullName(props.user)
  if (name !== '—') return name
  return props.user.email || props.user.phone || 'Usuário rastreado'
})

const formatDate = (iso: string): string => {
  if (!iso) return ''
  return new Date(iso).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
</script>

<script lang="ts">
import { defineComponent, h } from 'vue'

const Field = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, default: '—' },
    mono: { type: Boolean, default: false },
    small: { type: Boolean, default: false },
    readonly: { type: Boolean, default: false },
  },
  setup(props) {
    return () =>
      h('div', { class: 'field' }, [
        h('span', { class: 'field-label' }, props.label),
        h(
          'span',
          {
            class: ['field-value', { mono: props.mono, small: props.small, readonly: props.readonly }],
          },
          props.value || '—'
        ),
      ])
  },
})

const EditField = defineComponent({
  props: {
    label: { type: String, required: true },
    modelValue: { type: String, default: '' },
    original: { type: String, default: '' },
    editing: { type: Boolean, default: false },
    mono: { type: Boolean, default: false },
    small: { type: Boolean, default: false },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('div', { class: 'field' }, [
        h('span', { class: 'field-label' }, props.label),
        props.editing
          ? h('input', {
              class: ['field-input', { mono: props.mono, small: props.small }],
              type: 'text',
              value: props.modelValue,
              onInput: (e: Event) =>
                emit('update:modelValue', (e.target as HTMLInputElement).value),
            })
          : h(
              'span',
              {
                class: ['field-value', { mono: props.mono, small: props.small }],
              },
              props.original || '—'
            ),
      ])
  },
})

export { Field, EditField }
</script>

<style scoped>
.user-details {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.section {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}

.section-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.625rem 1rem;
}

:deep(.field) {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

:deep(.field-label) {
  font-size: 0.6875rem;
  color: var(--text-3);
  font-weight: 500;
}

:deep(.field-value) {
  font-size: 0.875rem;
  color: var(--text-1);
  word-break: break-all;
}

:deep(.field-value.mono) {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

:deep(.field-value.small) {
  font-size: 0.75rem;
}

:deep(.field-value.readonly) {
  color: var(--text-2);
  font-style: italic;
}

:deep(.field-input) {
  font-size: 0.875rem;
  background: var(--surface-deep);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0.375rem 0.625rem;
  color: var(--text-1);
  outline: none;
  width: 100%;
  transition: border-color 0.15s;
}

:deep(.field-input:focus) {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

:deep(.field-input.mono) {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

:deep(.field-input.small) {
  font-size: 0.75rem;
}

.ua,
.ua-edit {
  display: block;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  padding: 0.625rem;
  border-radius: 8px;
  word-break: break-all;
  color: var(--text-2);
  width: 100%;
  resize: vertical;
}

.ua-edit {
  color: var(--text-1);
}

.url-link {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.8125rem;
  color: var(--brand);
  text-decoration: none;
  word-break: break-all;
}

.url-link:hover {
  text-decoration: underline;
}

.url-edit {
  font-size: 0.8125rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-deep);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0.5rem 0.75rem;
  color: var(--text-1);
  width: 100%;
  outline: none;
  transition: border-color 0.15s;
}

.url-edit:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.muted {
  color: var(--text-3);
  font-style: italic;
}
</style>
