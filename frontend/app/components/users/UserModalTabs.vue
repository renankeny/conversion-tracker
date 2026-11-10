<template>
  <div class="tab-nav-bar">
    <button
      type="button"
      class="tab-nav-btn"
      :class="{ active: modelValue === 'journey' }"
      @click="emit('update:modelValue', 'journey')"
    >
      <Activity class="w-3.5 h-3.5" />
      <span>Jornada & Vendas</span>
      <span v-if="timelineCount" class="tab-pill">{{ timelineCount }}</span>
    </button>

    <button
      type="button"
      class="tab-nav-btn"
      :class="{ active: modelValue === 'whatsapp' }"
      @click="emit('update:modelValue', 'whatsapp')"
    >
      <MessageCircle class="w-3.5 h-3.5" />
      <span>WhatsApp</span>
    </button>

    <button
      type="button"
      class="tab-nav-btn"
      :class="{ active: modelValue === 'profile' }"
      @click="emit('update:modelValue', 'profile')"
    >
      <UserCheck class="w-3.5 h-3.5" />
      <span>Dados & Contato</span>
    </button>

    <button
      type="button"
      class="tab-nav-btn"
      :class="{ active: modelValue === 'attribution' }"
      @click="emit('update:modelValue', 'attribution')"
    >
      <Target class="w-3.5 h-3.5" />
      <span>Origem & Atribuição</span>
    </button>

    <button
      type="button"
      class="tab-nav-btn"
      :class="{ active: modelValue === 'technical' }"
      @click="emit('update:modelValue', 'technical')"
    >
      <Cpu class="w-3.5 h-3.5" />
      <span>Cookies & Técnico</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { Activity, MessageCircle, UserCheck, Target, Cpu } from 'lucide-vue-next'

export type UserModalTabKey = 'journey' | 'whatsapp' | 'profile' | 'attribution' | 'technical'

defineProps<{
  modelValue: UserModalTabKey
  timelineCount?: number
}>()

const emit = defineEmits<{
  'update:modelValue': [val: UserModalTabKey]
}>()
</script>

<style scoped>
.tab-nav-bar {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem;
  background: var(--surface-deep);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  overflow-x: auto;
}
.tab-nav-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  padding: 0.5rem 0.75rem;
  border-radius: 7px;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--text-3);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}
.tab-nav-btn:hover { color: var(--text-1); }
.tab-nav-btn.active {
  color: var(--brand);
  background: var(--surface-soft);
}
.tab-pill {
  font-size: var(--fs-2xs);
  padding: 0.0625rem 0.375rem;
  border-radius: 9999px;
  background: var(--brand-soft);
  color: var(--brand);
}
@media (max-width: 640px) {
  .tab-nav-bar { justify-content: flex-start; }
  .tab-nav-btn { flex: none; }
}
</style>

