<template>
  <section class="guard-section">
    <h4 class="sec-title">Agendamento, filtros de nome e alertas</h4>

    <div class="grid-3">
      <template v-if="form.rule_kind === 'schedule'">
        <BaseInput
          v-model="form.schedule_time"
          label="Horário (Brasília)"
          placeholder="08:00"
          hint="Formato 24h (ex: 00:05, 08:00, 22:00)"
        />
      </template>
      <template v-else>
        <BaseDropdown
          :model-value="form.frequency_minutes"
          label="Frequência de verificação"
          :options="RULE_FREQUENCY_OPTIONS"
          size="sm"
          @update:model-value="(v) => (form.frequency_minutes = Number(v || 30))"
        />
        <BaseInput
          :model-value="String(form.min_hour_sp)"
          type="number"
          label="Só avaliar a partir das (hora)"
          placeholder="0"
          hint="Ex: 11 = só avalia após 11h (0 = qualquer hora)"
          @update:model-value="(v) => (form.min_hour_sp = clampHour(Number(v || 0)))"
        />
      </template>

      <BaseInput
        :model-value="String(cooldownHours)"
        type="number"
        label="Cooldown por item (horas)"
        placeholder="6"
        hint="Evita disparar repetidamente no mesmo item"
        @update:model-value="(v) => (form.cooldown_minutes = Math.max(0, Math.round(Number(v || 0) * 60)))"
      />
    </div>

    <div class="days-block">
      <span class="field-label">Dias da semana (vazio = todos os dias)</span>
      <div class="days-row">
        <button
          v-for="d in RULE_WEEKDAYS"
          :key="d.value"
          type="button"
          class="day-pill"
          :class="{ active: form.schedule_days.includes(d.value) }"
          @click="toggleDay(d.value)"
        >
          {{ d.short }}
        </button>
      </div>
    </div>

    <div class="grid-2">
      <BaseInput
        v-model="form.name_contains"
        label="Aplicar somente se o nome contiver"
        placeholder="Ex: [ESCALA], [SURF]"
        hint="Separe por vírgula. Vazio = todas as campanhas."
      />
      <BaseInput
        v-model="form.name_excludes"
        label="Ignorar se o nome contiver"
        placeholder="Ex: [FIXO], [NAO_PAUSAR]"
        hint="Protege campanhas críticas contra alteração."
      />
    </div>

    <div class="alerts-strip">
      <BaseCheckbox
        v-model="form.notify_push"
        label="Notificar no painel e Web Push"
        hint="Envia alerta imediato no sino e no navegador quando a regra disparar."
      />
      <BaseCheckbox
        v-model="form.notify_whatsapp"
        label="Enviar alerta no WhatsApp"
        hint="Usa a instância Evolution Go conectada desta empresa."
      />
    </div>

    <div v-if="form.notify_whatsapp" class="wa-phone-wrap">
      <BaseInput
        v-model="form.whatsapp_phone"
        label="WhatsApp do gestor (com DDI + DDD)"
        placeholder="5511999998888"
        hint="Somente números, incluindo código do país (55 para Brasil)."
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import type { CampaignRulePayload } from '#shared/types'
import { RULE_FREQUENCY_OPTIONS, RULE_WEEKDAYS } from '~/utils/campaignRuleOptions'

const form = defineModel<CampaignRulePayload>({ required: true })

const cooldownHours = computed(() => Math.round((form.value.cooldown_minutes || 0) / 60))

const clampHour = (h: number) => Math.max(0, Math.min(23, Math.floor(h)))

const toggleDay = (day: number) => {
  const exists = form.value.schedule_days.includes(day)
  form.value.schedule_days = exists
    ? form.value.schedule_days.filter((d) => d !== day)
    : [...form.value.schedule_days, day].sort((a, b) => a - b)
}
</script>

<style scoped>
.guard-section {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border-soft);
}
.sec-title { margin: 0; font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); }
.grid-3 { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; }
.grid-2 { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.75rem; }
.days-block { display: flex; flex-direction: column; gap: 0.4rem; }
.field-label { font-size: var(--fs-base); font-weight: var(--fw-medium); color: var(--text-2); }
.days-row { display: flex; flex-wrap: wrap; gap: 0.4rem; }
.day-pill {
  padding: 0.35rem 0.65rem;
  border-radius: 7px;
  border: 1px solid var(--border);
  background: var(--surface-soft);
  color: var(--text-2);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  cursor: pointer;
}
.day-pill.active {
  background: var(--brand-soft);
  border-color: var(--brand-border);
  color: var(--brand);
  font-weight: var(--fw-semibold);
}
.alerts-strip {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
  padding: 0.75rem 0.9rem;
  border-radius: 10px;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
}
.wa-phone-wrap { max-width: 340px; }
@media (max-width: 720px) {
  .grid-3, .grid-2, .alerts-strip { grid-template-columns: 1fr; }
}
</style>
