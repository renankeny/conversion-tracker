<template>
  <article class="rule-card" :class="{ 'is-disabled': !rule.enabled }">
    <div class="card-top">
      <div class="title-group">
        <button
          type="button"
          class="switch-btn"
          :class="{ on: rule.enabled }"
          role="switch"
          :aria-checked="rule.enabled"
          :title="rule.enabled ? 'Pausar regra' : 'Ativar regra'"
          @click="$emit('toggle', rule)"
        >
          <span class="switch-thumb" />
        </button>

        <div class="title-meta">
          <div class="title-line">
            <h3 class="rule-name">{{ rule.name }}</h3>
            <BaseBadge v-if="rule.dry_run || rule.action_type === 'notify_only'" variant="warning">Simulação</BaseBadge>
            <BaseBadge v-else-if="rule.enabled" variant="success">Ativa</BaseBadge>
            <BaseBadge v-else variant="default">Pausada</BaseBadge>
          </div>
          <p v-if="rule.description" class="rule-desc">{{ rule.description }}</p>
        </div>
      </div>

      <div class="card-actions">
        <BaseButton variant="ghost" size="sm" :loading="running" title="Simular agora sem alterar campanhas" @click="$emit('run', rule, true)">
          <FlaskConical :size="14" />
          <span>Simular</span>
        </BaseButton>
        <BaseButton variant="outline" size="sm" :loading="running" title="Executar regra agora" @click="$emit('run', rule, false)">
          <Play :size="14" />
          <span>Executar</span>
        </BaseButton>
        <button type="button" class="icon-btn" title="Editar regra" @click="$emit('edit', rule)">
          <Pencil :size="14" />
        </button>
        <button type="button" class="icon-btn is-danger" title="Excluir regra" @click="$emit('delete', rule)">
          <Trash2 :size="14" />
        </button>
      </div>
    </div>

    <div class="formula-strip">
      <span class="formula-part">
        <strong>SE</strong> {{ conditionsText }}
      </span>
      <ArrowRight :size="14" class="formula-arrow" />
      <span class="formula-action">{{ actionText }}</span>
    </div>

    <div class="meta-strip">
      <span>{{ platformLabel }} · {{ levelLabel }}</span>
      <span>Janela: {{ periodLabel }}</span>
      <span v-if="rule.rule_kind === 'schedule'">Horário: {{ rule.schedule_time }} BRT</span>
      <span v-else>A cada {{ rule.frequency_minutes }} min{{ rule.min_hour_sp > 0 ? ` (após ${rule.min_hour_sp}h)` : '' }}</span>
      <span v-if="rule.cooldown_minutes > 0">Cooldown: {{ Math.round(rule.cooldown_minutes / 60) }}h</span>
      <span v-if="rule.pixel_name">Pixel: {{ rule.pixel_name }}</span>
      <span class="meta-right">
        {{ rule.trigger_count }} {{ rule.trigger_count === 1 ? 'disparo' : 'disparos' }}
        <template v-if="rule.last_run_at"> · Última leitura {{ formatShortDate(rule.last_run_at) }}</template>
      </span>
    </div>

    <div v-if="rule.last_error" class="err-strip">
      <AlertTriangle :size="13" />
      <span>Último aviso: {{ rule.last_error }}</span>
    </div>
  </article>
</template>

<script setup lang="ts">
import { AlertTriangle, ArrowRight, FlaskConical, Pencil, Play, Trash2 } from 'lucide-vue-next'
import type { CampaignRule } from '#shared/types'
import {
  formatRuleActionSummary,
  formatRuleConditionsSummary,
  RULE_LEVEL_OPTIONS,
  RULE_PERIOD_OPTIONS,
  RULE_PLATFORM_OPTIONS,
} from '~/utils/campaignRuleOptions'

const props = defineProps<{
  rule: CampaignRule
  running?: boolean
}>()

defineEmits<{
  toggle: [rule: CampaignRule]
  run: [rule: CampaignRule, dryRun: boolean]
  edit: [rule: CampaignRule]
  delete: [rule: CampaignRule]
}>()

const conditionsText = computed(() =>
  formatRuleConditionsSummary(props.rule.conditions, props.rule.schedule_time, props.rule.rule_kind),
)
const actionText = computed(() => formatRuleActionSummary(props.rule))
const platformLabel = computed(() => RULE_PLATFORM_OPTIONS.find((o) => o.value === props.rule.platform)?.label || 'Todas')
const levelLabel = computed(() => RULE_LEVEL_OPTIONS.find((o) => o.value === props.rule.level)?.label || props.rule.level)
const periodLabel = computed(() => RULE_PERIOD_OPTIONS.find((o) => o.value === props.rule.period_preset)?.label || props.rule.period_preset)

const formatShortDate = (iso: string) =>
  new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(iso))
</script>

<style scoped>
.rule-card {
  display: flex; flex-direction: column; gap: 0.7rem; padding: 0.95rem 1.05rem;
  border-radius: 12px; background: var(--surface-2); border: 1px solid var(--border-soft);
  transition: border-color 0.15s ease;
}
.rule-card:hover { border-color: var(--border); }
.rule-card.is-disabled { opacity: 0.72; }
.card-top { display: flex; align-items: flex-start; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; }
.title-group { display: flex; align-items: flex-start; gap: 0.75rem; min-width: 0; flex: 1; }
.switch-btn {
  width: 36px; height: 20px; border-radius: 999px; border: 1px solid var(--border);
  background: var(--surface-soft); padding: 2px; cursor: pointer; flex-shrink: 0; margin-top: 2px;
  display: inline-flex; align-items: center; transition: all 0.15s ease;
}
.switch-btn.on { background: var(--brand); border-color: var(--brand); justify-content: flex-end; }
.switch-thumb { width: 14px; height: 14px; border-radius: 50%; background: var(--text-1); }
.switch-btn.on .switch-thumb { background: var(--brand-text); }
.title-meta { min-width: 0; flex: 1; }
.title-line { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.rule-name { margin: 0; font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-1); }
.rule-desc { margin: 0.2rem 0 0; font-size: var(--fs-base); color: var(--text-3); }
.card-actions { display: flex; align-items: center; gap: 0.35rem; flex-wrap: wrap; }
.icon-btn {
  width: 30px; height: 30px; border-radius: 8px; border: 1px solid var(--border-soft);
  background: transparent; color: var(--text-2); display: inline-flex; align-items: center; justify-content: center; cursor: pointer;
}
.icon-btn:hover { background: var(--surface-soft); color: var(--text-1); }
.icon-btn.is-danger:hover { background: var(--danger-soft); border-color: var(--danger-border); color: var(--danger); }
.formula-strip {
  display: flex; align-items: center; gap: 0.6rem; flex-wrap: wrap;
  padding: 0.55rem 0.75rem; border-radius: 8px; background: var(--surface-deep); border: 1px solid var(--border-soft);
  font-size: var(--fs-base); color: var(--text-2);
}
.formula-part strong { color: var(--brand); font-weight: var(--fw-semibold); margin-right: 0.2rem; }
.formula-arrow { color: var(--text-3); flex-shrink: 0; }
.formula-action { color: var(--text-1); font-weight: var(--fw-semibold); }
.meta-strip {
  display: flex; align-items: center; flex-wrap: wrap; gap: 0.45rem 1rem;
  font-size: var(--fs-sm); color: var(--text-3);
}
.meta-right { margin-left: auto; }
.err-strip {
  display: flex; align-items: center; gap: 0.4rem; padding: 0.45rem 0.7rem; border-radius: 8px;
  background: var(--danger-soft); border: 1px solid var(--danger-border); color: var(--danger); font-size: var(--fs-sm);
}
@media (max-width: 720px) { .meta-right { margin-left: 0; } }
</style>
