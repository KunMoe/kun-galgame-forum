<script setup lang="ts">
defineProps<{ page: number }>()
</script>

<template>
  <span v-if="page > 1" class="sr-only">，第 {{ page }} 页</span>
</template>
