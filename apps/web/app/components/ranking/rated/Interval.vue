<script setup lang="ts">
const props = defineProps<{
  lower: number
  upper: number
  value: number
  min: number
  max: number
}>()

const at = (v: number) => {
  const span = props.max - props.min
  return span > 0
    ? Math.min(100, Math.max(0, ((v - props.min) / span) * 100))
    : 0
}

const band = computed(() => {
  const left = at(props.lower)
  return { left: `${left}%`, width: `${Math.max(at(props.upper) - left, 2)}%` }
})
</script>

<template>
  <div class="bg-default-200 relative h-1.5 w-full rounded-full">
    <div class="bg-primary/35 absolute inset-y-0 rounded-full" :style="band" />
    <div
      class="bg-primary absolute top-1/2 size-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full"
      :style="{ left: `${at(value)}%` }"
    />
  </div>
</template>
