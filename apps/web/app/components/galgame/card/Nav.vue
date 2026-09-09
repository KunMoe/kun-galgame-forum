<script setup lang="ts">
import {
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_GALGAME_RESOURCE_SORT_FIELD_MAP
} from '~/constants/galgame'
import {
  KUN_GALGAME_PROVIDER_LABEL_MAP,
  PROVIDER_KEY_OPTIONS,
  type ProviderKey
} from '~/constants/galgameResource'
import { KUN_GALGAME_RATING_GAME_TYPE_MAP } from '~/constants/galgame-rating'
import type {
  KunGalgameResourceTypeOptions,
  KunGalgameResourceLanguageOptions,
  KunGalgameResourcePlatformOptions
} from '~/constants/galgame'

withDefaults(
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

const typeOptions = Object.entries(KUN_GALGAME_RESOURCE_TYPE_MAP)
  .filter(([k]) => k !== 'name')
  .map(([value, label]) => ({ value, label }))

const langOptions = Object.entries(KUN_GALGAME_RESOURCE_LANGUAGE_MAP).map(
  ([value, label]) => ({ value, label })
)

const platformOptions = Object.entries(KUN_GALGAME_RESOURCE_PLATFORM_MAP)
  .filter(([k]) => k !== 'name')
  .map(([value, label]) => ({ value, label }))

const gameTypeOptions = [
  { value: 'all', label: '全部作品' },
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
const loadCollectedCalendar = async () => {
  const res = await kunFetch<{ year: number; month: number }[]>(
    '/galgame/collected-calendar'
  )
  if (res) collectedCalendar.value = res
}
onMounted(loadCollectedCalendar)

const selectedCollectedYear = computed(() => collectedFrom.value || collectedTo.value || '')
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
}
const setCollectedMonth = (value: string) => {
  if (value === '' || selectedCollectedYear.value) {
    collectedMonths.value = value
  }
}

const onPickCollectedYear = (values: string[]) => {
  setCollectedYear(values.length ? values[values.length - 1] : '')
}
const onPickCollectedMonth = (values: string[]) => {
  if (!selectedCollectedYear.value) {
    return
  }
  setCollectedMonth(values.length ? values[values.length - 1] : '')
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
  if (type.value !== 'all') {
    list.push({
      key: 'type',
      prefix: '类型',
      label: labelOf(typeOptions, type.value)
    })
  }
  if (language.value !== 'all') {
    list.push({
      key: 'language',
      prefix: '语言',
      label: labelOf(langOptions, language.value)
    })
  }
  if (platform.value !== 'all') {
    list.push({
      key: 'platform',
      prefix: '平台',
      label: labelOf(platformOptions, platform.value)
    })
  }
  if (gameType.value !== 'all') {
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
    type.value = 'all'
  } else if (dimension === 'language') {
    language.value = 'all'
  } else if (dimension === 'platform') {
    platform.value = 'all'
  } else if (dimension === 'gameType') {
    gameType.value = 'all'
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
  type.value = 'all'
  language.value = 'all'
  platform.value = 'all'
  gameType.value = 'all'
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
        empty-value="all"
        @update:model-value="type = $event as KunGalgameResourceTypeOptions"
      />
      <FilterMenu
        icon="lucide:languages"
        label="语言"
        :options="langOptions"
        :model-value="language"
        empty-value="all"
        @update:model-value="
          language = $event as KunGalgameResourceLanguageOptions
        "
      />
      <FilterMenu
        icon="lucide:monitor-smartphone"
        label="平台"
        :options="platformOptions"
        :model-value="platform"
        empty-value="all"
        @update:model-value="
          platform = $event as KunGalgameResourcePlatformOptions
        "
      />
      <FilterMenu
        icon="lucide:gamepad-2"
        label="游戏类型"
        :options="gameTypeOptions"
        :model-value="gameType"
        empty-value="all"
        @update:model-value="gameType = $event as string"
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
