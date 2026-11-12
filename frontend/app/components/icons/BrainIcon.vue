<template>
  <Brain :size="size || 15" :stroke-width="2" aria-hidden="true" />
</template>

<script setup lang="ts">
import { Brain } from 'lucide-vue-next'

withDefaults(
  defineProps<{
    size?: number | string
  }>(),
  {
    size: 15,
  },
)
</script>

