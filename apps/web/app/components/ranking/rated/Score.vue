<script setup lang="ts">
import type {
  RatedWorkRankingEntry,
  RatedWorkRankingSort,
  RatingSite
} from '#shared/utils/api/schemas'
import {
  formatDeviation,
  formatSiteScore,
  formatSiteValue,
  formatVotes,
  percentileTone
} from './format'

const props = defineProps<{
  entry: RatedWorkRankingEntry
  site: RatingSite | null
  sort: RatedWorkRankingSort
}>()

const score = computed(() => {
  const { consensus, site_ratings } = props.entry
  if (!props.site) {
    if (!consensus) {
      return null
    }
    return {
      value: formatDeviation(consensus.deviation),
      suffix: '',
      tone: percentileTone(consensus.percentile),
      interval: {
        lower: consensus.lower,
        upper: consensus.upper,
        value: consensus.deviation,
        min: 0,
        max: 100
      },
      range: `区间 ${formatDeviation(consensus.lower)} – ${formatDeviation(consensus.upper)}`,
      tip: `90% 置信区间 ${formatDeviation(consensus.lower)} – ${formatDeviation(consensus.upper)}`
    }
  }
  const r = site_ratings.find((s) => s.site === props.site)
  if (!r) {
    return null
  }
  return {
    value: formatSiteScore(r.published_score, r.scale_max),
    suffix: `/${r.scale_max}`,
    tone: percentileTone(r.percentile),
    interval: {
      lower: r.lower,
      upper: r.upper,
      value: r.estimate,
      min: r.scale_min,
      max: r.scale_max
    },
    range: `估计 ${formatSiteValue(r, r.estimate)}`,
    tip: `模型估计 ${formatSiteValue(r, r.estimate)}, 90% 置信区间 ${formatSiteValue(r, r.lower)} – ${formatSiteValue(r, r.upper)}`
  }
})

const caption = computed(() => {
  const value = props.entry.metric_value
  switch (props.sort) {
    case 'divergence_desc':
      return `分歧 ${value.toFixed(1)} 点`
    case 'votes_desc':
      return `${formatVotes(value)} 票`
    case 'spread_desc':
      return `标准差 ${value.toFixed(2)}`
    default:
      return score.value?.range ?? ''
  }
})
</script>

<template>
  <div v-if="score" class="flex w-16 shrink-0 flex-col items-end gap-1 sm:w-28">
    <span
      :class="cn('text-xl leading-none font-bold tabular-nums', score.tone)"
    >
      {{ score.value }}
      <span v-if="score.suffix" class="text-default-400 text-xs font-normal">
        {{ score.suffix }}
      </span>
    </span>
    <KunTooltip :text="score.tip" position="left" class-name="w-full">
      <RankingRatedInterval v-bind="score.interval" class="mt-0.5" />
    </KunTooltip>
    <span class="text-default-500 hidden text-xs tabular-nums sm:block">
      {{ caption }}
    </span>
  </div>
</template>
