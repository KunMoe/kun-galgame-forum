<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'

const props = defineProps<{
  active: string
}>()

const { sections } = useTopicSections()

const items = computed<KunTabItem[]>(() => [
  { value: '', textValue: '全部话题' },
  ...sections.value
    .filter((s) => !s.is_archived || s.section === props.active)
    .map((s) => ({ value: s.section, textValue: s.label }))
])

const go = (value: string | number) =>
  navigateTo(value ? `/section/${value}` : '/topic')
</script>

<template>
  <KunTab
    :model-value="active"
    :items="items"
    variant="pills"
    color="primary"
    size="sm"
    scrollable
    @update:model-value="go"
  />
</template>
