<script setup lang="ts">
withDefaults(defineProps<{ rows?: number }>(), { rows: 5 })
</script>

<template>
  <div aria-hidden="true">
    <div v-for="n in rows" :key="n" class="space-y-2 p-2">
      <div class="flex items-center gap-2">
        <KunSkeleton variant="circle" width="1.125rem" height="1.125rem" />
        <KunSkeleton variant="text" :width="n % 2 ? '55%' : '40%'" />
      </div>
      <KunSkeleton variant="text" width="85%" class-name="text-sm" />
    </div>
  </div>
</template>
