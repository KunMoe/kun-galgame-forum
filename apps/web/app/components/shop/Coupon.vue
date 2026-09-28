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
  <div class="bg-content1 shadow-kun-sm flex w-60 rounded-xl">
    <div class="min-w-0 flex-1 px-5 py-4">
      <template v-if="face">
        <p
          v-if="face.brand"
          class="text-default-500 truncate text-xs font-medium tracking-wide"
        >
          {{ face.brand }}
        </p>
        <p class="text-warning-600 mt-1 flex items-baseline gap-1 leading-none">
          <span class="text-4xl font-bold tracking-tight tabular-nums">
            {{ face.value }}
          </span>
          <span class="text-base font-semibold">{{ face.unit }}</span>
        </p>
        <p v-if="face.label" class="text-default-500 mt-2 truncate text-xs">
          {{ face.label }}
        </p>
      </template>
      <p v-else class="text-foreground line-clamp-3 text-sm font-semibold">
        {{ name }}
      </p>
    </div>
    <div
      class="border-warning-200 relative flex w-14 shrink-0 items-center justify-center border-l-2 border-dashed"
    >
      <span
        class="bg-warning-50 absolute -top-2 -left-[9px] size-4 rounded-full"
      />
      <span
        class="bg-warning-50 absolute -bottom-2 -left-[9px] size-4 rounded-full"
      />
      <KunIcon name="lucide:ticket-percent" class="text-warning-500 size-6" />
    </div>
  </div>
</template>
