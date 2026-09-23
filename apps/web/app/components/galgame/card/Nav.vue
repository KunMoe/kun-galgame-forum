<script setup lang="ts">
import { KUN_GALGAME_RESOURCE_SORT_FIELD_MAP } from '~/constants/galgame'
import {
  KUN_GALGAME_PROVIDER_LABEL_MAP,
  PROVIDER_KEY_OPTIONS,
  type ProviderKey
} from '~/constants/galgameResource'
import { KUN_GALGAME_RATING_GAME_TYPE_MAP } from '~/constants/galgame-rating'
import {
  LANGUAGE_OPTIONS,
  PLATFORM_OPTIONS,
  RESOURCE_TYPE_OPTIONS
} from '#shared/utils/galgameResourceVocab'
import { settle } from '#shared/utils/api/problem'

const props = withDefaults(
  defineProps<{
    isShowAdvanced?: boolean
    total?: number | null
    pending?: boolean
  }>(),
  { isShowAdvanced: false, total: null, pending: false }
)

const {
  page,
  type,
  language,
  platform,
  gameType,
  sortField,
  sortOrder,
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

const showDisplay = ref(false)

watch(
  () => [
    type.value,
    language.value,
    platform.value,
    gameType.value,
    sortField.value,
    sortOrder.value,
    releasedFrom.value,
    releasedTo.value,
    releasedMonths.value,
    collectedFrom.value,
    collectedTo.value,
    collectedMonths.value,
    includeProviders.value,
    excludeOnlyProviders.value,
    minRatingCount.value,
    minRating.value
  ],
  () => {
    page.value = 1
  }
)

const csvToArray = (csv: string) => csv.split(',').filter(Boolean)

const firstOf = (value: string | string[]) =>
  Array.isArray(value) ? (value[0] ?? '') : value

const typeOptions = [
  { value: '', label: '全部类型' },
  ...RESOURCE_TYPE_OPTIONS
]
const langOptions = [
  { value: '', label: '全部语言' },
  ...LANGUAGE_OPTIONS
]
const platformOptions = [
  { value: '', label: '全部平台' },
  ...PLATFORM_OPTIONS
]

const gameTypeOptions = [
  { value: '', label: '全部作品' },
  ...Object.entries(KUN_GALGAME_RATING_GAME_TYPE_MAP).map(([value, label]) => ({
    value,
    label
  })),
  { value: 'uncategorized', label: '未分类' }
]

const sortOptions = Object.entries(KUN_GALGAME_RESOURCE_SORT_FIELD_MAP).map(
  ([value, label]) => ({
    value: value === 'views' ? 'view' : value,
    label
  })
)

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
const setCsv = (values: string[]) => [...values].sort().join(',')

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
// Only /galgame renders the advanced pills; the four entity pages mount this
// same Nav with isShowAdvanced false, and an unconditional load spent a
// two-sequential-scan aggregate (21ms over 8.5k published rows on prod) per
// visit to fill a dropdown none of them draw.
onMounted(() => {
  if (props.isShowAdvanced) loadCollectedCalendar()
})
watch(allowsNsfw, () => {
  if (props.isShowAdvanced) loadCollectedCalendar()
})

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

const collectedLabel = computed(() => {
  const parts: string[] = []
  if (selectedCollectedYear.value) {
    parts.push(`收录年份 ${selectedCollectedYear.value}`)
  }
  if (collectedMonths.value) {
    parts.push(`收录月份 ${Number(collectedMonths.value)}`)
  }
  return parts.join(' / ')
})
const hasCollectedFilter = computed(
  () => !!selectedCollectedYear.value || !!collectedMonths.value
)

const labelOf = (options: FilterOption[], value: string) =>
  options.find((option) => option.value === value)?.label ?? value

const chips = computed<FilterChip[]>(() => {
  const list: FilterChip[] = []
  if (type.value) {
    list.push({
      key: 'type',
      prefix: '类型',
      label: labelOf(typeOptions, type.value)
    })
  }
  if (language.value) {
    list.push({
      key: 'language',
      prefix: '语言',
      label: labelOf(langOptions, language.value)
    })
  }
  if (platform.value) {
    list.push({
      key: 'platform',
      prefix: '平台',
      label: labelOf(platformOptions, platform.value)
    })
  }
  if (gameType.value) {
    list.push({
      key: 'gameType',
      label: labelOf(gameTypeOptions, gameType.value)
    })
  }
  if (releasedFrom.value || releasedTo.value) {
    list.push({ key: 'years', label: yearRangeLabel.value })
  }
  for (const month of months.value) {
    list.push({ key: `month:${month}`, label: `${month} 月` })
  }
  if (hasCollectedFilter.value) {
    list.push({ key: 'collected', label: collectedLabel.value })
  }
  for (const key of includes.value) {
    list.push({
      key: `include:${key}`,
      prefix: '含',
      label: KUN_GALGAME_PROVIDER_LABEL_MAP[key as ProviderKey] ?? key
    })
  }
  for (const key of excludes.value) {
    list.push({
      key: `exclude:${key}`,
      prefix: '排除仅',
      label: KUN_GALGAME_PROVIDER_LABEL_MAP[key as ProviderKey] ?? key
    })
  }
  if (minRatingCount.value > 0) {
    list.push({
      key: 'minRatingCount',
      label: `≥${minRatingCount.value} 人评分`
    })
  }
  if (minRating.value > 0) {
    list.push({ key: 'minRating', label: `${minRating.value} 分+` })
  }
  return list
})

const removeChip = (key: string) => {
  const [dimension, value] = key.split(':')
  if (dimension === 'type') {
    type.value = ''
  } else if (dimension === 'language') {
    language.value = ''
  } else if (dimension === 'platform') {
    platform.value = ''
  } else if (dimension === 'gameType') {
    gameType.value = ''
  } else if (dimension === 'years') {
    setYears({ from: '', to: '' })
  } else if (dimension === 'month') {
    setMonths(months.value.filter((month) => month !== value))
  } else if (dimension === 'collected') {
    collectedFrom.value = ''
    collectedTo.value = ''
    collectedMonths.value = ''
  } else if (dimension === 'include') {
    includeProviders.value = setCsv(
      includes.value.filter((item) => item !== value)
    )
  } else if (dimension === 'exclude') {
    excludeOnlyProviders.value = setCsv(
      excludes.value.filter((item) => item !== value)
    )
  } else if (dimension === 'minRatingCount') {
    minRatingCount.value = 0
  } else if (dimension === 'minRating') {
    minRating.value = 0
  }
}

const clearFilters = () => {
  type.value = ''
  language.value = ''
  platform.value = ''
  gameType.value = ''
  releasedFrom.value = ''
  releasedTo.value = ''
  releasedMonths.value = ''
  collectedFrom.value = ''
  collectedTo.value = ''
  collectedMonths.value = ''
  includeProviders.value = ''
  excludeOnlyProviders.value = ''
  minRatingCount.value = 0
  minRating.value = 0
}
</script>

<template>
  <div class="space-y-2">
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
        :model-value="sortField"
        empty-value="time"
        @update:model-value="sortField = $event as typeof sortField"
      />

      <KunTooltip
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

      <span class="bg-default-200 h-6 w-px" aria-hidden="true" />

      <FilterMenu
        icon="lucide:package"
        label="资源类型"
        :options="typeOptions"
        :model-value="type"
        empty-value=""
        @update:model-value="type = firstOf($event)"
      />
      <FilterMenu
        icon="lucide:languages"
        label="语言"
        :options="langOptions"
        :model-value="language"
        empty-value=""
        @update:model-value="language = firstOf($event)"
      />
      <FilterMenu
        icon="lucide:monitor-smartphone"
        label="平台"
        :options="platformOptions"
        :model-value="platform"
        empty-value=""
        @update:model-value="platform = firstOf($event)"
      />
      <FilterMenu
        icon="lucide:gamepad-2"
        label="游戏类型"
        :options="gameTypeOptions"
        :model-value="gameType"
        empty-value=""
        @update:model-value="gameType = firstOf($event)"
      />

      <template v-if="isShowAdvanced">
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
            @update:model-value="
              excludeOnlyProviders = setCsv($event as string[])
            "
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
      </template>

      <template v-if="isShowAdvanced" #end>
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
