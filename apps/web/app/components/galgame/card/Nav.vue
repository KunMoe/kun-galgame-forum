<script setup lang="ts">
import { KUN_GALGAME_RESOURCE_SORT_FIELD_MAP } from '~/constants/galgame'
import { KUN_GALGAME_PROVIDER_LABEL_MAP } from '~/constants/galgameResource'
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

type Filters = QueryValues<typeof GALGAME_FILTER_SCHEMA>

const { state, set } = useGalgameFilters()

const { open: openSettingPanel } = useSettingPanel()

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

const isEmulator = computed(() => state.value.platform.includes('emulator'))

const setPlatforms = (platform: string[]) => {
  set(platform.includes('emulator') ? { platform } : { platform, runtime: [] })
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

const yearRangeLabel = computed(() => {
  const { releasedFrom: from, releasedTo: to } = state.value
  if (from && to) {
    return from === to ? `${from} 年` : `${from} - ${to}`
  }
  return from ? `${from} 年至今` : `${to} 年以前`
})

const collectedYear = computed(
  () => state.value.collectedFrom || state.value.collectedTo
)
const collectedMonth = computed(() => state.value.collectedMonths[0] ?? 0)
const hasCollectedFilter = computed(
  () => !!collectedYear.value || !!collectedMonth.value
)
const collectedLabel = computed(() =>
  [
    collectedYear.value && `收录年份 ${collectedYear.value}`,
    collectedMonth.value && `收录月份 ${collectedMonth.value}`
  ]
    .filter(Boolean)
    .join(' / ')
)

const moreCount = computed(() => {
  const f = state.value
  return [
    f.includeProviders.length,
    f.excludeOnlyProviders.length,
    f.minRating > 0,
    f.minRatingCount > 0,
    f.releasedFrom || f.releasedTo,
    f.releasedMonths.length,
    hasCollectedFilter.value
  ].filter(Boolean).length
})
const isMoreOpen = ref(moreCount.value > 0)

const labelOf = (options: FilterOption[], value: string) =>
  options.find((option) => option.value === value)?.label ?? value

const chips = computed<FilterChip[]>(() => {
  const f = state.value
  const list: FilterChip[] = []
  for (const value of f.platform) {
    list.push({
      key: `platform:${value}`,
      prefix: '平台',
      label: labelOf(PLATFORM_FILTER_OPTIONS, value)
    })
  }
  if (isEmulator.value) {
    for (const value of f.runtime) {
      list.push({
        key: `runtime:${value}`,
        prefix: '模拟器',
        label: labelOf(EMULATOR_RUNTIME_OPTIONS, value)
      })
    }
  }
  if (f.gameType) {
    list.push({
      key: 'gameType',
      label: labelOf(gameTypeOptions, f.gameType)
    })
  }
  if (f.type) {
    list.push({
      key: 'type',
      prefix: '类型',
      label: labelOf(typeOptions, f.type)
    })
  }
  for (const value of f.language) {
    list.push({
      key: `language:${value}`,
      prefix: '语言',
      label: labelOf(LANGUAGE_OPTIONS, value)
    })
  }
  for (const key of f.includeProviders) {
    list.push({
      key: `include:${key}`,
      prefix: '含',
      label: KUN_GALGAME_PROVIDER_LABEL_MAP[key]
    })
  }
  for (const key of f.excludeOnlyProviders) {
    list.push({
      key: `exclude:${key}`,
      prefix: '排除仅',
      label: KUN_GALGAME_PROVIDER_LABEL_MAP[key]
    })
  }
  if (f.minRating > 0) {
    list.push({ key: 'minRating', label: `${f.minRating} 分+` })
  }
  if (f.minRatingCount > 0) {
    list.push({
      key: 'minRatingCount',
      label: `≥${f.minRatingCount} 人评分`
    })
  }
  if (f.releasedFrom || f.releasedTo) {
    list.push({ key: 'years', label: yearRangeLabel.value })
  }
  for (const month of f.releasedMonths) {
    list.push({ key: `month:${month}`, label: `${month} 月` })
  }
  if (hasCollectedFilter.value) {
    list.push({ key: 'collected', label: collectedLabel.value })
  }
  return list
})

const removeChip = (key: string) => {
  const [dimension, value] = key.split(':')
  const f = state.value
  const without = <T extends string | number>(values: T[]) =>
    values.filter((item) => String(item) !== value)
  if (dimension === 'platform') {
    setPlatforms(without(f.platform))
  } else if (dimension === 'runtime') {
    set({ runtime: without(f.runtime) })
  } else if (dimension === 'gameType') {
    set({ gameType: '' })
  } else if (dimension === 'type') {
    set({ type: '' })
  } else if (dimension === 'language') {
    set({ language: without(f.language) })
  } else if (dimension === 'include') {
    set({ includeProviders: without(f.includeProviders) })
  } else if (dimension === 'exclude') {
    set({ excludeOnlyProviders: without(f.excludeOnlyProviders) })
  } else if (dimension === 'minRating') {
    set({ minRating: 0 })
  } else if (dimension === 'minRatingCount') {
    set({ minRatingCount: 0 })
  } else if (dimension === 'years') {
    set({ releasedFrom: '', releasedTo: '' })
  } else if (dimension === 'month') {
    set({ releasedMonths: without(f.releasedMonths) })
  } else if (dimension === 'collected') {
    set({ collectedFrom: '', collectedTo: '', collectedMonths: [] })
  }
}

const clearFilters = () => {
  set({
    platform: [],
    runtime: [],
    gameType: '',
    type: '',
    language: [],
    includeProviders: [],
    excludeOnlyProviders: [],
    minRating: 0,
    minRatingCount: 0,
    releasedFrom: '',
    releasedTo: '',
    releasedMonths: [],
    collectedFrom: '',
    collectedTo: '',
    collectedMonths: []
  })
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
      :model-value="state.sortField"
      empty-value="time"
      @update:model-value="set({ sortField: firstOf($event) })"
    />

    <KunTooltip
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

    <FilterMenu
      icon="lucide:monitor-smartphone"
      label="平台"
      :multiple="isShowAdvanced"
      :options="platformOptions"
      :model-value="axisModel(state.platform)"
      empty-value=""
      @update:model-value="setPlatforms(toValues($event))"
    />
    <FilterMenu
      v-if="isEmulator"
      icon="lucide:joystick"
      label="模拟器"
      :multiple="isShowAdvanced"
      :options="emulatorOptions"
      :model-value="axisModel(state.runtime)"
      empty-value=""
      @update:model-value="set({ runtime: toValues($event) })"
    />
    <FilterMenu
      icon="lucide:gamepad-2"
      label="游戏类型"
      :options="gameTypeOptions"
      :model-value="state.gameType"
      empty-value=""
      @update:model-value="
        set({ gameType: firstOf($event) as Filters['gameType'] })
      "
    />
    <FilterMenu
      icon="lucide:package"
      label="资源类型"
      :options="typeOptions"
      :model-value="state.type"
      empty-value=""
      @update:model-value="set({ type: firstOf($event) })"
    />
    <FilterMenu
      icon="lucide:languages"
      label="语言"
      :multiple="isShowAdvanced"
      :options="langOptions"
      :model-value="axisModel(state.language)"
      empty-value=""
      @update:model-value="set({ language: toValues($event) })"
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
