<script setup lang="ts">
import { useLocalStorage } from '@vueuse/core'

const props = defineProps<{
  query: string
  placeholder: string
  historyKey: string
}>()
const emit = defineEmits<{ search: [term: string] }>()

const HISTORY_MAX = 8

const text = ref(props.query)
watch(
  () => props.query,
  (value) => {
    text.value = value
  }
)

const history = useLocalStorage<string[]>(
  `kungal-search-history:${props.historyKey}`,
  [],
  { initOnMounted: true }
)

const popover = useTemplateRef<{ close: () => void }>('popover')

const run = (term: string) => {
  if (term) {
    history.value = [term, ...history.value.filter((h) => h !== term)].slice(
      0,
      HISTORY_MAX
    )
  }
  text.value = term
  emit('search', term)
  popover.value?.close()
}

const forget = (term: string) => {
  history.value = history.value.filter((h) => h !== term)
}
</script>

<template>
  <KunPopover
    ref="popover"
    position="bottom-end"
    aria-label="搜索"
    inner-class="p-3 w-80 max-w-[calc(100vw-2rem)]"
  >
    <template #trigger>
      <KunTooltip :text="query ? `搜索: ${query}` : '搜索'" position="bottom">
        <button
          type="button"
          :class="filterPillSquareClass(!!query)"
          :aria-label="query ? `搜索: ${query}` : '搜索'"
        >
          <KunIcon name="lucide:search" class="size-4 text-inherit" />
        </button>
      </KunTooltip>
    </template>

    <form class="flex gap-2" @submit.prevent="run(text.trim())">
      <KunInput
        v-model="text"
        :placeholder="placeholder"
        aria-label="搜索"
        size="sm"
        is-clearable
        autofocus
        class="min-w-0 flex-1"
      />
      <KunButton type="submit" color="primary" size="sm">搜索</KunButton>
    </form>

    <div v-if="history.length" class="mt-3">
      <div class="flex items-center justify-between">
        <span class="text-default-400 text-xs">搜索历史</span>
        <KunButton variant="light" size="xs" @click="history = []">
          清空
        </KunButton>
      </div>
      <ul class="mt-1 flex flex-col">
        <li v-for="h in history" :key="h" class="flex items-center gap-1">
          <button
            type="button"
            class="hover:bg-default-100 text-default-600 flex min-w-0 flex-1 cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm"
            @click="run(h)"
          >
            <KunIcon
              name="lucide:history"
              class="text-default-400 size-4 shrink-0"
            />
            <span class="truncate">{{ h }}</span>
          </button>
          <button
            type="button"
            class="text-default-400 hover:text-danger cursor-pointer rounded-md p-1.5"
            :aria-label="`删除搜索历史: ${h}`"
            @click="forget(h)"
          >
            <KunIcon name="lucide:x" class="size-3.5" />
          </button>
        </li>
      </ul>
    </div>
  </KunPopover>
</template>
