<template>
  <section class="triggers-card">
    <div class="card-head">
      <h2 class="card-title">Gatilhos de Notificação</h2>
      <p class="card-desc">
        Marque quais eventos deseja receber e use o botão ao lado para disparar um teste imediato.
      </p>
    </div>

    <div class="trigger-list">
      <div
        v-for="item in items"
        :key="item.key"
        class="trigger-row"
      >
        <div class="trigger-left" @click="onToggle(item.key)">
          <BaseCheckbox
            :checked="Boolean(preferences[item.key])"
            @click.stop
            @change="onToggle(item.key)"
          />
          <div class="trigger-texts">
            <div class="trigger-title-line">
              <span class="trigger-name">{{ item.title }}</span>
              <BaseBadge v-if="item.optional" variant="default">Opcional</BaseBadge>
            </div>
            <p class="trigger-sub">{{ item.description }}</p>
          </div>
        </div>

        <BaseButton
          variant="outline"
          size="sm"
          :loading="testingCategory === item.key"
          @click="$emit('test', item.key)"
        >
          <Send :size="14" />
          <span>Testar</span>
        </BaseButton>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Send } from 'lucide-vue-next'
import type { ConfigurableNotificationCategory, NotificationPreferences } from '#shared/types'

const props = defineProps<{
  preferences: NotificationPreferences
  testingCategory: ConfigurableNotificationCategory | null
}>()

const emit = defineEmits<{
  (e: 'toggle', key: ConfigurableNotificationCategory, nextVal: boolean): void
  (e: 'test', key: ConfigurableNotificationCategory): void
}>()

interface TriggerMeta {
  key: ConfigurableNotificationCategory
  title: string
  description: string
  optional?: boolean
}

const items: TriggerMeta[] = [
  {
    key: 'whatsapp_connection',
    title: 'WhatsApp conectado ou desconectado',
    description: 'Alerta imediato para qualquer provedor (QR Code / Evolution Go, API Premium ou Uazapi).',
  },
  {
    key: 'lead_created',
    title: 'Novo lead recebido',
    description: 'Dispara quando um novo lead entra via Landing Page, WhatsApp, anúncio CTWA, Link Rastreável ou CRM.',
  },
  {
    key: 'ai_lead_qualified',
    title: 'Lead qualificado pela IA',
    description: 'Avisa quando a IA analisa a conversa no WhatsApp e qualifica o lead com alta intenção.',
  },
  {
    key: 'sale_approved',
    title: 'Nova venda realizada',
    description: 'Qualquer origem (checkout, webhook, API, site ou WhatsApp), detalhando valor, origem, produto, pixel e cliente.',
  },
  {
    key: 'meta_account_lifecycle',
    title: 'Conta Meta Ads criada ou excluída',
    description: 'Notifica sempre que uma conexão de anúncios da Meta é adicionada ou removida.',
  },
  {
    key: 'google_ads_account_lifecycle',
    title: 'Conta Google Ads criada ou excluída',
    description: 'Notifica sempre que uma conta Google Ads é vinculada ou excluída.',
  },
  {
    key: 'pixel_lifecycle',
    title: 'Pixel criado ou excluído',
    description: 'Avisa quando um Pixel de rastreamento é criado ou removido da empresa.',
  },
  {
    key: 'event_dispatch_error',
    title: 'Erro no envio de evento',
    description: 'Alerta com proteção anti-spam quando o envio para Meta, Google Ads, TikTok ou GA4 retorna erro.',
    optional: true,
  },
]

const onToggle = (key: ConfigurableNotificationCategory) => {
  emit('toggle', key, !props.preferences[key])
}
</script>

<style scoped>
.triggers-card {
  background: var(--surface-2); border: 1px solid var(--border); border-radius: 14px;
  padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem;
}
.card-head { display: flex; flex-direction: column; gap: 0.2rem; }
.card-title { font-size: var(--fs-lg); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.card-desc { font-size: var(--fs-base); color: var(--text-2); margin: 0; }
.trigger-list { display: flex; flex-direction: column; gap: 0.5rem; }
.trigger-row {
  display: flex; align-items: center; justify-content: space-between; gap: 1rem;
  padding: 0.8rem 0.9rem; border-radius: 10px; background: var(--surface-1);
  border: 1px solid var(--border-soft); transition: border-color 0.15s ease;
}
.trigger-row:hover { border-color: var(--brand-border); }
.trigger-left { flex: 1; min-width: 0; display: flex; align-items: flex-start; gap: 0.75rem; cursor: pointer; }
.trigger-texts { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 0.15rem; }
.trigger-title-line { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.trigger-name { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-1); }
.trigger-sub { font-size: var(--fs-sm); color: var(--text-3); margin: 0; line-height: 1.45; }
@media (max-width: 640px) {
  .trigger-row { flex-direction: column; align-items: stretch; }
}
</style>
