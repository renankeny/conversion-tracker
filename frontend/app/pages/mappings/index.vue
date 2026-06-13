<template>
  <PagePlaceholder
    title="URL Mappings"
    description="Mapeie URLs externas (campanhas) para origem/medium/campaign internos para padronizar UTMs."
  />
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'app',
  middleware: 'auth',
})
</script>
