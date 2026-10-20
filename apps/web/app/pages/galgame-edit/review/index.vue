<script setup lang="ts">
const { editorUrl } = useNextmoeEditor()

await navigateTo(editorUrl('/review', '/'), {
  external: true,
  redirectCode: 301
})
</script>

<template>
  <div />
</template>
