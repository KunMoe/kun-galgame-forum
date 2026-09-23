<script setup lang="ts">
import { KUN_GALGAME_LIBRARY_SORT_FIELD_MAP } from '~/constants/galgame'

withDefaults(defineProps<{ total?: number | null; pending?: boolean }>(), {
  total: null,
  pending: false
})

const { page, sortField, sortOrder, releasedFrom, releasedTo } =
  useLibraryFilters()

const showDisplay = ref(false)

watch(
  () => [
    sortField.value,
    sortOrder.value,
    releasedFrom.value,
    releasedTo.value
  ],
  () => {
    page.value = 1
  }
)

const sortOptions = Object.entries(KUN_GALGAME_LIBRARY_SORT_FIELD_MAP).map(
  ([value, label]) => ({ value, label })
)

// Only the release-date sort reads the direction; the catalog's popularity and
// updated cursors are descending and have no ascending counterpart.
const isOrderable = computed(() => sortField.value === 'release_date')

const yearRangeLabel = computed(() => {
  if (releasedFrom.value && releasedTo.value) {
    return releasedFrom.value === releasedTo.value
      ? `${releasedFrom.value} 年`
      : `${releasedFrom.value} - ${releasedTo.value}`
  }
  return releasedFrom.value
    ? `${releasedFrom.value} 年至今`
    : `${releasedTo.value} 年以前`
})

const setYears = (range: { from: string; to: string }) => {
  releasedFrom.value = range.from
  releasedTo.value = range.to
}

const chips = computed<FilterChip[]>(() =>
  releasedFrom.value || releasedTo.value
    ? [{ key: 'years', label: yearRangeLabel.value }]
    : []
)
</script>

<template>
  <div class="space-y-2">
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
        :model-value="sortField"
        empty-value="popularity"
        @update:model-value="sortField = $event as typeof sortField"
      />

      <KunTooltip
        v-if="isOrderable"
        :text="sortOrder === 'desc' ? '当前降序' : '当前升序'"
        position="bottom"
      >
        <button
          type="button"
          aria-label="切换排序方向"
          :class="filterPillSquareClass(false)"
          @click="sortOrder = sortOrder === 'desc' ? 'asc' : 'desc'"
        >
          <KunIcon
            :name="
              sortOrder === 'desc' ? 'lucide:arrow-down' : 'lucide:arrow-up'
            "
            class="size-4 text-inherit"
          />
        </button>
      </KunTooltip>

      <FilterYears :from="releasedFrom" :to="releasedTo" @update="setYears" />

      <template #end>
        <button
          type="button"
          :class="filterPillSquareClass(showDisplay)"
          aria-label="显示设置"
          @click="showDisplay = !showDisplay"
        >
          <KunIcon name="lucide:layout-grid" class="size-4 text-inherit" />
        </button>
      </template>
    </FilterBar>

    <GalgameCardDisplaySettings v-if="showDisplay" />
  </div>
</template>
