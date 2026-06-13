<template>
  <PagePlaceholder
    title="Eventos"
    description="Histórico de eventos enviados para a Meta CAPI com status, payload, resposta e filtros por pixel, tipo e período."
  />
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'app',
  middleware: 'auth',
})
</script>
