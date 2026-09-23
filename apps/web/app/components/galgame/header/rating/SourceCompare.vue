<script setup lang="ts">
import {
  KUN_GALGAME_EXTERNAL_RATING_CONST,
  KUN_GALGAME_EXTERNAL_RATING_MAP,
  KUN_GALGAME_LOCAL_RATING_META,
  KUN_GALGAME_LOCAL_RATING_SOURCE,
  kunGalgameRatingTierBadge
} from '~/constants/galgame-rating'
import { GALGAME_RATING_TIER_CONST } from '~~/shared/utils/galgameRatingTier'
import type { Work } from '#shared/utils/api/schemas'

const props = defineProps<{
  galgame: Work
  highlight: string
}>()

const rows = computed(() => {
  const local = props.galgame.rating_count
    ? [
        {
          key: KUN_GALGAME_LOCAL_RATING_SOURCE,
          label: KUN_GALGAME_LOCAL_RATING_META.label,
          max: KUN_GALGAME_LOCAL_RATING_META.max,
          score: props.galgame.rating_score ?? 0,
          count: props.galgame.rating_count,
          tier: kunGalgameRatingTierBadge(
            KUN_GALGAME_LOCAL_RATING_META,
            props.galgame.rating_score,
            props.galgame.rating_count
          )
        }
      ]
    : []

  const external = KUN_GALGAME_EXTERNAL_RATING_CONST.flatMap((source) => {
    const row = props.galgame.external_ratings.find((r) => r.site === source)
    if (!row) return []
    const meta = KUN_GALGAME_EXTERNAL_RATING_MAP[source]
    return [
      {
        key: source as string,
        label: meta.label,
        max: meta.max,
        score: row.rating_value,
        count: row.vote_count,
        tier: kunGalgameRatingTierBadge(meta, row.rating_value, row.vote_count)
      }
    ]
  })

  return [...local, ...external]
})

const normalized = (score: number, max: number) => (score / max) * 10

const tierSpread = computed(() => {
  const ranks = rows.value.flatMap((row) =>
    row.tier ? [GALGAME_RATING_TIER_CONST.indexOf(row.tier.key)] : []
  )
  return ranks.length > 1 ? Math.max(...ranks) - Math.min(...ranks) : 0
})

const hasDlsite = computed(() => rows.value.some((row) => row.key === 'dlsite'))
</script>

<template>
  <div v-if="rows.length > 1" class="space-y-2">
    <div class="flex items-baseline justify-between">
      <h4 class="font-medium">各来源对比</h4>
      <span class="text-default-500 text-xs">条形图统一折算为满分 10</span>
    </div>

    <div
      v-for="row in rows"
      :key="row.key"
      :class="
        cn(
          'grid grid-cols-[5rem_1fr_auto_2.75rem] items-center gap-2 rounded-lg px-2 py-1.5 sm:grid-cols-[7rem_1fr_auto_3rem] sm:gap-3',
          row.key === highlight && 'bg-default-100'
        )
      "
    >
      <span
        :class="
          cn(
            'truncate text-xs',
            row.key === highlight ? 'font-medium' : 'text-default-500'
          )
        "
      >
        {{ row.label }}
      </span>
      <KunProgress
        :value="normalized(row.score, row.max)"
        :max="10"
        size="sm"
        :color="row.key === highlight ? 'warning' : 'primary'"
      />
      <span class="text-default-500 shrink-0 text-xs tabular-nums">
        {{ normalized(row.score, row.max).toFixed(1) }}
        <span class="text-default-400">
          · {{ row.count.toLocaleString('en-US') }} 人
        </span>
      </span>

      <span class="flex justify-end">
        <KunChip
          v-if="row.tier"
          size="xs"
          variant="flat"
          :color="row.tier.color"
        >
          {{ row.tier.label }}
        </KunChip>
      </span>
    </div>

    <p v-if="tierSpread >= 2" class="text-default-400 text-xs">
      各来源对这部作品的评价差了 {{ tierSpread }} 个等级。评分人群不一样,
      分歧本来就正常{{
        hasDlsite ? ', 其中 DLsite 只统计购买者, 普遍要比其他来源高一档' : ''
      }}。
    </p>
  </div>
</template>
