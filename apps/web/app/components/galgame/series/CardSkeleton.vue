<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 4 })
</script>

<template>
  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2" aria-hidden="true">
    <KunCard
      v-for="n in count"
      :key="n"
      :is-hoverable="false"
      :is-transparent="false"
      class-name="flex h-full flex-col overflow-hidden"
    >
      <KunSkeleton height="8rem" rounded="lg" class-name="mb-4" />
      <div class="px-1 pb-1">
        <KunSkeleton
          variant="text"
          width="60%"
          class-name="mb-2 text-xl font-bold"
        />
        <KunSkeleton variant="text" width="85%" class-name="mb-4 text-xs" />
        <KunSkeleton variant="text" width="6rem" class-name="text-sm" />
      </div>
    </KunCard>
  </div>
</template>
