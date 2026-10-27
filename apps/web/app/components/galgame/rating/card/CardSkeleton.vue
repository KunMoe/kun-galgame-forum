<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 12 })
</script>

<template>
  <div
    class="grid grid-cols-2 gap-2 sm:grid-cols-2 sm:gap-3 lg:grid-cols-3 xl:grid-cols-4"
    aria-hidden="true"
  >
    <KunCard
      v-for="n in count"
      :key="n"
      :is-transparent="false"
      :is-hoverable="false"
    >
      <div class="flex justify-between">
        <KunSkeleton variant="circle" width="100px" height="100px" />
        <KunSkeleton height="1.75rem" width="3rem" rounded="md" />
      </div>
      <div class="flex flex-col gap-2">
        <KunSkeleton variant="text" width="80%" class-name="font-medium" />
        <KunSkeleton variant="text" width="60%" class-name="text-sm" />
        <KunSkeleton variant="text" width="40%" class-name="text-sm" />
      </div>
    </KunCard>
  </div>
</template>
