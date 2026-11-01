<script setup lang="ts">
import type {
  RatedWorkRankingSort,
  RatingSite,
  WorkFacets
} from '#shared/utils/api/schemas'
import { PLATFORM_OPTIONS } from '#shared/utils/galgameResourceVocab'
import { TAG_FILTER_MAX } from '~/components/search/items'
import {
  RATED_RANKING_CIRCULATION_OPTIONS,
  RATED_RANKING_PLAYTIME_OPTIONS,
  RATED_RANKING_SCOPE_OPTIONS,
  RATED_RANKING_SITE_COUNT_OPTIONS,
  RATED_RANKING_VOTE_OPTIONS,
  ratedRankingPresets
} from '~/constants/ratedRanking'

const props = withDefaults(
  defineProps<{
    site: RatingSite | null
    sort: RatedWorkRankingSort
    total?: number | null
    pending?: boolean
    fittedAt?: string | null
    facets?: WorkFacets
  }>(),
  { total: null, pending: false, fittedAt: null, facets: undefined }
)

const { state, set } = useRatedRankingFilters()

const entityNames = useEntityNames()
const suggestions = useEntityFacets(() => props.facets, entityNames)

const presets = computed(() => ratedRankingPresets(props.site))

const presetOptions = computed<FilterOption[]>(() =>
  presets.value.map((p) => ({ value: p.sort, label: p.label, hint: p.hint }))
)

const presetIcon = computed(
  () =>
    presets.value.find((p) => p.sort === props.sort)?.icon ?? 'lucide:trophy'
)

const pickPreset = (sort: string) => {
  const preset = presets.value.find((p) => p.sort === sort)
  if (preset) {
    set({ sort: preset.sort, votes: preset.votes ?? '' })
  }
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

const labelOf = (options: FilterOption[], value: string) =>
  options.find((option) => option.value === value)?.label ?? value

const yearRangeLabel = computed(() => {
  const { releasedFrom: from, releasedTo: to } = state.value
  if (from && to) {
    return from === to ? `${from} 年` : `${from} - ${to}`
  }
  return from ? `${from} 年至今` : `${to} 年以前`
})

const chips = computed<FilterChip[]>(() => {
  const f = state.value
  const list: FilterChip[] = []
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
  if (f.scope) {
    list.push({
      key: 'scope',
      label: labelOf(RATED_RANKING_SCOPE_OPTIONS, f.scope)
    })
  }
  if (f.releasedFrom || f.releasedTo) {
    list.push({ key: 'years', label: yearRangeLabel.value })
  }
  if (f.circulation) {
    list.push({
      key: 'circulation',
      label: labelOf(RATED_RANKING_CIRCULATION_OPTIONS, f.circulation)
    })
  }
  for (const value of f.platforms) {
    list.push({
      key: `platform:${value}`,
      prefix: '平台',
      label: labelOf(PLATFORM_OPTIONS, value)
    })
  }
  if (f.playtime) {
    list.push({
      key: 'playtime',
      prefix: '时长',
      label: labelOf(RATED_RANKING_PLAYTIME_OPTIONS, f.playtime)
    })
  }
  if (f.votes) {
    list.push({
      key: 'votes',
      label: labelOf(RATED_RANKING_VOTE_OPTIONS, f.votes)
    })
  }
  if (f.minSites && !props.site) {
    list.push({
      key: 'minSites',
      label: labelOf(RATED_RANKING_SITE_COUNT_OPTIONS, f.minSites)
    })
  }
  return list
})

const removeChip = (key: string) => {
  const [dimension, value] = key.split(':')
  if (dimension === 'tag') {
    set({ tags: state.value.tags.filter((id) => id !== Number(value)) })
  } else if (dimension === 'company') {
    set({ company: 0 })
  } else if (dimension === 'years') {
    set({ releasedFrom: '', releasedTo: '' })
  } else if (dimension === 'platform') {
    set({ platforms: state.value.platforms.filter((p) => p !== value) })
  } else if (
    dimension === 'scope' ||
    dimension === 'circulation' ||
    dimension === 'playtime' ||
    dimension === 'votes' ||
    dimension === 'minSites'
  ) {
    set({ [dimension]: '' })
  }
}

const clearFilters = () => {
  set({
    tags: [],
    company: 0,
    scope: '',
    releasedFrom: '',
    releasedTo: '',
    circulation: '',
    platforms: [],
    playtime: '',
    votes: '',
    minSites: ''
  })
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
      :icon="presetIcon"
      label="榜单"
      :options="presetOptions"
      :model-value="sort"
      empty-value="lower_desc"
      @update:model-value="pickPreset($event as string)"
    />

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

    <FilterMenu
      icon="lucide:gamepad-2"
      label="作品范围"
      :options="RATED_RANKING_SCOPE_OPTIONS"
      :model-value="state.scope"
      @update:model-value="set({ scope: $event as typeof state.scope })"
    />
    <FilterYears
      :from="state.releasedFrom"
      :to="state.releasedTo"
      @update="set({ releasedFrom: $event.from, releasedTo: $event.to })"
    />
    <FilterMenu
      icon="lucide:building"
      label="发行"
      :options="RATED_RANKING_CIRCULATION_OPTIONS"
      :model-value="state.circulation"
      @update:model-value="
        set({ circulation: $event as typeof state.circulation })
      "
    />
    <FilterMenu
      icon="lucide:monitor-smartphone"
      label="平台"
      multiple
      :columns="2"
      :options="PLATFORM_OPTIONS"
      :model-value="state.platforms"
      @update:model-value="set({ platforms: $event as typeof state.platforms })"
    />
    <FilterMenu
      icon="lucide:clock"
      label="游玩时长"
      :options="RATED_RANKING_PLAYTIME_OPTIONS"
      :model-value="state.playtime"
      @update:model-value="set({ playtime: $event as string })"
    />
    <FilterMenu
      icon="lucide:users"
      label="票数"
      :options="RATED_RANKING_VOTE_OPTIONS"
      :model-value="state.votes"
      @update:model-value="set({ votes: $event as string })"
    />
    <FilterMenu
      v-if="!site"
      icon="lucide:layers"
      label="站点数"
      :options="RATED_RANKING_SITE_COUNT_OPTIONS"
      :model-value="state.minSites"
      @update:model-value="set({ minSites: $event as string })"
    />

    <template #end>
      <RankingRatedMethod :fitted-at="fittedAt" />
    </template>
  </FilterBar>
</template>
