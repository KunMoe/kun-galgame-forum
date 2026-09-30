<script setup lang="ts">
import { KUN_GALGAME_LIBRARY_SORT_FIELD_MAP } from '~/constants/galgame'

withDefaults(defineProps<{ total?: number | null; pending?: boolean }>(), {
  total: null,
  pending: false
})

const { state, set } = useLibraryFilters()

const { open: openSettingPanel } = useSettingPanel()

const sortOptions = Object.entries(KUN_GALGAME_LIBRARY_SORT_FIELD_MAP).map(
  ([value, label]) => ({ value, label })
)

// Only the release-date sort reads the direction; the catalog's popularity and
// updated cursors are descending and have no ascending counterpart.
const isOrderable = computed(() => state.value.sortField === 'release_date')

const yearRangeLabel = computed(() => {
  const { releasedFrom: from, releasedTo: to } = state.value
  if (from && to) {
    return from === to ? `${from} 年` : `${from} - ${to}`
  }
  return from ? `${from} 年至今` : `${to} 年以前`
})

const setYears = (range: { from: string; to: string }) => {
  set({ releasedFrom: range.from, releasedTo: range.to })
}

const chips = computed<FilterChip[]>(() =>
  state.value.releasedFrom || state.value.releasedTo
    ? [{ key: 'years', label: yearRangeLabel.value }]
    : []
)
</script>

<template>
  <FilterBar
    :chips="chips"
    :total="total"
    :pending="pending"
    @remove="setYears({ from: '', to: '' })"
    @clear="setYears({ from: '', to: '' })"
  >
    <FilterMenu
      icon="lucide:arrow-down-up"
      label="排序"
      :options="sortOptions"
      :model-value="state.sortField"
      empty-value="popularity"
      @update:model-value="set({ sortField: $event as string })"
    />

    <KunTooltip
      v-if="isOrderable"
      :text="state.sortOrder === 'desc' ? '当前降序' : '当前升序'"
      position="bottom"
    >
      <button
        type="button"
        aria-label="切换排序方向"
        :class="filterPillSquareClass(false)"
        @click="set({ sortOrder: state.sortOrder === 'desc' ? 'asc' : 'desc' })"
      >
        <KunIcon
          :name="
            state.sortOrder === 'desc' ? 'lucide:arrow-down' : 'lucide:arrow-up'
          "
          class="size-4 text-inherit"
        />
      </button>
    </KunTooltip>

    <FilterYears
      :from="state.releasedFrom"
      :to="state.releasedTo"
      @update="setYears"
    />

    <template #end>
      <KunTooltip text="卡片显示设置" position="bottom">
        <button
          type="button"
          :class="filterPillSquareClass(false)"
          aria-label="卡片显示设置"
          @click="openSettingPanel('galgame')"
        >
          <KunIcon name="lucide:layout-grid" class="size-4 text-inherit" />
        </button>
      </KunTooltip>
    </template>
  </FilterBar>
</template>
