<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    :width="size || 15"
    :height="size || 15"
    fill="currentColor"
    aria-hidden="true"
  >
    <path :d="siWhatsapp.path" />
  </svg>
</template>

<script setup lang="ts">
import { siWhatsapp } from 'simple-icons'

withDefaults(
  defineProps<{
    size?: number | string
  }>(),
  {
    size: 15,
  },
)
</script>

