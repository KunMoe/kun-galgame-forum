<script setup lang="ts">
import { SEARCH_GALGAME_SORTS, TAG_FILTER_MAX } from './items'

withDefaults(defineProps<{ total?: number; pending?: boolean }>(), {
  total: 0,
  pending: false
})

const { state, set, clear: clearFilters } = useSearchGalgameFilters()

const entityNames = useEntityNames()

const yearRangeLabel = computed(() => {
  const { released_from: from, released_to: to } = state.value
  if (from && to) {
    return from === to ? `${from} 年` : `${from} - ${to}`
  }
  return from ? `${from} 年至今` : `${to} 年以前`
})

const setYears = (range: { from: string; to: string }) => {
  set({ released_from: range.from, released_to: range.to })
}

const toggleCompany = (item: SearchEntityItem) => {
  entityNames.remember(item)
  set({ company_id: state.value.company_id === item.id ? 0 : item.id })
}

const toggleTag = (item: SearchEntityItem) => {
  entityNames.remember(item)
  const ids = state.value.tag_ids
  set({
    tag_ids: ids.includes(item.id)
      ? ids.filter((id) => id !== item.id)
      : [...ids, item.id].slice(0, TAG_FILTER_MAX)
  })
}

const chips = computed<FilterChip[]>(() => {
  const f = state.value
  const list: FilterChip[] = []
  if (f.company_id) {
    list.push({
      key: 'company',
      prefix: '会社',
      label: entityNames.labelOf('company', f.company_id)
    })
  }
  for (const id of f.tag_ids) {
    list.push({
      key: `tag:${id}`,
      prefix: '标签',
      label: entityNames.labelOf('tag', id)
    })
  }
  if (f.released_from || f.released_to) {
    list.push({ key: 'years', label: yearRangeLabel.value })
  }
  return list
})

const removeChip = (key: string) => {
  const [dimension, value] = key.split(':')
  if (dimension === 'company') {
    set({ company_id: 0 })
  } else if (dimension === 'tag') {
    set({ tag_ids: state.value.tag_ids.filter((id) => id !== Number(value)) })
  } else if (dimension === 'years') {
    setYears({ from: '', to: '' })
  }
}

watch(
  [() => state.value.company_id, () => state.value.tag_ids],
  ([company, tags]) => entityNames.resolve({ company: [company], tag: tags }),
  { immediate: true }
)
</script>

<template>
  <FilterBar
    :chips="chips"
    :total="total"
    :pending="pending"
    unit="个 Galgame"
    @remove="removeChip"
    @clear="clearFilters"
  >
    <FilterMenu
      icon="lucide:arrow-down-up"
      label="排序"
      :options="SEARCH_GALGAME_SORTS"
      :model-value="state.sort"
      empty-value="relevance"
      @update:model-value="set({ sort: $event as string })"
    />

    <span class="bg-default-200 h-6 w-px" aria-hidden="true" />

    <FilterEntityMenu
      family="company"
      icon="lucide:building-2"
      label="会社"
      placeholder="搜索会社名, 例如 Key"
      :selected-ids="state.company_id ? [state.company_id] : []"
      :selected-items="entityNames.itemsOf('company', [state.company_id])"
      @toggle="toggleCompany"
    />
    <FilterEntityMenu
      family="tag"
      icon="lucide:tag"
      label="标签"
      placeholder="搜索标签, 例如 校园"
      :selected-ids="state.tag_ids"
      :selected-items="entityNames.itemsOf('tag', state.tag_ids)"
      multiple
      :max="TAG_FILTER_MAX"
      @toggle="toggleTag"
    />

    <FilterYears
      :from="state.released_from"
      :to="state.released_to"
      @update="setYears"
    />

    <template #end>
      <KunTooltip
        text="资料库的搜索索引里没有评分这个属性, 也没有按评分排序。想按评分筛选请前往 Galgame 资源资料库, 那里的评分是本站自己的。"
        position="bottom"
      >
        <KunIcon
          name="lucide:circle-help"
          class="text-default-400 hover:text-default-600 size-4 cursor-help"
        />
      </KunTooltip>
    </template>
  </FilterBar>
</template>
