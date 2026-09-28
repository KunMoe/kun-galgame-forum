<script setup lang="ts">
const props = defineProps<{ name: string }>()

const face = computed(() => {
  const m = props.name.match(/(\d[\d,]*)\s*(円|日元|元)/)
  if (!m || m.index === undefined) return null
  return {
    brand: props.name.slice(0, m.index).trim(),
    value: m[1],
    unit: m[2],
    label: props.name.slice(m.index + m[0].length).trim()
  }
})
</script>

<template>
  <div class="max-w-full text-center">
    <template v-if="face">
      <p v-if="face.brand" class="text-default-500 truncate text-sm">
        {{ face.brand }}
      </p>
      <p
        class="text-foreground mt-1 flex items-baseline justify-center gap-0.5 leading-none font-semibold tabular-nums"
      >
        <span class="text-5xl tracking-tight">{{ face.value }}</span>
        <span class="text-lg">{{ face.unit }}</span>
      </p>
      <p v-if="face.label" class="text-default-500 mt-2 truncate text-sm">
        {{ face.label }}
      </p>
    </template>
    <p v-else class="text-foreground line-clamp-3 text-lg font-semibold">
      {{ name }}
    </p>
  </div>
</template>
