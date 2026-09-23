<script setup lang="ts">
import { SEARCH_ENTITY_FAMILY_MAP } from './items'

const props = defineProps<{
  group: SearchEntityGroup
  keywords: string
  showHeader?: boolean
  showCap?: boolean
}>()

const emit = defineEmits<{
  open: [family: SearchEntityFamily]
}>()

const meta = computed(() => SEARCH_ENTITY_FAMILY_MAP[props.group.family])

const isCapped = computed(
  () => props.showCap && props.group.total > props.group.items.length
)
</script>

<template>
  <section v-if="group.failed" class="space-y-3">
    <header v-if="showHeader" class="flex items-center gap-2">
      <KunIcon :name="meta.icon" class="text-default-500 size-4" />
      <h3 class="text-sm font-medium">{{ meta.textValue }}</h3>
    </header>
    <p class="text-default-400 text-sm">
      {{ meta.textValue }}搜索没能完成, 请稍后重试
    </p>
  </section>

  <section v-else-if="group.items.length" class="space-y-3">
    <header v-if="showHeader" class="flex items-center gap-2">
      <KunIcon :name="meta.icon" class="text-default-500 size-4" />
      <h3 class="text-sm font-medium">{{ meta.textValue }}</h3>
      <span class="text-default-400 text-xs tabular-nums">{{
        group.total
      }}</span>
    </header>

    <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
      <SearchEntityCard
        v-for="item in group.items"
        :key="`${item.family}-${item.id}`"
        :item="item"
        :keywords="keywords"
      />
    </div>

    <KunButton
      v-if="isCapped"
      variant="light"
      size="sm"
      icon-position="right"
      @click="emit('open', group.family)"
    >
      共 {{ group.total }} 条, 查看全部{{ meta.textValue }}
      <KunIcon name="lucide:arrow-right" class="size-4" />
    </KunButton>
  </section>
</template>
