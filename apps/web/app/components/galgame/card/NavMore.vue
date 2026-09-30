<script setup lang="ts">
import {
  KUN_GALGAME_PROVIDER_LABEL_MAP,
  PROVIDER_KEY_OPTIONS
} from '~/constants/galgameResource'
import { settle } from '#shared/utils/api/problem'

type Filters = QueryValues<typeof GALGAME_FILTER_SCHEMA>

const { state, set } = useGalgameFilters()

const monthOptions = Array.from({ length: 12 }, (_, i) => ({
  value: String(i + 1),
  label: `${i + 1} 月`
}))

const providerOptions = PROVIDER_KEY_OPTIONS.map((key) => ({
  value: key,
  label: KUN_GALGAME_PROVIDER_LABEL_MAP[key]
}))

// The pill's resting text is the selected option's own label, so a bare 不限
// draws two identically-labelled pills side by side.
const minCountOptions = [
  { value: '0', label: '人数不限' },
  { value: '5', label: '≥5 人', hint: '过滤小样本' },
  { value: '10', label: '≥10 人' },
  { value: '20', label: '≥20 人' },
  { value: '50', label: '≥50 人' }
]

const minRatingOptions = [
  { value: '0', label: '评分不限' },
  { value: '7', label: '7 分+', hint: '贝叶斯平滑后' },
  { value: '8', label: '8 分+' },
  { value: '9', label: '9 分+' }
]

const months = computed(() => state.value.releasedMonths.map(String))

const setYears = (range: { from: string; to: string }) => {
  set({ releasedFrom: range.from, releasedTo: range.to })
}

const collectedCalendar = ref<{ year: number; month: number }[]>([])
const api = useApiClient()
const { allowsNsfw } = useContentStance()
const loadCollectedCalendar = async () => {
  const res = await settle(
    api.GET('/works/collected-months', {
      params: { query: { include_nsfw: allowsNsfw.value } }
    })
  )
  if (!res.ok) {
    reportProblem(res.problem)
    return
  }
  collectedCalendar.value = res.data.items
}
onMounted(loadCollectedCalendar)
watch(allowsNsfw, loadCollectedCalendar)

const selectedCollectedYear = computed(
  () => state.value.collectedFrom || state.value.collectedTo
)
const collectedYearOptions = computed(() => {
  const years = new Set<number>()
  for (const c of collectedCalendar.value) years.add(c.year)
  return [...years]
    .sort((a, b) => b - a)
    .map((year) => ({ value: String(year), label: String(year) }))
})
const collectedMonthOptions = computed(() => {
  const year = Number(selectedCollectedYear.value)
  const months = new Set<number>()
  if (year) {
    for (const c of collectedCalendar.value) {
      if (c.year === year) months.add(c.month)
    }
  }
  return [...months]
    .sort((a, b) => a - b)
    .map((month) => ({ value: String(month), label: `${month} 月` }))
})
// Multi-select under the hood (native click-to-toggle like 发售月份), but we
// keep at most one item so it behaves as a single choice with deselect.
const collectedYearsModel = computed(() =>
  state.value.collectedFrom ? [state.value.collectedFrom] : []
)
const collectedMonthsModel = computed(() =>
  state.value.collectedMonths.map(String)
)

const setCollectedYear = (value: string) => {
  // Browsing by collection time wants collection order, but the server used to
  // force SortField = created whenever this filter was set: the sort pill went
  // on reading 评分 while the list came back in another order entirely. Move the
  // pill instead, and only off the page default, so a sort the reader actually
  // picked survives the filter.
  set({
    collectedFrom: value,
    collectedTo: value,
    collectedMonths: [],
    ...(value && state.value.sortField === 'time'
      ? { sortField: 'created' }
      : {})
  })
}

const lastOf = (values: string[]) => values[values.length - 1] ?? ''

const onPickCollectedYear = (values: string[]) => {
  setCollectedYear(lastOf(values))
}
const onPickCollectedMonth = (values: string[]) => {
  if (!selectedCollectedYear.value) {
    return
  }
  const month = Number(lastOf(values))
  set({ collectedMonths: month ? [month] : [] })
}
</script>

<template>
  <KunTooltip text="只保留至少有一个所选网盘的作品" position="bottom">
    <FilterMenu
      icon="lucide:hard-drive-download"
      label="含网盘"
      multiple
      :options="providerOptions"
      :model-value="state.includeProviders"
      @update:model-value="
        set({ includeProviders: $event as Filters['includeProviders'] })
      "
    />
  </KunTooltip>
  <KunTooltip text="丢掉只有这些网盘可选的作品" position="bottom">
    <FilterMenu
      icon="lucide:hard-drive-upload"
      label="排除仅含"
      multiple
      :options="providerOptions"
      :model-value="state.excludeOnlyProviders"
      @update:model-value="
        set({ excludeOnlyProviders: $event as Filters['excludeOnlyProviders'] })
      "
    />
  </KunTooltip>

  <FilterMenu
    icon="lucide:star"
    label="最低评分"
    :options="minRatingOptions"
    :model-value="String(state.minRating)"
    empty-value="0"
    @update:model-value="set({ minRating: Number($event) })"
  />
  <FilterMenu
    icon="lucide:users"
    label="评分人数"
    :options="minCountOptions"
    :model-value="String(state.minRatingCount)"
    empty-value="0"
    @update:model-value="set({ minRatingCount: Number($event) })"
  />

  <FilterYears
    :from="state.releasedFrom"
    :to="state.releasedTo"
    @update="setYears"
  />
  <FilterMenu
    icon="lucide:calendar-days"
    label="发售月份"
    multiple
    :columns="3"
    :options="monthOptions"
    :model-value="months"
    @update:model-value="
      set({ releasedMonths: ($event as string[]).map(Number) })
    "
  />

  <FilterMenu
    icon="lucide:archive"
    label="收录年份"
    multiple
    :columns="3"
    :options="collectedYearOptions"
    :model-value="collectedYearsModel"
    @update:model-value="onPickCollectedYear($event as string[])"
  />
  <FilterMenu
    icon="lucide:calendar-check-2"
    label="收录月份"
    multiple
    :columns="3"
    :options="collectedMonthOptions"
    :model-value="collectedMonthsModel"
    @update:model-value="onPickCollectedMonth($event as string[])"
  />
</template>
