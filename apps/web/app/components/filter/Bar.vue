<script setup lang="ts">
withDefaults(
  defineProps<{
    chips: FilterChip[]
    total?: number | null
    /** Reads after 共 N. */
    unit?: string
    pending?: boolean
  }>(),
  { total: null, unit: '部', pending: false }
)

defineEmits<{ remove: [key: string]; clear: [] }>()
</script>

<template>
  <div class="border-default-200 bg-default-50 space-y-2 rounded-xl border p-2">
    <div class="flex flex-wrap items-center gap-2">
      <slot />

      <div class="ml-auto flex shrink-0 items-center gap-2">
        <slot name="end" />
        <span
          v-if="total !== null"
          class="text-default-500 px-1 text-sm tabular-nums"
          :class="pending && 'opacity-50'"
        >
          共 {{ total.toLocaleString('en-US') }} {{ unit }}
        </span>
      </div>
    </div>

    <div
      v-if="$slots.more"
      class="border-default-200 flex flex-wrap items-center gap-2 border-t pt-2"
    >
      <slot name="more" />
    </div>

    <div
      v-if="chips.length"
      class="border-default-200 flex flex-wrap items-center gap-1.5 border-t pt-2"
    >
      <KunChip
        v-for="chip in chips"
        :key="chip.key"
        size="sm"
        variant="flat"
        color="primary"
        closable
        @close="$emit('remove', chip.key)"
      >
        <span v-if="chip.prefix" class="opacity-60">{{ chip.prefix }}</span>
        {{ chip.label }}
      </KunChip>

      <button
        type="button"
        class="text-default-500 hover:text-danger ml-auto flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-sm transition-colors"
        @click="$emit('clear')"
      >
        <KunIcon name="lucide:rotate-ccw" class="size-4 text-inherit" />
        <span>清空筛选</span>
      </button>
    </div>
  </div>
</template>
