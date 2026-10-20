<script setup lang="ts">
const { editorUrl } = useNextmoeEditor()

await navigateTo(editorUrl('/me/proposals', '/'), {
  external: true,
  redirectCode: 301
})
</script>

<template>
  <div />
</template>
