<script setup lang="ts">
defineProps<{
  icon: string
  label: string
  active?: boolean
  disabled?: boolean
}>()
const emit = defineEmits<{ click: [] }>()
</script>

<template>
  <KunTooltip :text="label" hide-on-mobile>
    <KunButton
      variant="light"
      size="sm"
      is-icon-only
      :aria-label="label"
      :aria-pressed="active"
      :disabled="disabled"
      :class="
        cn('size-8 min-w-8', active ? 'text-primary' : 'text-default-500')
      "
      @click="emit('click')"
    >
      <KunIcon :name="icon" :class="cn('size-4', active && 'icon-solid')" />
    </KunButton>
  </KunTooltip>
</template>
