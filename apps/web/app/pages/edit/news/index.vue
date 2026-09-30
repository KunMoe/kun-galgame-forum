<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('发布 Gal 情报')
</script>

<template>
  <EditNewsForm />
</template>
