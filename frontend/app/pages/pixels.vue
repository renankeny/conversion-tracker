<template>
  <!-- Redirect para integracoes/pixels -->
  <div />
</template>

<script setup lang="ts">
// Redireciona para a página de pixels que fica em /integracoes/pixels
const router = useRouter()

onMounted(() => {
  router.push('/integracoes/pixels')
})
</script>
