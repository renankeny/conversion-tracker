<template>
  <div class="section-box">
    <h4 class="box-title">Destino no WhatsApp</h4>
    <BaseDropdown
      :model-value="routingMode"
      label="Como redirecionar os cliques"
      :options="routingOptions"
      @update:model-value="$emit('update:routingMode', String($event) as LinkRoutingMode)"
    />

    <BaseDropdown
      v-if="routingMode === 'specific_instance'"
      :model-value="whatsappInstanceId"
      label="Conexão WhatsApp deste Pixel"
      :options="instanceOptions"
      placeholder="Selecione a conexão WhatsApp..."
      @update:model-value="$emit('update:whatsappInstanceId', $event ? Number($event) : null)"
    />

    <div v-if="routingMode !== 'manual_phone' && visibleInstances.length > 0" class="conn-list">
      <div v-for="inst in visibleInstances" :key="inst.id" class="conn-item">
        <div class="conn-left">
          <span class="conn-dot" :class="`dot-${getWhatsAppStatusState(inst.connection_status, inst.enabled)}`" />
          <span class="conn-name">{{ inst.name }}</span>
          <span class="conn-prov">{{ getWhatsAppProviderLabel(inst.provider) }}</span>
          <span v-if="formatOwnerJidPhone(inst.owner_jid)" class="conn-phone">
            {{ formatOwnerJidPhone(inst.owner_jid) }}
          </span>
        </div>
        <span class="conn-badge" :class="`badge-${getWhatsAppStatusState(inst.connection_status, inst.enabled)}`">
          {{ getWhatsAppStatusLabel(inst.connection_status, inst.provider, inst.enabled) }}
        </span>
      </div>
    </div>

    <div v-else-if="routingMode !== 'manual_phone' && instances.length === 0" class="conn-empty">
      Nenhuma conexão WhatsApp vinculada a este Pixel ainda. Conecte em Integrações → WhatsApp ou informe um número abaixo.
    </div>

    <BaseInput
      :model-value="fallbackPhone"
      :label="routingMode === 'manual_phone' ? 'Número de WhatsApp (obrigatório)' : 'Número reserva se desconectar (opcional)'"
      placeholder="Ex: 5511999998888"
      :hint="routingMode === 'manual_phone'
        ? 'Código do país + DDD + número (apenas dígitos).'
        : 'Usado caso o aparelho desconecte ou ainda não tenha sincronizado o número via Webhook.'"
      :error="destinationError"
      @update:model-value="$emit('update:fallbackPhone', $event)"
    />

    <BaseInput
      :model-value="prefillMessage"
      label="Mensagem inicial pré-preenchida"
      placeholder="Olá! Vim pelo Instagram e gostaria de mais informações."
      hint="O código Protocolo (igual ao das Landing Pages) é adicionado automaticamente ao final da mensagem."
      @update:model-value="$emit('update:prefillMessage', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import type { LinkRoutingMode, WhatsAppInstance } from '#shared/types'
import {
  formatInstanceOptionLabel,
  formatOwnerJidPhone,
  getWhatsAppProviderLabel,
  getWhatsAppStatusLabel,
  getWhatsAppStatusState,
} from '~/utils/whatsappInstanceStatus'

const props = defineProps<{
  routingMode: LinkRoutingMode
  whatsappInstanceId: number | null
  fallbackPhone: string
  prefillMessage: string
  destinationError?: string
  instances: WhatsAppInstance[]
}>()

defineEmits<{
  'update:routingMode': [val: LinkRoutingMode]
  'update:whatsappInstanceId': [val: number | null]
  'update:fallbackPhone': [val: string]
  'update:prefillMessage': [val: string]
}>()

const routingOptions = [
  { value: 'auto_pixel', label: 'Automático / Rodízio entre conexões deste Pixel' },
  { value: 'specific_instance', label: 'Escolher uma conexão WhatsApp específica deste Pixel' },
  { value: 'manual_phone', label: 'Usar apenas número fixo manual' },
]

const instanceOptions = computed(() =>
  props.instances.map((i) => ({
    value: i.id,
    label: formatInstanceOptionLabel(i),
  })),
)

const visibleInstances = computed(() => {
  if (props.routingMode === 'specific_instance') {
    return props.instances.filter((i) => i.id === props.whatsappInstanceId)
  }
  return props.instances
})
</script>

<style scoped>
.section-box { display: flex; flex-direction: column; gap: 0.75rem; padding: 0.875rem; background: var(--surface-soft-2); border: 1px solid var(--border-soft); border-radius: 10px; }
.box-title { font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text-strong); margin: 0; }
.conn-list { display: flex; flex-direction: column; gap: 0.375rem; }
.conn-item { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; padding: 0.45rem 0.65rem; background: var(--surface-deep); border: 1px solid var(--border-soft); border-radius: 8px; flex-wrap: wrap; }
.conn-left { display: flex; align-items: center; gap: 0.45rem; flex-wrap: wrap; min-width: 0; }
.conn-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.dot-connected { background: var(--success); }
.dot-connecting { background: var(--warning); }
.dot-disconnected { background: var(--text-3); }
.conn-name { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--text-strong); }
.conn-prov { font-size: var(--fs-xs); color: var(--text-2); background: var(--surface-soft); padding: 0.0625rem 0.4rem; border-radius: 4px; }
.conn-phone { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-2); }
.conn-badge { font-size: var(--fs-2xs); font-weight: var(--fw-semibold); text-transform: uppercase; letter-spacing: 0.04em; padding: 0.15rem 0.5rem; border-radius: 999px; }
.badge-connected { color: var(--success); background: var(--success-soft); }
.badge-connecting { color: var(--warning); background: var(--warning-soft); }
.badge-disconnected { color: var(--text-3); background: var(--surface-soft); }
.conn-empty { font-size: var(--fs-sm); color: var(--warning); padding: 0.5rem 0.65rem; background: var(--surface-deep); border: 1px dashed var(--border); border-radius: 8px; }
</style>
