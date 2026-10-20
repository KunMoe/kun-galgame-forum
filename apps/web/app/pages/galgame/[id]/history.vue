<script setup lang="ts">
const route = useRoute()
const { workHistoryUrl } = useNextmoeEditor()
const workId = String(route.params.id)

await navigateTo(workHistoryUrl(workId, `/galgame/${workId}`), {
  external: true,
  redirectCode: 301
})
</script>

<template>
  <div />
</template>
