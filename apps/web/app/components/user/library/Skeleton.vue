<script setup lang="ts">
withDefaults(defineProps<{ part?: 'all' | 'stats' }>(), { part: 'all' })
</script>

<template>
  <div class="space-y-3" aria-hidden="true">
    <div class="grid grid-cols-3 gap-3">
      <KunSkeleton v-for="n in 3" :key="n" height="4.75rem" rounded="lg" />
    </div>
    <template v-if="part === 'all'">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap gap-2">
          <KunSkeleton
            v-for="n in 5"
            :key="n"
            height="1.75rem"
            width="4rem"
            rounded="full"
          />
        </div>
        <KunSkeleton height="2rem" width="8rem" rounded="lg" />
      </div>
      <GalgameCardGridSkeleton />
    </template>
  </div>
</template>
