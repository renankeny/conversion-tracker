<template>
  <div class="placeholder">
    <div class="placeholder-icon">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M9 4.804A7.968 7.968 0 005.5 4c-1.255 0-2.443.29-3.5.804v10A7.969 7.969 0 015.5 14c1.669 0 3.218.51 4.5 1.385A7.962 7.962 0 0114.5 14c1.255 0 2.443.29 3.5.804v-10A7.968 7.968 0 0014.5 4c-1.255 0-2.443.29-3.5.804V12a1 1 0 11-2 0V4.804z" clip-rule="evenodd" />
      </svg>
    </div>
    <h2 class="placeholder-title">{{ title }}</h2>
    <p class="placeholder-description">{{ description }}</p>
    <p class="placeholder-hint">Em desenvolvimento</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description: string
}>()
</script>

<style scoped>
.placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 4rem 2rem;
  background: var(--surface-2);
  border: 1px dashed var(--border);
  border-radius: 12px;
  gap: 0.5rem;
}

.placeholder-icon {
  color: var(--text-3);
  margin-bottom: 0.5rem;
}

.placeholder-title {
  font-family: var(--font-sans);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}

.placeholder-description {
  color: var(--text-2);
  max-width: 420px;
}

.placeholder-hint {
  margin-top: 0.75rem;
  font-size: var(--fs-sm);
  color: var(--brand);
  background: var(--brand-soft);
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: var(--fw-semibold);
}
</style>
