<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 5 })
</script>

<template>
  <div class="divide-default-200/60 divide-y" aria-hidden="true">
    <div v-for="n in count" :key="n" class="py-5 first:pt-0 last:pb-0">
      <ActivityCardSkeleton />
    </div>
  </div>
</template>
