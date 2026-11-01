<script setup lang="ts">
import type { WorkFacets } from '#shared/utils/api/schemas'
import { KUN_GALGAME_LIBRARY_SORT_FIELD_MAP } from '~/constants/galgame'
import { TAG_FILTER_MAX } from '~/components/search/items'

const props = withDefaults(
  defineProps<{
    total?: number | null
    pending?: boolean
    facets?: WorkFacets
  }>(),
  { total: null, pending: false, facets: undefined }
)

const { state, set } = useLibraryFilters()

const { open: openSettingPanel } = useSettingPanel()

const entityNames = useEntityNames()
const suggestions = useEntityFacets(() => props.facets, entityNames)

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

const toggleTag = (item: SearchEntityItem) => {
  entityNames.remember(item)
  const tags = state.value.tags
  set({
    tags: tags.includes(item.id)
      ? tags.filter((id) => id !== item.id)
      : [...tags, item.id].slice(0, TAG_FILTER_MAX)
  })
}

const toggleCompany = (item: SearchEntityItem) => {
  entityNames.remember(item)
  set({ company: state.value.company === item.id ? 0 : item.id })
}

const chips = computed<FilterChip[]>(() => {
  const f = state.value
  const list: FilterChip[] = []
  if (f.q) {
    list.push({ key: 'q', prefix: '搜索', label: f.q })
  }
  for (const id of f.tags) {
    list.push({
      key: `tag:${id}`,
      prefix: '标签',
      label: entityNames.labelOf('tag', id)
    })
  }
  if (f.company) {
    list.push({
      key: 'company',
      prefix: '会社',
      label: entityNames.labelOf('company', f.company)
    })
  }
  if (f.releasedFrom || f.releasedTo) {
    list.push({ key: 'years', label: yearRangeLabel.value })
  }
  return list
})

const removeChip = (key: string) => {
  const [dimension, value] = key.split(':')
  if (dimension === 'q') {
    set({ q: '' })
  } else if (dimension === 'tag') {
    set({ tags: state.value.tags.filter((id) => id !== Number(value)) })
  } else if (dimension === 'company') {
    set({ company: 0 })
  } else if (dimension === 'years') {
    setYears({ from: '', to: '' })
  }
}

const clearFilters = () => {
  set({ q: '', tags: [], company: 0, releasedFrom: '', releasedTo: '' })
}

watch(
  [() => state.value.company, () => state.value.tags],
  ([company, tags]) => entityNames.resolve({ company: [company], tag: tags }),
  { immediate: true }
)
</script>

<template>
  <FilterBar
    :chips="chips"
    :total="total"
    :pending="pending"
    @remove="removeChip"
    @clear="clearFilters"
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

    <span class="bg-default-200 h-6 w-px" aria-hidden="true" />

    <FilterEntityMenu
      family="tag"
      icon="lucide:tag"
      label="标签"
      placeholder="搜索标签, 例如 校园"
      :selected-ids="state.tags"
      :selected-items="entityNames.itemsOf('tag', state.tags)"
      :suggestions="suggestions.tags.value"
      multiple
      :max="TAG_FILTER_MAX"
      @toggle="toggleTag"
    />
    <FilterEntityMenu
      family="company"
      icon="lucide:building-2"
      label="会社"
      placeholder="搜索会社名, 例如 Key"
      :selected-ids="state.company ? [state.company] : []"
      :selected-items="entityNames.itemsOf('company', [state.company])"
      :suggestions="suggestions.companies.value"
      @toggle="toggleCompany"
    />

    <span class="bg-default-200 h-6 w-px" aria-hidden="true" />

    <FilterYears
      :from="state.releasedFrom"
      :to="state.releasedTo"
      @update="setYears"
    />

    <template #end>
      <FilterSearch
        :query="state.q"
        placeholder="搜索 Galgame 名称"
        history-key="gallib"
        @search="set({ q: $event })"
      />
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
