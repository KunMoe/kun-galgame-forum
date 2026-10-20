<script setup lang="ts">
const route = useRoute()
const { editorUrl } = useNextmoeEditor()

await navigateTo(editorUrl(`/review/${String(route.params.id)}`, '/'), {
  external: true,
  redirectCode: 301
})
</script>

<template>
  <div />
</template>
