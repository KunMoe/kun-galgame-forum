<script setup lang="ts">
import {
  KUN_GALGAME_PROVIDER_LABEL_MAP,
  PROVIDER_KEY_OPTIONS,
  type ProviderKey
} from '~/constants/galgameResource'
import { settle } from '#shared/utils/api/problem'

const {
  sortField,
  releasedFrom,
  releasedTo,
  releasedMonths,
  collectedFrom,
  collectedTo,
  collectedMonths,
  includeProviders,
  excludeOnlyProviders,
  minRatingCount,
  minRating
} = useGalgameFilters()

const csvToArray = (csv: string) => csv.split(',').filter(Boolean)
const setCsv = (values: string[]) => [...values].sort().join(',')

const monthOptions = Array.from({ length: 12 }, (_, i) => ({
  value: String(i + 1),
  label: `${i + 1} 月`
}))

const providerOptions = PROVIDER_KEY_OPTIONS.map((key) => ({
  value: key,
  label: KUN_GALGAME_PROVIDER_LABEL_MAP[key as ProviderKey]
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

const months = computed(() => csvToArray(releasedMonths.value))
const setMonths = (values: string[]) => {
  releasedMonths.value = values
    .map(Number)
    .sort((a, b) => a - b)
    .join(',')
}

const includes = computed(() => csvToArray(includeProviders.value))
const excludes = computed(() => csvToArray(excludeOnlyProviders.value))

const setYears = (range: { from: string; to: string }) => {
  releasedFrom.value = range.from
  releasedTo.value = range.to
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
  () => collectedFrom.value || collectedTo.value || ''
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
  collectedFrom.value ? [collectedFrom.value] : []
)
const collectedMonthsModel = computed(() =>
  collectedMonths.value ? [collectedMonths.value] : []
)

const setCollectedYear = (value: string) => {
  collectedFrom.value = value
  collectedTo.value = value
  collectedMonths.value = ''
  // Browsing by collection time wants collection order, but the server used to
  // force SortField = created whenever this filter was set: the sort pill went
  // on reading 评分 while the list came back in another order entirely. Move the
  // pill instead, and only off the page default, so a sort the reader actually
  // picked survives the filter.
  if (value && sortField.value === 'time') {
    sortField.value = 'created'
  }
}
const setCollectedMonth = (value: string) => {
  if (value === '' || selectedCollectedYear.value) {
    collectedMonths.value = value
  }
}

const lastOf = (values: string[]) => values[values.length - 1] ?? ''

const onPickCollectedYear = (values: string[]) => {
  setCollectedYear(lastOf(values))
}
const onPickCollectedMonth = (values: string[]) => {
  if (!selectedCollectedYear.value) {
    return
  }
  setCollectedMonth(lastOf(values))
}
</script>

<template>
  <KunTooltip text="只保留至少有一个所选网盘的作品" position="bottom">
    <FilterMenu
      icon="lucide:hard-drive-download"
      label="含网盘"
      multiple
      :options="providerOptions"
      :model-value="includes"
      @update:model-value="includeProviders = setCsv($event as string[])"
    />
  </KunTooltip>
  <KunTooltip text="丢掉只有这些网盘可选的作品" position="bottom">
    <FilterMenu
      icon="lucide:hard-drive-upload"
      label="排除仅含"
      multiple
      :options="providerOptions"
      :model-value="excludes"
      @update:model-value="excludeOnlyProviders = setCsv($event as string[])"
    />
  </KunTooltip>

  <FilterMenu
    icon="lucide:star"
    label="最低评分"
    :options="minRatingOptions"
    :model-value="String(minRating)"
    empty-value="0"
    @update:model-value="minRating = Number($event)"
  />
  <FilterMenu
    icon="lucide:users"
    label="评分人数"
    :options="minCountOptions"
    :model-value="String(minRatingCount)"
    empty-value="0"
    @update:model-value="minRatingCount = Number($event)"
  />

  <FilterYears :from="releasedFrom" :to="releasedTo" @update="setYears" />
  <FilterMenu
    icon="lucide:calendar-days"
    label="发售月份"
    multiple
    :columns="3"
    :options="monthOptions"
    :model-value="months"
    @update:model-value="setMonths($event as string[])"
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
