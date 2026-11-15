<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
await navigateTo('/settings/webhooks', { replace: true })
</script>

<template><div /></template>
