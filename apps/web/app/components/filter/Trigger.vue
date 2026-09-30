<script setup lang="ts">
withDefaults(
  defineProps<{
    icon: string
    label: string
    /** Replaces the label once the dimension has a value. */
    value?: string
    active?: boolean
    open?: boolean
  }>(),
  { value: '', active: false, open: false }
)
</script>

<template>
  <button
    type="button"
    :class="
      cn(
        'flex shrink-0 cursor-pointer items-center gap-2 rounded-full px-3.5 py-1.5 text-sm',
        filterPillClass(active)
      )
    "
  >
    <KunIcon :name="icon" class="shrink-0" />
    <span class="max-w-40 truncate">{{ value || label }}</span>
    <KunIcon
      name="lucide:chevron-down"
      :class="cn('shrink-0 transition-transform', open && 'rotate-180')"
    />
  </button>
</template>
