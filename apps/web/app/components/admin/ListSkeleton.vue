<script setup lang="ts">
withDefaults(defineProps<{ rows?: number }>(), { rows: 4 })
</script>

<template>
  <div class="space-y-3">
    <KunCard
      v-for="n in rows"
      :key="n"
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-2"
    >
      <KunSkeleton variant="text" width="45%" />
      <KunSkeleton variant="text" width="80%" class-name="text-sm" />
      <KunSkeleton variant="text" width="30%" class-name="text-xs" />
    </KunCard>
  </div>
</template>
