<script setup lang="ts">
const route = useRoute()

await navigateTo(
  {
    path: taxonomyDetailPath(
      'official',
      Number((route.params as { id: string }).id)
    ),
    query: route.query
  },
  { redirectCode: 301, replace: true }
)
</script>

<template>
  <div />
</template>
