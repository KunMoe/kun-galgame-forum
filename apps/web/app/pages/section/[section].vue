<script setup lang="ts">
const route = useRoute()
const { sectionOf } = useTopicSections()
const key = (route.params as { section: string }).section
await navigateTo(
  `/topic?section=${encodeURIComponent(sectionOf(key)?.section ?? key)}`,
  { redirectCode: 301 }
)
</script>

<template>
  <div />
</template>
