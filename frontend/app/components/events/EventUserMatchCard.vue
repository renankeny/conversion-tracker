<template>
  <div class="user-strip">
    <div class="user-strip-left">
      <div class="user-avatar">
        <span v-if="customerInitials">{{ customerInitials }}</span>
        <User v-else class="w-4 h-4 opacity-75" />
      </div>

      <div class="user-info">
        <div class="user-name-line">
          <span class="user-name" :title="displayName">{{ displayName }}</span>
          <span v-if="customerName" class="badge-identified">Identificado</span>
        </div>

        <div v-if="externalId" class="user-id-line">
          <span class="id-label">ID:</span>
          <span class="id-val font-mono">{{ externalId }}</span>
          <button
            type="button"
            class="id-copy-action"
            :class="{ copied: copied }"
            :title="copied ? 'ID Copiado!' : 'Copiar ID do usuário'"
            @click="copyExternalId"
          >
            <Check v-if="copied" class="w-3 h-3 text-emerald-400" />
            <Copy v-else class="w-3 h-3" />
            <span>{{ copied ? 'Copiado' : 'Copiar' }}</span>
          </button>
        </div>
      </div>
    </div>

    <button
      v-if="showUserButton"
      type="button"
      class="btn-view-profile"
      @click="$emit('openProfile')"
    >
      <span>Ver perfil do usuário</span>
      <ArrowUpRight class="w-3.5 h-3.5" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Copy, Check, ArrowUpRight, User } from 'lucide-vue-next'
import type { TrackedUser } from '#shared/types'

interface Props {
  externalId?: string
  trackedUser: TrackedUser | null
  allowUserDetails?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  externalId: '',
  allowUserDetails: true,
})

defineEmits<{
  openProfile: []
}>()

const copied = ref(false)

const customerName = computed(() => {
  const u = props.trackedUser
  if (!u) return ''
  return [u.first_name, u.last_name].filter(Boolean).join(' ').trim()
})

const displayName = computed(() => {
  if (customerName.value) return customerName.value
  const u = props.trackedUser
  if (u?.email) return u.email
  if (u?.phone) return u.phone
  return 'Visitante'
})

const customerInitials = computed(() => {
  if (customerName.value) {
    const parts = customerName.value.trim().split(/\s+/).filter(Boolean)
    if (parts.length === 1) return (parts[0]?.substring(0, 2) || '').toUpperCase()
    const first = parts[0]?.charAt(0) || ''
    const last = parts[parts.length - 1]?.charAt(0) || ''
    return (first + last).toUpperCase()
  }
  const u = props.trackedUser
  if (u?.email) return u.email.substring(0, 2).toUpperCase()
  return ''
})

const showUserButton = computed(() => props.allowUserDetails && !!props.trackedUser)

const copyExternalId = async () => {
  if (!props.externalId) return
  try {
    await navigator.clipboard.writeText(props.externalId)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    /* silent */
  }
}
</script>

<style scoped>
.user-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 1rem;
  background: var(--surface-soft-2);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
}

.user-strip-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}

.user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  color: var(--brand);
  font-weight: var(--fw-bold);
  font-size: var(--fs-base);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.user-info {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
}

.user-name-line {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.user-name {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.badge-identified {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  color: var(--brand);
  background: var(--brand-soft);
  padding: 0.0625rem 0.375rem;
  border-radius: 4px;
}

.user-id-line {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: var(--fs-xs);
  color: var(--text-3);
}

.id-label { font-weight: var(--fw-semibold); color: var(--text-3); }
.id-val { color: var(--text-2); }

.id-copy-action {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.1rem 0.35rem;
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  border-radius: 4px;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  color: var(--text-3);
  cursor: pointer;
  transition: all 0.15s ease;
}

.id-copy-action:hover { border-color: var(--brand); color: var(--brand); }
.id-copy-action.copied { color: var(--success); border-color: var(--success); }

.btn-view-profile {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  padding: 0.4rem 0.75rem;
  border-radius: 8px;
  background: var(--brand-soft);
  border: 1px solid var(--brand-border);
  color: var(--brand);
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-view-profile:hover {
  background: var(--brand);
  color: #111;
  border-color: var(--brand);
}

@media (max-width: 640px) {
  .user-strip { flex-direction: column; align-items: flex-start; }
  .btn-view-profile { width: 100%; justify-content: center; }
}
</style>
