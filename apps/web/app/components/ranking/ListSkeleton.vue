<script setup lang="ts">
withDefaults(defineProps<{ count?: number; height?: string }>(), {
  count: 10,
  height: '4.5rem'
})
</script>

<template>
  <div class="space-y-3" aria-hidden="true">
    <KunSkeleton v-for="n in count" :key="n" :height="height" rounded="lg" />
  </div>
</template>
