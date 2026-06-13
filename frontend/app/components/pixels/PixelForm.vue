<template>
  <div class="pixel-form">
    <!-- Abas -->
    <div class="tabs">
      <button
        type="button"
        class="tab"
        :class="{ active: activeTab === 'basic' }"
        @click="activeTab = 'basic'"
      >
        Básico
      </button>
      <button
        type="button"
        class="tab"
        :class="{ active: activeTab === 'tracker' }"
        @click="activeTab = 'tracker'"
      >
        Tracker Config
      </button>
    </div>

    <!-- Aba Básico -->
    <form v-show="activeTab === 'basic'" class="tab-content" @submit.prevent="onSubmit">
      <BaseInput
        v-model="form.name"
        label="Nome"
        placeholder="Ex: principal"
        hint="Identificador interno (slug). Usado em /tracker.js?pixel=&lt;nome&gt;"
        :error="errors.name"
        required
      />

      <BaseInput
        v-model="form.label"
        label="Rótulo"
        placeholder="Ex: Pixel principal de vendas"
        hint="Nome amigável exibido no painel"
      />

      <BaseInput
        v-model="form.tracking_host"
        label="Subdomínio first-party (opcional)"
        placeholder="tracking.seu-dominio.com"
        hint="Subdomínio do cliente apontado via CNAME para o servidor TrackFind. Garante cookies first-party e contorna adblockers."
      />

      <div class="form-section">
        <div class="section-header">
          <h4 class="section-title">Pixel IDs Meta</h4>
          <BaseButton type="button" variant="ghost" size="sm" @click="addPixelRow">
            + Adicionar pixel
          </BaseButton>
        </div>
        <p class="section-hint">
          Você pode espelhar o evento para múltiplos pixels da Meta. Cada linha precisa de
          ID + access token.
        </p>

        <div v-for="(_, idx) in form.pixel_ids" :key="idx" class="pixel-row">
          <div class="pixel-row-grid">
            <BaseInput
              v-model="form.pixel_ids[idx]"
              label="Pixel ID"
              placeholder="123456789012345"
              :error="idx === 0 ? errors.pixel_ids : ''"
            />
            <BaseInput
              v-model="form.access_tokens[idx]"
              label="Access Token"
              :type="showTokens ? 'text' : 'password'"
              :placeholder="isEditing ? 'Mantém o atual se vazio' : 'EAAxxxxxxxxxxxx...'"
              :hint="isEditing && !form.access_tokens[idx] ? 'Em branco mantém o token atual' : ''"
            />
            <BaseInput
              v-model="form.test_codes[idx]"
              label="Test Code"
              placeholder="TEST12345"
            />
          </div>
          <button
            v-if="form.pixel_ids.length > 1"
            type="button"
            class="row-remove"
            @click="removePixelRow(idx)"
            aria-label="Remover linha"
          >
            ×
          </button>
        </div>

        <label class="show-tokens">
          <input v-model="showTokens" type="checkbox" />
          <span>Mostrar tokens</span>
        </label>
      </div>

      <div class="form-section">
        <h4 class="section-title">Domínios autorizados</h4>
        <p class="section-hint">
          Sites onde o tracker roda. Separe por vírgula. Em produção (STRICT_CORS)
          só esses domínios podem enviar eventos.
        </p>
        <BaseInput
          v-model="domainsInput"
          placeholder="exemplo.com.br, www.exemplo.com.br"
        />
      </div>

      <div class="form-section">
        <label class="enabled-toggle">
          <input v-model="form.enabled" type="checkbox" />
          <span>Pixel ativo</span>
        </label>
        <p class="section-hint">Quando desativado, o /tracker.js retorna `pixel disabled`.</p>
      </div>
    </form>

    <!-- Aba Tracker Config -->
    <div v-show="activeTab === 'tracker'" class="tab-content">
      <TrackerConfigEditor v-model="form.tracker_config" />
    </div>
  </div>
</template>

<script setup lang="ts">
import TrackerConfigEditor from '~/components/pixels/TrackerConfigEditor.vue'
import type { Pixel, PixelInput, TrackerConfig } from '#shared/types'

interface Props {
  pixel?: Pixel | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  submit: [input: PixelInput]
}>()

const isEditing = computed(() => !!props.pixel)

const activeTab = ref<'basic' | 'tracker'>('basic')

const defaultTrackerConfig = (): TrackerConfig => ({
  product_value: [],
  product_currency: 'BRL',
  product_checkout: '',
  auto_page_view: true,
  auto_checkout_buttons: true,
  auto_form_scraping: true,
  whatsapp_protocol: true,
  send_scrolls: false,
  send_timer_1min: false,
  send_view_video: false,
  page_rules: [],
  form_mappings: {
    email: ['email', 'form_fields[email]'],
    phone: ['tel', 'telefone', 'phone', 'form_fields[phone]'],
    first_name: ['nome', 'name', 'first_name', 'form_fields[name]', 'form_fields[firstname]'],
    last_name: ['sobrenome', 'lastname', 'last_name', 'form_fields[lastname]'],
  },
  meta_init_pixel_id: '',
  utm_origin_param: 'utm_source',
})

const form = reactive<{
  name: string
  label: string
  tracking_host: string
  pixel_ids: string[]
  access_tokens: string[]
  test_codes: string[]
  domains: string[]
  enabled: boolean
  tracker_config: TrackerConfig
}>({
  name: '',
  label: '',
  tracking_host: '',
  pixel_ids: [''],
  access_tokens: [''],
  test_codes: [''],
  domains: [],
  enabled: true,
  tracker_config: defaultTrackerConfig(),
})

const domainsInput = ref('')
const showTokens = ref(false)

const errors = reactive({
  name: '',
  pixel_ids: '',
})

const fillFromPixel = (pixel: Pixel) => {
  form.name = pixel.name
  form.label = pixel.label || ''
  form.tracking_host = pixel.tracking_host || ''
  form.pixel_ids = [...(pixel.pixel_ids || [''])]
  form.access_tokens = pixel.pixel_ids.map(() => '')
  form.test_codes = [...(pixel.test_codes || pixel.pixel_ids.map(() => ''))]
  form.domains = [...(pixel.domains || [])]
  form.enabled = pixel.enabled
  form.tracker_config = {
    ...defaultTrackerConfig(),
    ...(pixel.tracker_config || {}),
  }
  domainsInput.value = (pixel.domains || []).join(', ')
}

watchEffect(() => {
  if (props.pixel) {
    fillFromPixel(props.pixel)
  }
})

const addPixelRow = () => {
  form.pixel_ids.push('')
  form.access_tokens.push('')
  form.test_codes.push('')
}

const removePixelRow = (idx: number) => {
  form.pixel_ids.splice(idx, 1)
  form.access_tokens.splice(idx, 1)
  form.test_codes.splice(idx, 1)
}

const validate = (): boolean => {
  errors.name = ''
  errors.pixel_ids = ''

  let ok = true

  if (!form.name.trim()) {
    errors.name = 'Nome é obrigatório'
    activeTab.value = 'basic'
    ok = false
  }

  const pixelIds = form.pixel_ids.map((s) => s.trim()).filter(Boolean)
  if (pixelIds.length === 0) {
    errors.pixel_ids = 'Pelo menos um pixel ID é obrigatório'
    activeTab.value = 'basic'
    ok = false
  }

  return ok
}

const buildPayload = (): PixelInput => {
  const indexes = form.pixel_ids
    .map((id, i) => (id.trim() ? i : -1))
    .filter((i) => i !== -1)

  const pixel_ids = indexes.map((i) => form.pixel_ids[i].trim())
  const test_codes = indexes.map((i) => (form.test_codes[i] || '').trim())

  const domains = domainsInput.value
    .split(',')
    .map((d) => d.trim().toLowerCase())
    .filter(Boolean)

  const payload: PixelInput = {
    name: form.name.trim(),
    label: form.label.trim(),
    tracking_host: form.tracking_host.trim().toLowerCase(),
    pixel_ids,
    test_codes,
    domains,
    enabled: form.enabled,
    tracker_config: form.tracker_config,
  }

  const tokens = indexes.map((i) => (form.access_tokens[i] || '').trim())
  const allFilled = tokens.every((t) => t.length > 0)

  if (allFilled) {
    payload.access_tokens = tokens
  } else if (!isEditing.value) {
    payload.access_tokens = tokens
  }

  return payload
}

const onSubmit = () => {
  if (!validate()) return
  emit('submit', buildPayload())
}

defineExpose({ submit: onSubmit })
</script>

<style scoped>
.pixel-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.tabs {
  display: flex;
  gap: 0.25rem;
  border-bottom: 1px solid var(--border);
  padding: 0;
  margin: 0;
}

.tab {
  padding: 0.75rem 1.25rem;
  background: transparent;
  border: none;
  color: var(--text-2);
  font-size: 0.9375rem;
  font-weight: 500;
  cursor: pointer;
  position: relative;
  transition: color 0.15s;
}

.tab:hover {
  color: var(--text-1);
}

.tab.active {
  color: var(--brand);
}

.tab.active::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--brand);
}

.tab-content {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--border-soft);
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.section-title {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text-strong);
}

.section-hint {
  font-size: 0.8125rem;
  color: var(--text-3);
}

.pixel-row {
  position: relative;
  padding: 0.75rem;
  background: var(--surface-soft);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  margin-top: 0.5rem;
}

.pixel-row-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 0.75rem;
}

@media (max-width: 768px) {
  .pixel-row-grid {
    grid-template-columns: 1fr;
  }
}

.row-remove {
  position: absolute;
  top: 0.25rem;
  right: 0.5rem;
  background: transparent;
  border: none;
  color: var(--text-3);
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  transition: all 0.15s;
}

.row-remove:hover {
  color: var(--danger);
  background: var(--danger-soft);
}

.show-tokens,
.enabled-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: var(--text-1);
  cursor: pointer;
  user-select: none;
}

.show-tokens input[type='checkbox'],
.enabled-toggle input[type='checkbox'] {
  width: 1rem;
  height: 1rem;
  accent-color: var(--brand);
  cursor: pointer;
}
</style>
