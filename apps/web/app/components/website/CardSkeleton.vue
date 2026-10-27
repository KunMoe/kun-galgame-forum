<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 9 })
</script>

<template>
  <div
    class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3"
    aria-hidden="true"
  >
    <KunSkeleton v-for="n in count" :key="n" height="9.5rem" rounded="lg" />
  </div>
</template>
