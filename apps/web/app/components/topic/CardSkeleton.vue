<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 6 })
</script>

<template>
  <div class="divide-default-200/60 divide-y" aria-hidden="true">
    <div v-for="n in count" :key="n" class="space-y-2 py-4 first:pt-0">
      <KunSkeleton width="70%" height="1.25rem" />
      <KunSkeleton width="6rem" height="1.25rem" rounded="full" />
      <div class="flex items-center gap-2">
        <KunSkeleton variant="circle" width="1.25rem" height="1.25rem" />
        <KunSkeleton width="40%" height="0.875rem" />
      </div>
    </div>
  </div>
</template>
