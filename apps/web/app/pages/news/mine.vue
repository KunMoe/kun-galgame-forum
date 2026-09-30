<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('我的情报投稿')
</script>

<template>
  <NewsMine />
</template>
