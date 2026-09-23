<script setup lang="ts">
import { SEARCH_CATEGORIES } from './items'

const props = defineProps<{
  modelValue: SearchType
  totals: SearchOverviewTotals | null
  pending: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: SearchType]
}>()

// Partial, not Record: the community lane answers no total, so its count is
// absent rather than zero — and SearchNavCount draws nothing for absent.
const counts = computed<Partial<Record<SearchType, number>>>(() => {
  const totals = props.totals
  if (!totals) {
    return {}
  }
  const sum = Object.values(totals).reduce(
    (acc, value) => acc + (value ?? 0),
    0
  )
  return { ...totals, all: sum }
})

const countOf = (value: SearchType) => counts.value[value]

const handleSelect = (value: string) =>
  emit('update:modelValue', value as SearchType)
</script>

<template>
  <nav aria-label="搜索分类">
    <div class="hidden lg:block">
      <KunTab
        :items="SEARCH_CATEGORIES"
        :model-value="modelValue"
        orientation="vertical"
        variant="underlined"
        align="start"
        :full-width="true"
        @update:model-value="handleSelect"
      >
        <template #tab="{ item }">
          <span class="flex w-full items-center gap-2">
            <KunIcon :name="item.icon" class="size-4 shrink-0" />
            <span class="truncate">{{ item.textValue }}</span>
            <SearchNavCount
              :value="countOf(item.value as SearchType)"
              :pending="pending"
            />
          </span>
        </template>
      </KunTab>
    </div>

    <div class="lg:hidden">
      <KunTab
        :items="SEARCH_CATEGORIES"
        :model-value="modelValue"
        variant="light"
        size="sm"
        :scrollable="true"
        @update:model-value="handleSelect"
      >
        <template #tab="{ item }">
          <span class="flex items-center gap-1.5">
            <KunIcon :name="item.icon" class="size-4 shrink-0" />
            <span>{{ item.textValue }}</span>
            <SearchNavCount
              :value="countOf(item.value as SearchType)"
              :pending="pending"
              :inline="true"
            />
          </span>
        </template>
      </KunTab>
    </div>
  </nav>
</template>
