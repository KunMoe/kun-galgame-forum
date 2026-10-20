<script setup lang="ts">
const route = useRoute()
const { workEditUrl } = useNextmoeEditor()
const workId = String(route.params.id)

await navigateTo(workEditUrl(workId, `/galgame/${workId}`), {
  external: true,
  redirectCode: 301
})
</script>

<template>
  <div />
</template>
