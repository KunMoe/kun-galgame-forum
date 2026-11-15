<script setup lang="ts">
definePageMeta({
  redirect: (to) => ({
    path: `/user/${String(to.params.id)}/library`,
    query: { sort: 'playtime_desc' }
  })
})
</script>

<template>
  <div />
</template>
