<template>
  <div class="sidebar-group" :class="{ open: isOpen }">
    <button
      type="button"
      class="group-toggle"
      :class="{ 'has-active-child': hasActiveChild }"
      @click="toggle"
    >
      <span class="link-icon" v-html="icon" />
      <span class="link-label">{{ label }}</span>
      <svg
        class="chevron"
        :class="{ rotated: isOpen }"
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path
          fill-rule="evenodd"
          d="M5.23 7.21a.75.75 0 011.06.02L10 11.085l3.71-3.855a.75.75 0 111.08 1.04l-4.25 4.4a.75.75 0 01-1.08 0l-4.25-4.4a.75.75 0 01.02-1.06z"
          clip-rule="evenodd"
        />
      </svg>
    </button>

    <Transition name="collapse">
      <div v-show="isOpen" class="group-children">
        <slot />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
interface Props {
  label: string
  icon: string
  basePath: string
}

const props = defineProps<Props>()

const route = useRoute()

const hasActiveChild = computed(() =>
  route.path === props.basePath || route.path.startsWith(props.basePath + '/')
)

// Mantém o submenu aberto se uma rota filha estiver ativa.
const isOpen = ref(hasActiveChild.value)

const toggle = () => {
  isOpen.value = !isOpen.value
}

watch(hasActiveChild, (active) => {
  if (active) isOpen.value = true
})
</script>

<style scoped>
.sidebar-group {
  display: flex;
  flex-direction: column;
}

.group-toggle {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0.875rem;
  border-radius: 8px;
  background: transparent;
  border: none;
  color: var(--text-2);
  font-size: 0.9375rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
  width: 100%;
}

.group-toggle:hover {
  background: var(--surface-soft);
  color: var(--text-1);
}

.group-toggle.has-active-child {
  color: var(--text-1);
}

.link-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.link-icon :deep(svg) {
  width: 1.125rem;
  height: 1.125rem;
}

.link-label {
  flex: 1;
}

.chevron {
  width: 1rem;
  height: 1rem;
  transition: transform 0.2s ease;
  flex-shrink: 0;
  opacity: 0.6;
}

.chevron.rotated {
  transform: rotate(180deg);
}

.group-children {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  padding: 0.25rem 0 0.25rem 1.625rem;
  margin-top: 0.125rem;
}

.collapse-enter-active,
.collapse-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}

.collapse-enter-from,
.collapse-leave-to {
  max-height: 0;
  opacity: 0;
}

.collapse-enter-to,
.collapse-leave-from {
  max-height: 500px;
  opacity: 1;
}
</style>
