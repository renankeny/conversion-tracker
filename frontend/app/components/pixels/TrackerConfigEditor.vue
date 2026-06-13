<template>
  <div class="tracker-config">
    <!-- 1. Eventos automáticos -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Eventos automáticos</h4>
        <p class="section-hint">
          Eventos que o tracker dispara sem precisar de código no site.
        </p>
      </header>

      <div class="toggles-grid">
        <ToggleCard
          v-model="config.auto_page_view"
          title="PageView automático"
          description="Dispara PageView ao carregar a página (recomendado)"
          tag="PageView"
        />
        <ToggleCard
          v-model="config.auto_checkout_buttons"
          title="Botões de checkout"
          description="Detecta links com a palavra-chave abaixo, reescreve UTMs e dispara InitiateCheckout no clique"
          tag="InitiateCheckout"
        />
        <ToggleCard
          v-model="config.auto_form_scraping"
          title="Captura de formulário"
          description="Lê email, telefone e nome dos forms da página automaticamente"
          tag="PII"
        />
        <ToggleCard
          v-model="config.auto_form_lead"
          title="Lead em formulário"
          description="Dispara Lead automaticamente quando um form com email ou telefone é submetido"
          tag="Lead"
        />
        <ToggleCard
          v-model="config.whatsapp_protocol"
          title="Protocolo WhatsApp"
          description="Adiciona &quot;Protocolo: <id>&quot; em links wa.me e dispara Lead no clique"
          tag="Lead"
        />
        <ToggleCard
          v-model="config.send_scrolls"
          title="Scroll 25/50/75/90"
          description="Dispara eventos custom ao atingir percentuais de rolagem"
          tag="Scroll_*"
        />
        <ToggleCard
          v-model="config.send_timer_1min"
          title="Timer 1 minuto"
          description="Dispara Timer_1min após 60s na página"
          tag="Timer_1min"
        />
        <ToggleCard
          v-model="config.send_view_video"
          title="View Video"
          description="Dispara ViewVideo_25/50/75/90 ao reproduzir &lt;video&gt;"
          tag="ViewVideo_*"
        />
      </div>
    </section>

    <!-- 2. Produto -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Produto e checkout</h4>
        <p class="section-hint">
          Valor e moeda enviados nos eventos InitiateCheckout/Purchase.
        </p>
      </header>

      <div class="form-grid">
        <BaseInput
          v-model="config.product_checkout"
          label="Palavra-chave do checkout"
          placeholder="Ex: pay, /checkout, comprar"
          hint="Substring usada para identificar links de checkout no site"
        />
        <BaseInput
          v-model="config.product_currency"
          label="Moeda"
          placeholder="BRL"
          hint="Ex: BRL, USD, EUR"
        />
      </div>

      <div class="prices-block">
        <label class="prices-label">Valores dos produtos / ofertas</label>
        <p class="section-hint">
          Cada valor corresponde a um botão de checkout, em ordem de aparição na página.
        </p>
        <div
          v-for="(_, idx) in (config.product_value || [])"
          :key="idx"
          class="price-row"
        >
          <BaseInput
            :model-value="formatPrice(config.product_value[idx])"
            type="number"
            :placeholder="'Valor do botão ' + (idx + 1)"
            @update:model-value="(v) => updatePrice(idx, v)"
          />
          <button
            type="button"
            class="row-remove"
            @click="removePrice(idx)"
            aria-label="Remover preço"
          >
            ×
          </button>
        </div>
        <BaseButton type="button" variant="ghost" size="sm" @click="addPrice">
          + Adicionar valor
        </BaseButton>
      </div>
    </section>

    <!-- 3. Meta Pixel ID -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Meta Pixel (browser side)</h4>
        <p class="section-hint">
          ID usado no <code>fbq('init', ...)</code> do Meta Pixel browser.
          Em branco usa automaticamente o primeiro Pixel ID configurado.
        </p>
      </header>

      <BaseInput
        v-model="config.meta_init_pixel_id"
        label="Meta Init Pixel ID (opcional)"
        placeholder="Deixe vazio para usar o primeiro Pixel ID"
      />
    </section>

    <!-- 4. UTM origin -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Parâmetro de UTM</h4>
        <p class="section-hint">
          Nome do parâmetro de URL onde o tracker injeta o <code>userId</code>.
          Padrão: <code>utm_source</code>. Para Hotmart, use <code>xcod</code>.
        </p>
      </header>

      <BaseInput
        v-model="config.utm_origin_param"
        label="Nome do parâmetro"
        placeholder="utm_source"
      />
    </section>

    <!-- 5. Page rules -->
    <section class="config-section">
      <header class="section-header">
        <div class="section-title-row">
          <h4 class="section-title">Regras por página</h4>
          <BaseButton type="button" variant="ghost" size="sm" @click="addPageRule">
            + Adicionar regra
          </BaseButton>
        </div>
        <p class="section-hint">
          Dispara um evento quando a URL contém a substring informada.
        </p>
      </header>

      <div v-if="!config.page_rules.length" class="empty-rules">
        Nenhuma regra cadastrada.
      </div>

      <div v-for="(rule, idx) in config.page_rules" :key="idx" class="rule-row">
        <div class="rule-grid">
          <BaseInput
            :model-value="rule.page"
            label="Substring na URL"
            placeholder="ex: /obrigado"
            @update:model-value="(v) => updateRuleField(idx, 'page', v)"
          />
          <BaseInput
            :model-value="rule.event"
            label="Evento"
            placeholder="ex: Lead, Purchase"
            @update:model-value="(v) => updateRuleField(idx, 'event', v)"
          />
          <BaseInput
            :model-value="formatPrice(rule.value)"
            label="Valor (opcional)"
            type="number"
            placeholder="0"
            @update:model-value="(v) => updateRuleValue(idx, v)"
          />
          <BaseInput
            :model-value="rule.currency || ''"
            label="Moeda (opcional)"
            placeholder="BRL"
            @update:model-value="(v) => updateRuleField(idx, 'currency', v)"
          />
        </div>
        <button
          type="button"
          class="row-remove"
          @click="removePageRule(idx)"
          aria-label="Remover regra"
        >
          ×
        </button>
      </div>
    </section>

    <!-- 6. Form mappings -->
    <section class="config-section">
      <header class="section-header">
        <h4 class="section-title">Mapeamento de campos de formulário</h4>
        <p class="section-hint">
          Atributos <code>name</code> dos inputs que o tracker procura para extrair
          PII automaticamente. Separe por vírgula.
        </p>
      </header>

      <div class="form-grid">
        <BaseInput
          v-model="formEmail"
          label="Campos de e-mail"
          placeholder="email, form_fields[email]"
        />
        <BaseInput
          v-model="formPhone"
          label="Campos de telefone"
          placeholder="tel, telefone, phone"
        />
        <BaseInput
          v-model="formFirstName"
          label="Campos de primeiro nome"
          placeholder="nome, name, first_name"
        />
        <BaseInput
          v-model="formLastName"
          label="Campos de sobrenome"
          placeholder="sobrenome, lastname"
        />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import ToggleCard from '~/components/pixels/ToggleCard.vue'
import type { TrackerConfig } from '#shared/types'

interface Props {
  modelValue: TrackerConfig
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: TrackerConfig]
}>()

// Acesso reativo direto ao objeto pai (mantém v-model imutável-friendly)
const config = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

// === product_value ===
const formatPrice = (n: number | undefined | null): string => {
  if (n === undefined || n === null) return ''
  if (typeof n !== 'number' || Number.isNaN(n)) return ''
  return n === 0 ? '' : String(n)
}

const parsePrice = (raw: string): number => {
  if (!raw) return 0
  const cleaned = String(raw).replace(',', '.').trim()
  const n = parseFloat(cleaned)
  return Number.isNaN(n) ? 0 : n
}

const updatePrice = (idx: number, raw: string) => {
  const arr = [...(config.value.product_value || [])]
  arr[idx] = parsePrice(raw)
  config.value = { ...config.value, product_value: arr }
}

const addPrice = () => {
  config.value = {
    ...config.value,
    product_value: [...(config.value.product_value || []), 0],
  }
}

const removePrice = (idx: number) => {
  const next = [...(config.value.product_value || [])]
  next.splice(idx, 1)
  config.value = { ...config.value, product_value: next }
}

// === page_rules ===
const updateRuleField = (idx: number, field: 'page' | 'event' | 'currency', value: string) => {
  const rules = [...(config.value.page_rules || [])]
  if (!rules[idx]) return
  rules[idx] = { ...rules[idx], [field]: value }
  config.value = { ...config.value, page_rules: rules }
}

const updateRuleValue = (idx: number, raw: string) => {
  const rules = [...(config.value.page_rules || [])]
  if (!rules[idx]) return
  const v = raw.trim() === '' ? undefined : parsePrice(raw)
  rules[idx] = { ...rules[idx], value: v }
  config.value = { ...config.value, page_rules: rules }
}

const addPageRule = () => {
  config.value = {
    ...config.value,
    page_rules: [
      ...(config.value.page_rules || []),
      { page: '', event: '', currency: 'BRL' },
    ],
  }
}

const removePageRule = (idx: number) => {
  const next = [...(config.value.page_rules || [])]
  next.splice(idx, 1)
  config.value = { ...config.value, page_rules: next }
}

// === form_mappings: array <-> string CSV ===
const csvToArray = (s: string): string[] =>
  s.split(',').map((x) => x.trim()).filter(Boolean)

const formEmail = computed({
  get: () => (config.value.form_mappings?.email || []).join(', '),
  set: (s) => {
    config.value = {
      ...config.value,
      form_mappings: { ...(config.value.form_mappings || emptyFormMappings()), email: csvToArray(s) },
    }
  },
})

const formPhone = computed({
  get: () => (config.value.form_mappings?.phone || []).join(', '),
  set: (s) => {
    config.value = {
      ...config.value,
      form_mappings: { ...(config.value.form_mappings || emptyFormMappings()), phone: csvToArray(s) },
    }
  },
})

const formFirstName = computed({
  get: () => (config.value.form_mappings?.first_name || []).join(', '),
  set: (s) => {
    config.value = {
      ...config.value,
      form_mappings: { ...(config.value.form_mappings || emptyFormMappings()), first_name: csvToArray(s) },
    }
  },
})

const formLastName = computed({
  get: () => (config.value.form_mappings?.last_name || []).join(', '),
  set: (s) => {
    config.value = {
      ...config.value,
      form_mappings: { ...(config.value.form_mappings || emptyFormMappings()), last_name: csvToArray(s) },
    }
  },
})

const emptyFormMappings = () => ({ email: [], phone: [], first_name: [], last_name: [] })
</script>

<style scoped>
.tracker-config {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.config-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  border-radius: 12px;
}

.section-header {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.section-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.section-title {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text-strong);
}

.section-hint {
  font-size: 0.8125rem;
  color: var(--text-2);
  line-height: 1.5;
}

.section-hint code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--surface-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
}

.toggles-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 0.625rem;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}

.prices-block {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.prices-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-1);
}

.price-row,
.rule-row {
  position: relative;
  display: flex;
  align-items: flex-end;
  gap: 0.5rem;
}

.price-row > :first-child {
  flex: 1;
}

.rule-row {
  background: var(--surface-2);
  padding: 0.75rem;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  margin-top: 0.5rem;
}

.rule-grid {
  flex: 1;
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 0.5rem;
}

@media (max-width: 768px) {
  .rule-grid {
    grid-template-columns: 1fr 1fr;
  }
}

.row-remove {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-3);
  font-size: 1.25rem;
  line-height: 1;
  cursor: pointer;
  padding: 0.375rem 0.625rem;
  border-radius: 6px;
  transition: all 0.15s;
  flex-shrink: 0;
  align-self: flex-end;
}

.row-remove:hover {
  background: var(--danger-soft);
  border-color: var(--danger);
  color: var(--danger);
}

.empty-rules {
  font-size: 0.8125rem;
  color: var(--text-3);
  font-style: italic;
  padding: 0.5rem 0;
}
</style>
