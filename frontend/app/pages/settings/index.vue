<template>
  <PagePlaceholder
    title="Configurações"
    description="Configurações do sistema, gestão de admins, preferências de tema e integrações."
  />
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'app',
  middleware: 'auth',
})
</script>
