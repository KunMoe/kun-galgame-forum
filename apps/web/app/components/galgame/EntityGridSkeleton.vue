<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 12 })
</script>

<template>
  <div
    class="grid grid-cols-2 gap-3 sm:grid-cols-2 sm:gap-3 lg:grid-cols-3 xl:grid-cols-4"
    aria-hidden="true"
  >
    <KunCard
      v-for="n in count"
      :key="n"
      :is-hoverable="false"
      :is-transparent="false"
    >
      <KunSkeleton variant="text" width="70%" class-name="font-semibold" />
      <KunSkeleton variant="text" width="45%" class-name="text-xs" />
    </KunCard>
  </div>
</template>
