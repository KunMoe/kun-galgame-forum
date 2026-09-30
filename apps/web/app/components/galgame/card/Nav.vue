<script setup lang="ts">
import { KUN_GALGAME_RESOURCE_SORT_FIELD_MAP } from '~/constants/galgame'
import {
  KUN_GALGAME_PROVIDER_LABEL_MAP,
  type ProviderKey
} from '~/constants/galgameResource'
import { KUN_GALGAME_RATING_GAME_TYPE_MAP } from '~/constants/galgame-rating'
import {
  EMULATOR_RUNTIME_OPTIONS,
  LANGUAGE_OPTIONS,
  PLATFORM_FILTER_OPTIONS,
  RESOURCE_TYPE_OPTIONS
} from '#shared/utils/galgameResourceVocab'

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
  languages,
  platforms,
  runtimes,
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

const { open: openSettingPanel } = useSettingPanel()

watch(
  () => [
    type.value,
    languages.value,
    platforms.value,
    runtimes.value,
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

// The entity pages list through an API that takes one value per resource
// axis, so only /galgame gets the multi-select.
const toValues = (value: string | string[]) =>
  Array.isArray(value) ? value : value ? [value] : []
const axisModel = (values: string[]) =>
  props.isShowAdvanced ? values : (values[0] ?? '')
const axisOptions = (allLabel: string, options: FilterOption[]) =>
  computed(() =>
    props.isShowAdvanced
      ? options
      : [{ value: '', label: allLabel }, ...options]
  )

const typeOptions = [{ value: '', label: '全部类型' }, ...RESOURCE_TYPE_OPTIONS]
const langOptions = axisOptions('全部语言', LANGUAGE_OPTIONS)
const platformOptions = axisOptions('全部平台', PLATFORM_FILTER_OPTIONS)
const emulatorOptions = axisOptions('全部模拟器', EMULATOR_RUNTIME_OPTIONS)

const isEmulator = computed(() => platforms.value.includes('emulator'))

const setPlatforms = (values: string[]) => {
  platforms.value = values
  if (!values.includes('emulator')) {
    runtimes.value = []
  }
}

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

const months = computed(() => csvToArray(releasedMonths.value))
const includes = computed(() => csvToArray(includeProviders.value))
const excludes = computed(() => csvToArray(excludeOnlyProviders.value))

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

const selectedCollectedYear = computed(
  () => collectedFrom.value || collectedTo.value || ''
)
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

const moreCount = computed(
  () =>
    [
      includes.value.length,
      excludes.value.length,
      minRating.value > 0,
      minRatingCount.value > 0,
      releasedFrom.value || releasedTo.value,
      months.value.length,
      hasCollectedFilter.value
    ].filter(Boolean).length
)
const isMoreOpen = ref(moreCount.value > 0)

const labelOf = (options: FilterOption[], value: string) =>
  options.find((option) => option.value === value)?.label ?? value

const chips = computed<FilterChip[]>(() => {
  const list: FilterChip[] = []
  for (const value of platforms.value) {
    list.push({
      key: `platform:${value}`,
      prefix: '平台',
      label: labelOf(PLATFORM_FILTER_OPTIONS, value)
    })
  }
  if (isEmulator.value) {
    for (const value of runtimes.value) {
      list.push({
        key: `runtime:${value}`,
        prefix: '模拟器',
        label: labelOf(EMULATOR_RUNTIME_OPTIONS, value)
      })
    }
  }
  if (gameType.value) {
    list.push({
      key: 'gameType',
      label: labelOf(gameTypeOptions, gameType.value)
    })
  }
  if (type.value) {
    list.push({
      key: 'type',
      prefix: '类型',
      label: labelOf(typeOptions, type.value)
    })
  }
  for (const value of languages.value) {
    list.push({
      key: `language:${value}`,
      prefix: '语言',
      label: labelOf(LANGUAGE_OPTIONS, value)
    })
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
  if (minRating.value > 0) {
    list.push({ key: 'minRating', label: `${minRating.value} 分+` })
  }
  if (minRatingCount.value > 0) {
    list.push({
      key: 'minRatingCount',
      label: `≥${minRatingCount.value} 人评分`
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
  return list
})

const removeChip = (key: string) => {
  const [dimension, value] = key.split(':')
  const without = (values: string[]) => values.filter((item) => item !== value)
  if (dimension === 'platform') {
    setPlatforms(without(platforms.value))
  } else if (dimension === 'runtime') {
    runtimes.value = without(runtimes.value)
  } else if (dimension === 'gameType') {
    gameType.value = ''
  } else if (dimension === 'type') {
    type.value = ''
  } else if (dimension === 'language') {
    languages.value = without(languages.value)
  } else if (dimension === 'include') {
    includeProviders.value = without(includes.value).join(',')
  } else if (dimension === 'exclude') {
    excludeOnlyProviders.value = without(excludes.value).join(',')
  } else if (dimension === 'minRating') {
    minRating.value = 0
  } else if (dimension === 'minRatingCount') {
    minRatingCount.value = 0
  } else if (dimension === 'years') {
    releasedFrom.value = ''
    releasedTo.value = ''
  } else if (dimension === 'month') {
    releasedMonths.value = without(months.value).join(',')
  } else if (dimension === 'collected') {
    collectedFrom.value = ''
    collectedTo.value = ''
    collectedMonths.value = ''
  }
}

const clearFilters = () => {
  setPlatforms([])
  gameType.value = ''
  type.value = ''
  languages.value = []
  includeProviders.value = ''
  excludeOnlyProviders.value = ''
  minRating.value = 0
  minRatingCount.value = 0
  releasedFrom.value = ''
  releasedTo.value = ''
  releasedMonths.value = ''
  collectedFrom.value = ''
  collectedTo.value = ''
  collectedMonths.value = ''
}
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
          :name="sortOrder === 'desc' ? 'lucide:arrow-down' : 'lucide:arrow-up'"
          class="size-4 text-inherit"
        />
      </button>
    </KunTooltip>

    <span class="bg-default-200 h-6 w-px" aria-hidden="true" />

    <FilterMenu
      icon="lucide:monitor-smartphone"
      label="平台"
      :multiple="isShowAdvanced"
      :options="platformOptions"
      :model-value="axisModel(platforms)"
      empty-value=""
      @update:model-value="setPlatforms(toValues($event))"
    />
    <FilterMenu
      v-if="isEmulator"
      icon="lucide:joystick"
      label="模拟器"
      :multiple="isShowAdvanced"
      :options="emulatorOptions"
      :model-value="axisModel(runtimes)"
      empty-value=""
      @update:model-value="runtimes = toValues($event)"
    />
    <FilterMenu
      icon="lucide:gamepad-2"
      label="游戏类型"
      :options="gameTypeOptions"
      :model-value="gameType"
      empty-value=""
      @update:model-value="gameType = firstOf($event)"
    />
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
      :multiple="isShowAdvanced"
      :options="langOptions"
      :model-value="axisModel(languages)"
      empty-value=""
      @update:model-value="languages = toValues($event)"
    />

    <FilterTrigger
      v-if="isShowAdvanced"
      icon="lucide:sliders-horizontal"
      label="更多筛选"
      :value="moreCount ? `更多筛选 · ${moreCount}` : ''"
      :active="moreCount > 0"
      :open="isMoreOpen"
      :aria-expanded="isMoreOpen"
      @click="isMoreOpen = !isMoreOpen"
    />

    <template v-if="isShowAdvanced && isMoreOpen" #more>
      <GalgameCardNavMore />
    </template>

    <template v-if="isShowAdvanced" #end>
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
