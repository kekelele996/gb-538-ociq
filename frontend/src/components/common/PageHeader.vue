<script setup lang="ts">
defineProps<{ eyebrow: string; title: string; description: string }>()
</script>

<template>
  <header class="page-header">
    <div><p class="eyebrow">{{ eyebrow }}</p><h1>{{ title }}</h1><p>{{ description }}</p></div>
    <div class="page-actions"><slot /></div>
  </header>
</template>
