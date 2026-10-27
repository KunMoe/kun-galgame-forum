<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 12 })
</script>

<template>
  <div
    class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6"
    aria-hidden="true"
  >
    <KunCard
      v-for="n in count"
      :key="n"
      :is-hoverable="false"
      :is-transparent="false"
      padding="none"
      class-name="overflow-hidden"
    >
      <KunSkeleton height="auto" rounded="none" class-name="aspect-[3/4]" />
      <div class="space-y-0.5 p-2">
        <KunSkeleton variant="text" width="75%" class-name="text-sm" />
        <KunSkeleton variant="text" width="50%" class-name="text-xs" />
      </div>
    </KunCard>
  </div>
</template>
