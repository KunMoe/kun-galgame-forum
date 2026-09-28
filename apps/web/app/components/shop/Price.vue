<script setup lang="ts">
defineProps<{ amount: number; className?: string }>()
</script>

<template>
  <span
    :class="
      cn(
        'text-foreground inline-flex items-center gap-1 font-semibold tabular-nums',
        className
      )
    "
  >
    <KunIcon name="lucide:lollipop" class="text-secondary size-[1.05em]" />
    {{ amount }}
  </span>
</template>
