<template>
  <div class="image-field">
    <div class="field-header">
      <div>
        <span class="field-label">{{ label }}</span>
        <p v-if="hint" class="field-hint">{{ hint }}</p>
      </div>
      <BaseButton
        v-if="modelValue"
        variant="ghost"
        size="sm"
        type="button"
        @click="emit('update:modelValue', '')"
      >
        <span class="text-danger">Remover</span>
      </BaseButton>
    </div>

    <div class="field-body" :class="[previewBgClass]">
      <div class="preview-box" :class="[previewBgClass]">
        <img
          v-if="modelValue"
          :src="modelValue"
          :alt="label"
          class="preview-img"
        />
        <span v-else class="preview-empty">Sem imagem</span>
      </div>

      <div class="field-controls">
        <div class="upload-row">
          <input
            ref="fileInput"
            type="file"
            accept="image/png,image/svg+xml,image/webp,image/jpeg,image/x-icon"
            class="hidden-file"
            @change="onFileSelect"
          />
          <BaseButton
            variant="outline"
            size="sm"
            type="button"
            :loading="optimizing"
            @click="fileInput?.click()"
          >
            Enviar arquivo
          </BaseButton>
          <span class="upload-note">PNG, SVG, WebP ou JPG (salvo no banco)</span>
        </div>

        <BaseInput
          :model-value="isDataUrl ? '' : modelValue"
          :placeholder="isDataUrl ? 'Imagem enviada (salva no banco de dados)' : 'Ou cole uma URL externa (https://...)'"
          @update:model-value="emit('update:modelValue', String($event))"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { optimizeBrandingImage } from '~/utils/imageOptimizer'

const props = withDefaults(
  defineProps<{
    modelValue: string
    label: string
    hint?: string
    kind?: 'logo' | 'favicon'
    previewTheme?: 'dark' | 'light' | 'neutral'
  }>(),
  {
    hint: '',
    kind: 'logo',
    previewTheme: 'neutral',
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const toast = useToast()
const fileInput = ref<HTMLInputElement | null>(null)
const optimizing = ref(false)

const isDataUrl = computed(() => props.modelValue?.startsWith('data:image/'))

const previewBgClass = computed(() => {
  if (props.previewTheme === 'dark') return 'preview-dark'
  if (props.previewTheme === 'light') return 'preview-light'
  return 'preview-neutral'
})

const onFileSelect = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  optimizing.value = true
  try {
    const dataUrl = await optimizeBrandingImage(file, props.kind)
    emit('update:modelValue', dataUrl)
    toast.success('Imagem processada', 'Otimizada e pronta para salvar no banco')
  } catch (err: any) {
    toast.error('Erro na imagem', err?.message || 'Não foi possível processar a imagem')
  } finally {
    optimizing.value = false
    input.value = ''
  }
}
</script>

<style scoped>
.image-field {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
  padding: 1rem;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  background: var(--surface-soft-2);
}

.field-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.5rem;
}

.field-label {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}

.field-hint {
  font-size: var(--fs-sm);
  color: var(--text-3);
  margin-top: 0.125rem;
}

.field-body {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.preview-box {
  width: 120px;
  height: 60px;
  border-radius: 8px;
  border: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.5rem;
  flex-shrink: 0;
  overflow: hidden;
}

.preview-dark {
  background: var(--surface-deep);
}

.preview-light {
  background: var(--surface-1);
}

.preview-neutral {
  background: var(--surface-2);
}

.preview-img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.preview-empty {
  font-size: var(--fs-xs);
  color: var(--text-3);
}

.field-controls {
  flex: 1;
  min-width: 220px;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.upload-row {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  flex-wrap: wrap;
}

.upload-note {
  font-size: var(--fs-sm);
  color: var(--text-3);
}

.hidden-file {
  display: none;
}

.text-danger {
  color: var(--danger);
}
</style>

