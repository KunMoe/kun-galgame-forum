<script setup lang="ts">
withDefaults(defineProps<{ count?: number; height?: string }>(), {
  count: 3,
  height: '8rem'
})
</script>

<template>
  <div class="space-y-3" aria-hidden="true">
    <KunSkeleton v-for="n in count" :key="n" :height="height" rounded="lg" />
  </div>
</template>
