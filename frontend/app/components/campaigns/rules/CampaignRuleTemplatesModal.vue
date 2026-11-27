<template>
  <BaseModal
    :model-value="modelValue"
    title="Modelos prontos de regras"
    description="Escolha um ponto de partida validado e ajuste os valores para a sua operação. Todos iniciam em modo simulação por segurança."
    size="xl"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="cat-tabs">
      <button
        v-for="cat in RULE_TEMPLATE_CATEGORIES"
        :key="cat.key"
        type="button"
        class="cat-tab"
        :class="{ active: activeCategory === cat.key }"
        @click="activeCategory = cat.key"
      >
        {{ cat.label }}
      </button>
    </div>

    <div class="tpl-grid">
      <button
        v-for="tpl in filteredTemplates"
        :key="tpl.key"
        type="button"
        class="tpl-card"
        @click="selectTemplate(tpl)"
      >
        <div class="tpl-head">
          <span class="tpl-cat">{{ tpl.categoryLabel }}</span>
          <BaseBadge :variant="badgeVariant(tpl.category)">{{ tpl.badge }}</BaseBadge>
        </div>
        <h3 class="tpl-title">{{ tpl.title }}</h3>
        <p class="tpl-summary">{{ tpl.summary }}</p>
        <div class="tpl-cta">
          <span>Usar modelo</span>
          <ArrowRight :size="14" />
        </div>
      </button>
    </div>

    <template #footer>
      <BaseButton variant="ghost" @click="$emit('update:modelValue', false)">Fechar</BaseButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ArrowRight } from 'lucide-vue-next'
import {
  CAMPAIGN_RULE_TEMPLATES,
  RULE_TEMPLATE_CATEGORIES,
  type CampaignRuleTemplate,
  type RuleTemplateCategory,
} from '~/utils/campaignRuleTemplates'

defineProps<{ modelValue: boolean }>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  select: [template: CampaignRuleTemplate]
}>()

const activeCategory = ref<'all' | RuleTemplateCategory>('all')

const filteredTemplates = computed(() => {
  if (activeCategory.value === 'all') return CAMPAIGN_RULE_TEMPLATES
  return CAMPAIGN_RULE_TEMPLATES.filter((t) => t.category === activeCategory.value)
})

const badgeVariant = (cat: RuleTemplateCategory): 'danger' | 'success' | 'info' | 'warning' => {
  if (cat === 'stop_loss') return 'danger'
  if (cat === 'scale') return 'success'
  if (cat === 'whatsapp_ai') return 'info'
  return 'warning'
}

const selectTemplate = (tpl: CampaignRuleTemplate) => {
  emit('select', tpl)
  emit('update:modelValue', false)
}
</script>

<style scoped>
.cat-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin-bottom: 1rem;
}
.cat-tab {
  padding: 0.4rem 0.8rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: var(--surface-soft);
  color: var(--text-2);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  cursor: pointer;
  transition: all 0.15s ease;
}
.cat-tab:hover { color: var(--text-1); border-color: var(--brand-border); }
.cat-tab.active {
  background: var(--brand-soft);
  border-color: var(--brand-border);
  color: var(--brand);
  font-weight: var(--fw-semibold);
}
.tpl-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}
.tpl-card {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  text-align: left;
  padding: 0.9rem 1rem;
  border-radius: 10px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  cursor: pointer;
  transition: all 0.15s ease;
}
.tpl-card:hover {
  border-color: var(--brand-border);
  background: var(--surface-soft);
}
.tpl-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}
.tpl-cat {
  font-size: var(--fs-xs);
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: var(--fw-semibold);
}
.tpl-title {
  margin: 0;
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
}
.tpl-summary {
  margin: 0;
  font-size: var(--fs-base);
  color: var(--text-2);
  line-height: 1.45;
  flex: 1;
}
.tpl-cta {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  margin-top: 0.25rem;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--brand);
}
@media (max-width: 720px) {
  .tpl-grid { grid-template-columns: 1fr; }
}
</style>
