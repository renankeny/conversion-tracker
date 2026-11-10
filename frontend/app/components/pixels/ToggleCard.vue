<template>
  <div
    class="toggle-card"
    :class="{ active: modelValue }"
    role="switch"
    :aria-checked="modelValue"
    tabindex="0"
    @click="emit('update:modelValue', !modelValue)"
    @keydown.space.prevent="emit('update:modelValue', !modelValue)"
    @keydown.enter.prevent="emit('update:modelValue', !modelValue)"
  >
    <div class="card-content">
      <div class="card-header">
        <span class="card-title">{{ title }}</span>
        <span v-if="tag" class="card-tag">{{ tag }}</span>
      </div>
      <p class="card-description">{{ description }}</p>
    </div>
    <div class="toggle" :class="{ checked: modelValue }">
      <span class="slider" />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  modelValue: boolean
  title: string
  description: string
  tag?: string
}

defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()
</script>

<style scoped>
.toggle-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  background: var(--surface-2);
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.toggle-card:hover {
  border-color: var(--border);
}

.toggle-card.active {
  border-color: var(--brand-border);
  background: var(--brand-soft-2);
}

.card-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.card-title {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}

.card-tag {
  font-size: var(--fs-xs);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-weight: var(--fw-semibold);
  background: var(--surface-soft);
  color: var(--text-2);
  padding: 0.0625rem 0.5rem;
  border-radius: 9999px;
  letter-spacing: 0.025em;
}

.toggle-card.active .card-tag {
  background: var(--brand-soft);
  color: var(--brand);
}

.card-description {
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.4;
}

.toggle {
  position: relative;
  width: 38px;
  height: 22px;
  flex-shrink: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: var(--border);
  border-radius: 9999px;
  transition: background 0.2s ease;
}

.slider::before {
  content: '';
  position: absolute;
  height: 16px;
  width: 16px;
  left: 3px;
  top: 3px;
  background: var(--brand-text);
  border-radius: 50%;
  transition: transform 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.toggle.checked .slider {
  background: var(--brand);
}

.toggle.checked .slider::before {
  transform: translateX(16px);
}

.toggle-card:focus-visible .slider {
  box-shadow: 0 0 0 3px var(--brand-soft);
}
</style>
