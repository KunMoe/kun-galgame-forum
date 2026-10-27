<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 3 })
</script>

<template>
  <div class="space-y-8" aria-hidden="true">
    <ActivityCardSkeleton v-for="n in count" :key="n" />
  </div>
</template>
