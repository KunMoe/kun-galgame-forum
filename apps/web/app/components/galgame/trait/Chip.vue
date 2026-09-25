<script setup lang="ts">
import type { TraitSummary } from '#shared/utils/api/schemas'

const props = withDefaults(
  defineProps<{
    trait: TraitSummary
    pickMode?: boolean
    picked?: boolean
    isHeading?: boolean
  }>(),
  { pickMode: false, picked: false, isHeading: false }
)

defineEmits<{ pick: [] }>()

const name = computed(() => catalogVocabularyName(props.trait))
const count = computed(() =>
  props.trait.character_count.toLocaleString('en-US')
)
</script>

<template>
  <button
    v-if="pickMode && trait.is_searchable"
    type="button"
    class="cursor-pointer"
    :aria-pressed="picked"
    :title="`${picked ? '移除' : '加入'}筛选: ${name}`"
    @click="$emit('pick')"
  >
    <KunChip
      :size="isHeading ? 'sm' : 'xs'"
      :variant="picked ? 'solid' : 'flat'"
      :color="picked ? 'primary' : trait.is_sexual ? 'danger' : 'default'"
      :class-name="cn(isHeading && 'font-semibold')"
    >
      <KunIcon v-if="picked" name="lucide:check" class="size-3" />
      {{ name }}
      <span class="text-xs tabular-nums opacity-60">{{ count }}</span>
    </KunChip>
  </button>

  <KunChip
    v-else-if="pickMode"
    :size="isHeading ? 'sm' : 'xs'"
    variant="light"
    :class-name="cn(isHeading && 'font-semibold')"
  >
    {{ name }}
  </KunChip>

  <KunLink v-else underline="none" :to="`/galgame/trait/${trait.id}`">
    <KunChip
      :size="isHeading ? 'sm' : 'xs'"
      variant="flat"
      :color="trait.is_sexual ? 'danger' : isHeading ? 'primary' : 'default'"
      :class-name="cn('cursor-pointer', isHeading && 'font-semibold')"
    >
      {{ name }}
      <span class="text-xs tabular-nums opacity-60">{{ count }}</span>
    </KunChip>
  </KunLink>
</template>
