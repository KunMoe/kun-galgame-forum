<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 12 })
</script>

<template>
  <div
    class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
    aria-hidden="true"
  >
    <KunSkeleton v-for="n in count" :key="n" height="7.5rem" rounded="lg" />
  </div>
</template>
