<script setup lang="ts">
defineProps<{ amount: number; className?: string }>()
</script>

<template>
  <span
    :class="
      cn(
        'text-foreground inline-flex items-baseline gap-1 font-semibold tracking-tight tabular-nums',
        className
      )
    "
  >
    {{ amount }}
    <span class="text-default-500 text-xs font-normal tracking-normal">
      萌萌点
    </span>
  </span>
</template>
