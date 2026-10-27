<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 8 })
</script>

<template>
  <div class="divide-default-200 divide-y" aria-hidden="true">
    <div v-for="n in count" :key="n" class="flex items-center gap-3 px-2 py-3">
      <KunSkeleton variant="circle" width="1.25rem" height="1.25rem" />
      <div class="min-w-0 flex-1">
        <KunSkeleton variant="text" :width="n % 2 ? '75%' : '60%'" />
        <KunSkeleton variant="text" width="12rem" class-name="mt-1 text-xs" />
      </div>
    </div>
  </div>
</template>
