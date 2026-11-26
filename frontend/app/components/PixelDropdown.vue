<template>
  <BaseDropdown
    :model-value="modelValue"
    :label="label"
    :options="pixelOptions"
    :disabled="pixelStore.loading"
    @update:model-value="emit('update:modelValue', String($event ?? ''))"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import BaseDropdown from './BaseDropdown.vue'

interface Props {
  modelValue?: string
  label?: string
  allowAll?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  label: '',
  allowAll: true,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const pixelStore = usePixelStore()

onMounted(() => {
  pixelStore.fetch()
})

const pixelOptions = computed(() => {
  const list = pixelStore.pixels?.map((p) => ({
    label: p.label ? `${p.label} (${p.name})` : p.name,
    value: p.name,
  })) ?? []
  if (props.allowAll) {
    return [{ label: 'Todos os pixels', value: '' }, ...list]
  }
  return list
})

watch(
  [() => props.modelValue, () => pixelStore.pixels, () => props.allowAll],
  ([modelValue, pixels, allowAll]) => {
    if (!pixels.length) return
    if (allowAll && !modelValue) return
    const exists = pixels.some((p) => p.name === modelValue)
    if (!exists) {
      emit('update:modelValue', allowAll ? '' : (pixels[0]?.name ?? ''))
    }
  },
  { immediate: true },
)
</script>
