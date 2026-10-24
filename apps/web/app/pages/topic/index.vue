<script setup lang="ts">
const route = useRoute()
const active = computed(() =>
  typeof route.query.section === 'string' ? route.query.section : ''
)
</script>

<template>
  <TopicShell :active="active">
    <TopicContainer :section-key="active" />
  </TopicShell>
</template>
