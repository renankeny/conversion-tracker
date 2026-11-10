<template>
  <div class="page-header">
    <div class="header-titles">
      <h1 class="page-title">CRM & Funil de Vendas</h1>
      <p class="page-subtitle">Acompanhe leads, monitore o valor de negociações e dispare conversões por etapa</p>
    </div>
    <div class="header-actions">
      <PixelDropdown
        :model-value="selectedPixel"
        @update:model-value="emit('update:selectedPixel', $event)"
      />
      <BaseButton variant="primary" size="sm" @click="emit('create-lead')">
        <template #iconLeft>
          <Plus :size="16" />
        </template>
        Novo Lead
      </BaseButton>
      <BaseButton v-if="canManageStages" variant="outline" size="sm" @click="emit('create-stage')">
        Nova Coluna
      </BaseButton>
      <BaseButton v-if="canManageStages" variant="outline" size="sm" @click="emit('open-pipeline')">
        <template #iconLeft>
          <Settings :size="15" />
        </template>
        Configurar Funil
      </BaseButton>
      <BaseButton variant="outline" size="sm" @click="emit('refresh')">
        <template #iconLeft>
          <RefreshCw :size="16" :class="{ spin: isLoading }" />
        </template>
        Atualizar
      </BaseButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Settings, RefreshCw } from 'lucide-vue-next'

withDefaults(defineProps<{
  selectedPixel: string
  isLoading: boolean
  canManageStages?: boolean
}>(), {
  canManageStages: true,
})

const emit = defineEmits<{
  (e: 'update:selectedPixel', val: string): void
  (e: 'create-lead'): void
  (e: 'create-stage'): void
  (e: 'open-pipeline'): void
  (e: 'refresh'): void
}>()
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: wrap;
}

.header-titles {
  flex: 1;
  min-width: 280px;
}

.page-title {
  font-family: var(--font-heading);
  font-size: var(--fs-3xl);
  font-weight: var(--fw-bold);
  color: var(--text-strong);
  margin-bottom: 0.25rem;
}

.page-subtitle {
  color: var(--text-2);
  font-size: var(--fs-md);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  flex-wrap: wrap;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  100% {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
  }

  .header-titles {
    min-width: 0;
  }

  .page-title {
    font-size: var(--fs-2xl);
    line-height: 1.25;
  }

  .page-subtitle {
    font-size: var(--fs-base);
    line-height: 1.35;
  }

  .header-actions {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .header-actions > * {
    flex: 1 1 auto;
  }
}
</style>
