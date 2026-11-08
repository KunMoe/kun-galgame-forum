<script setup lang="ts">
import type { CollectionProgress } from '#shared/utils/api/schemas'

const props = defineProps<{ progress: CollectionProgress }>()

const percent = computed(() =>
  props.progress.item_count
    ? Math.round((props.progress.done_count / props.progress.item_count) * 100)
    : 0
)
const parts = computed(() =>
  [
    { label: '在玩', value: props.progress.doing_count },
    { label: '想玩', value: props.progress.wish_count },
    { label: '搁置', value: props.progress.on_hold_count },
    { label: '弃坑', value: props.progress.dropped_count },
    { label: '在别的收藏夹里', value: props.progress.collected_count }
  ].filter((p) => p.value > 0)
)
</script>

<template>
  <div class="bg-default-100 space-y-2 rounded-lg px-4 py-3">
    <div class="flex flex-wrap items-baseline justify-between gap-2">
      <span class="font-medium">
        我的进度：已玩
        <span class="text-primary text-lg font-semibold">
          {{ progress.done_count }}
        </span>
        / {{ progress.item_count }}
      </span>
      <span class="text-default-500 text-sm">{{ percent }}%</span>
    </div>
    <KunProgress :value="percent" size="sm" aria-label="已玩比例" />
    <div
      v-if="parts.length"
      class="text-default-500 flex flex-wrap gap-x-4 gap-y-1 text-sm"
    >
      <span v-for="p in parts" :key="p.label">{{ p.label }} {{ p.value }}</span>
    </div>
  </div>
</template>
