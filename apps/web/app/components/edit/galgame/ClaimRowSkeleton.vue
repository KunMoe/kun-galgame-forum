<script setup lang="ts">
withDefaults(defineProps<{ rows?: number }>(), { rows: 4 })
</script>

<template>
  <div class="flex flex-col gap-3">
    <div
      v-for="n in rows"
      :key="n"
      class="flex flex-col gap-3 p-3 sm:flex-row sm:items-center"
    >
      <div class="min-w-0 flex-1 space-y-1">
        <div class="flex items-center gap-2">
          <KunSkeleton variant="text" width="40%" class-name="text-lg" />
          <KunSkeleton height="1.25rem" width="3rem" rounded="full" />
        </div>
        <KunSkeleton variant="text" width="12rem" class-name="text-sm" />
      </div>
      <KunSkeleton height="2rem" width="4rem" rounded="lg" />
    </div>
  </div>
</template>
