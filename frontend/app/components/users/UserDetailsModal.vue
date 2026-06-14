<template>
  <BaseModal
    :model-value="modelValue"
    :title="user ? userTitle : ''"
    :description="user ? user.external_id : ''"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="user" class="user-details">
      <section class="section">
        <h4 class="section-title">Identificação</h4>
        <div class="grid">
          <Field label="External ID" :value="user.external_id" mono />
          <Field label="ID WhatsApp" :value="user.id_wpp || '—'" mono />
          <Field label="Pixel" :value="user.pixel_name || '—'" />
          <Field label="Nome" :value="fullName(user)" />
          <Field label="E-mail" :value="user.email || '—'" mono />
          <Field label="Telefone" :value="user.phone || '—'" mono />
        </div>
      </section>

      <section class="section">
        <h4 class="section-title">Cookies do Meta</h4>
        <div class="grid">
          <Field label="FBP" :value="user.fbp || '—'" mono small />
          <Field label="FBC" :value="user.fbc || '—'" mono small />
          <Field label="GCLID" :value="user.gclid || '—'" mono small />
        </div>
      </section>

      <section class="section">
        <h4 class="section-title">Geolocalização</h4>
        <div class="grid">
          <Field label="País" :value="user.country || '—'" />
          <Field label="Estado" :value="user.state || '—'" />
          <Field label="Cidade" :value="user.city || '—'" />
          <Field label="CEP" :value="user.postal_code || '—'" />
          <Field label="IP" :value="user.client_ip || '—'" mono />
        </div>
      </section>

      <section v-if="user.client_user_agent" class="section">
        <h4 class="section-title">User-Agent</h4>
        <code class="ua">{{ user.client_user_agent }}</code>
      </section>

      <section class="section">
        <h4 class="section-title">UTM da primeira visita</h4>
        <div class="grid">
          <Field label="Source" :value="user.first_utm_source || '—'" mono small />
          <Field label="Medium" :value="user.first_utm_medium || '—'" mono small />
          <Field label="Campaign" :value="user.first_utm_campaign || '—'" mono small />
          <Field label="Content" :value="user.first_utm_content || '—'" mono small />
          <Field label="Term" :value="user.first_utm_term || '—'" mono small />
        </div>
      </section>

      <section class="section">
        <h4 class="section-title">UTM da última visita</h4>
        <div class="grid">
          <Field label="Source" :value="user.last_utm_source || '—'" mono small />
          <Field label="Medium" :value="user.last_utm_medium || '—'" mono small />
          <Field label="Campaign" :value="user.last_utm_campaign || '—'" mono small />
          <Field label="Content" :value="user.last_utm_content || '—'" mono small />
          <Field label="Term" :value="user.last_utm_term || '—'" mono small />
        </div>
      </section>

      <section v-if="user.url" class="section">
        <h4 class="section-title">Última URL</h4>
        <a :href="user.url" target="_blank" rel="noopener" class="url-link">
          {{ user.url }}
        </a>
      </section>

      <section class="section">
        <h4 class="section-title">Timestamps</h4>
        <div class="grid">
          <Field label="Criado em" :value="formatDate(user.created_at)" />
          <Field label="Atualizado em" :value="formatDate(user.updated_at)" />
        </div>
      </section>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">
        Fechar
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import type { TrackedUser } from '#shared/types'

interface Props {
  modelValue: boolean
  user: TrackedUser | null
}

const props = defineProps<Props>()

defineEmits<{
  'update:modelValue': [value: boolean]
}>()

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
  },
  setup(props) {
    return () =>
      h('div', { class: 'field' }, [
        h('span', { class: 'field-label' }, props.label),
        h(
          'span',
          {
            class: ['field-value', { mono: props.mono, small: props.small }],
          },
          props.value || '—'
        ),
      ])
  },
})

export { Field }
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
  gap: 0.5rem 1rem;
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

.ua {
  display: block;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.75rem;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  padding: 0.625rem;
  border-radius: 8px;
  word-break: break-all;
  color: var(--text-2);
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
</style>
