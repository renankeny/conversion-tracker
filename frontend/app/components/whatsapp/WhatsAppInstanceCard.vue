<template>
  <article class="inst-card">
    <header class="inst-header">
      <div class="inst-identity">
        <img
          v-if="instance.profile_pic_url"
          :src="instance.profile_pic_url"
          alt=""
          class="inst-avatar-img"
        >
        <div v-else class="inst-avatar">
          <Smartphone :size="18" />
        </div>
        <div class="inst-titles">
          <h3 class="inst-name">{{ instance.name }}</h3>
          <p class="inst-sub">
            <span v-if="formattedPhone">{{ formattedPhone }}</span>
            <span v-else>Aguardando conexão do aparelho</span>
          </p>
        </div>
      </div>

      <BaseBadge :variant="statusBadgeVariant">{{ statusLabel }}</BaseBadge>
    </header>

    <div class="inst-meta">
      <div class="meta-row">
        <span class="meta-label">Pixel vinculado</span>
        <span class="meta-val">{{ instance.pixel_name || `#${instance.pixel_id}` }}</span>
      </div>
      <div v-if="instance.profile_name" class="meta-row">
        <span class="meta-label">Perfil WhatsApp</span>
        <span class="meta-val">{{ instance.profile_name }}</span>
      </div>
      <div class="meta-row">
        <span class="meta-label">CRM Automático</span>
        <span class="meta-val">{{ instance.auto_create_crm_lead ? 'Ativo' : 'Desativado' }}</span>
      </div>
    </div>

    <footer class="inst-footer">
      <div class="left-actions">
        <BaseButton
          v-if="!isConnected"
          variant="primary"
          size="sm"
          @click="$emit('connect', instance)"
        >
          <QrCode :size="14" />
          <span>Conectar</span>
        </BaseButton>
        <BaseButton
          v-else
          variant="outline"
          size="sm"
          :loading="disconnecting"
          @click="$emit('disconnect', instance)"
        >
          Desconectar
        </BaseButton>

        <BaseButton
          variant="ghost"
          size="sm"
          :loading="refreshing"
          title="Sincronizar status"
          @click="$emit('refresh', instance)"
        >
          <RefreshCw :size="14" />
        </BaseButton>
      </div>

      <div class="right-actions">
        <BaseButton variant="ghost" size="sm" @click="$emit('edit', instance)">
          Editar
        </BaseButton>
        <BaseButton variant="ghost" size="sm" @click="$emit('delete', instance)">
          Excluir
        </BaseButton>
      </div>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { Smartphone, QrCode, RefreshCw } from 'lucide-vue-next'
import type { WhatsAppInstance } from '#shared/types'

const props = defineProps<{
  instance: WhatsAppInstance
  refreshing?: boolean
  disconnecting?: boolean
}>()

defineEmits<{
  connect: [inst: WhatsAppInstance]
  disconnect: [inst: WhatsAppInstance]
  refresh: [inst: WhatsAppInstance]
  edit: [inst: WhatsAppInstance]
  delete: [inst: WhatsAppInstance]
}>()

const isConnected = computed(() => props.instance.connection_status === 'OPEN')

const statusBadgeVariant = computed(() => {
  if (!props.instance.enabled) return 'default'
  if (props.instance.connection_status === 'OPEN') return 'success'
  if (props.instance.connection_status === 'CONNECTING') return 'warning'
  return 'danger'
})

const statusLabel = computed(() => {
  if (!props.instance.enabled) return 'Pausada'
  if (props.instance.connection_status === 'OPEN') return 'Conectada'
  if (props.instance.connection_status === 'CONNECTING') return 'Pareando'
  return 'Desconectada'
})

const formattedPhone = computed(() => {
  const jid = props.instance.owner_jid || ''
  const digits = jid.split('@')[0]?.replace(/\D/g, '') || ''
  if (!digits) return ''
  return `+${digits}`
})
</script>

<style scoped>
.inst-card {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.125rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.inst-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 0.75rem; }
.inst-identity { display: flex; align-items: center; gap: 0.75rem; min-width: 0; }
.inst-avatar, .inst-avatar-img {
  width: 38px; height: 38px; border-radius: 9px; flex-shrink: 0; object-fit: cover;
}
.inst-avatar {
  background: var(--surface-soft); color: var(--brand);
  display: flex; align-items: center; justify-content: center; border: 1px solid var(--border-soft);
}
.inst-titles { min-width: 0; }
.inst-name {
  font-family: var(--font-heading); font-size: var(--fs-lg); font-weight: var(--fw-semibold);
  color: var(--text-strong); margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.inst-sub { font-size: var(--fs-sm); color: var(--text-3); margin: 0.125rem 0 0; }
.inst-meta {
  display: flex; flex-direction: column; gap: 0.375rem;
  padding: 0.75rem; background: var(--surface-deep); border: 1px solid var(--border-faint); border-radius: 8px;
}
.meta-row { display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; font-size: var(--fs-sm); }
.meta-label { color: var(--text-3); }
.meta-val { color: var(--text-1); font-weight: var(--fw-medium); text-align: right; }
.inst-footer {
  display: flex; justify-content: space-between; align-items: center; gap: 0.5rem;
  padding-top: 0.5rem; border-top: 1px solid var(--border-faint); flex-wrap: wrap;
}
.left-actions, .right-actions { display: flex; align-items: center; gap: 0.375rem; }
</style>

