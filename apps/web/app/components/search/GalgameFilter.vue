<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import { SEARCH_GALGAME_SORTS, TAG_FILTER_MAX } from './items'

withDefaults(defineProps<{ total?: number; pending?: boolean }>(), {
  total: 0,
  pending: false
})

const {
  companyId,
  tagIds,
  releasedFrom,
  releasedTo,
  sort,
  clear: clearFilters
} = useSearchGalgameFilters()

const entityNames = useEntityNames()

// useRouteQuery batches every set made in the same tick into one replace, so
// dropping the page here lands in the same navigation as the filter itself —
// the list sees one change and fires one request.
const page = useRouteQuery('page', 1, { mode: 'replace', transform: Number })

watch([companyId, tagIds, releasedFrom, releasedTo, sort], () => {
  page.value = 1
})

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

const toggleCompany = (item: SearchEntityItem) => {
  entityNames.remember(item)
  companyId.value = companyId.value === item.id ? 0 : item.id
}

const toggleTag = (item: SearchEntityItem) => {
  entityNames.remember(item)
  tagIds.value = tagIds.value.includes(item.id)
    ? tagIds.value.filter((id) => id !== item.id)
    : [...tagIds.value, item.id].slice(0, TAG_FILTER_MAX)
}

const chips = computed<FilterChip[]>(() => {
  const list: FilterChip[] = []
  if (companyId.value) {
    list.push({
      key: 'company',
      prefix: '会社',
      label: entityNames.labelOf('company', companyId.value)
    })
  }
  for (const id of tagIds.value) {
    list.push({
      key: `tag:${id}`,
      prefix: '标签',
      label: entityNames.labelOf('tag', id)
    })
  }
  if (releasedFrom.value || releasedTo.value) {
    list.push({ key: 'years', label: yearRangeLabel.value })
  }
  return list
})

const removeChip = (key: string) => {
  const [dimension, value] = key.split(':')
  if (dimension === 'company') {
    companyId.value = 0
  } else if (dimension === 'tag') {
    tagIds.value = tagIds.value.filter((id) => id !== Number(value))
  } else if (dimension === 'years') {
    setYears({ from: '', to: '' })
  }
}

watch(
  [companyId, tagIds],
  () => entityNames.resolve({ company: [companyId.value], tag: tagIds.value }),
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
      :model-value="sort"
      empty-value="relevance"
      @update:model-value="sort = $event as string"
    />

    <span class="bg-default-200 h-6 w-px" aria-hidden="true" />

    <FilterEntityMenu
      family="company"
      icon="lucide:building-2"
      label="会社"
      placeholder="搜索会社名, 例如 Key"
      :selected-ids="companyId ? [companyId] : []"
      :selected-items="entityNames.itemsOf('company', [companyId])"
      @toggle="toggleCompany"
    />
    <FilterEntityMenu
      family="tag"
      icon="lucide:tag"
      label="标签"
      placeholder="搜索标签, 例如 校园"
      :selected-ids="tagIds"
      :selected-items="entityNames.itemsOf('tag', tagIds)"
      multiple
      :max="TAG_FILTER_MAX"
      @toggle="toggleTag"
    />

    <FilterYears :from="releasedFrom" :to="releasedTo" @update="setYears" />

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
