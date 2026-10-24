<script setup lang="ts">
const route = useRoute()
await navigateTo(
  `/topic?section=${encodeURIComponent((route.params as { section: string }).section)}`,
  { redirectCode: 301 }
)
</script>

<template>
  <div />
</template>
