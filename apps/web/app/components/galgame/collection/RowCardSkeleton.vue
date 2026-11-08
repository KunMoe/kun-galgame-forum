<script setup lang="ts">
withDefaults(defineProps<{ count?: number; columns?: 1 | 2 }>(), {
  count: 6,
  columns: 2
})
</script>

<template>
  <div
    :class="cn('grid grid-cols-1 gap-3', columns === 2 && 'md:grid-cols-2')"
    aria-hidden="true"
  >
    <div v-for="n in count" :key="n" class="flex gap-4 p-2">
      <KunSkeleton
        variant="rect"
        class-name="w-24 shrink-0 rounded-md sm:w-28"
        style="aspect-ratio: 5 / 7"
      />
      <div class="flex-1 space-y-2">
        <KunSkeleton variant="text" width="60%" class-name="text-lg" />
        <KunSkeleton variant="text" width="30%" />
        <KunSkeleton variant="text" width="85%" />
        <KunSkeleton variant="text" width="40%" />
      </div>
    </div>
  </div>
</template>
