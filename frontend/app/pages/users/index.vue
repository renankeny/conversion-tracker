<template>
  <PagePlaceholder
    title="Usuários Rastreados"
    description="Lista de visitantes identificados com external_id, GeoIP, UTMs e dados de contato."
  />
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'app',
  middleware: 'auth',
})
</script>
