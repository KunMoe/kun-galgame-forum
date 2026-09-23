<script setup lang="ts">
import {
  KUN_GALGAME_EXTERNAL_RATING_CONST,
  KUN_GALGAME_EXTERNAL_RATING_MAP,
  KUN_GALGAME_LOCAL_RATING_META,
  KUN_GALGAME_LOCAL_RATING_SOURCE,
  KUN_GALGAME_RATING_TIER_INSUFFICIENT,
  KUN_GALGAME_RATING_TIER_SCOPE_HINT,
  kunGalgameRatingTierBadge,
  type KunGalgameExternalRatingMeta
} from '~/constants/galgame-rating'
import type { Work } from '#shared/utils/api/schemas'

const props = defineProps<{
  galgame: Work
}>()

const emits = defineEmits<{
  openRating: []
  openDetail: [string]
}>()

const forumCount = computed(() => props.galgame.rating_count)

const buildTile = (
  key: string,
  meta: KunGalgameExternalRatingMeta,
  score: number,
  voteCount: number
) => {
  const tier = kunGalgameRatingTierBadge(meta, score, voteCount)
  return {
    key,
    short: meta.short,
    score: meta.formatScore(score),
    suffix: meta.scoreSuffix,
    votes: `${voteCount.toLocaleString('en-US')} 人`,
    tier,
    tooltip: [
      `${meta.label} · ${meta.hint}`,
      tier
        ? `${tier.label} · ${tier.description}`
        : KUN_GALGAME_RATING_TIER_INSUFFICIENT,
      KUN_GALGAME_RATING_TIER_SCOPE_HINT
    ].join(', ')
  }
}

const tiles = computed(() => {
  const local = forumCount.value
    ? [
        buildTile(
          KUN_GALGAME_LOCAL_RATING_SOURCE,
          KUN_GALGAME_LOCAL_RATING_META,
          props.galgame.rating_score ?? 0,
          forumCount.value
        )
      ]
    : []

  const external = KUN_GALGAME_EXTERNAL_RATING_CONST.flatMap((source) => {
    const row = props.galgame.external_ratings.find((r) => r.site === source)
    if (!row) return []
    return [
      buildTile(
        source,
        KUN_GALGAME_EXTERNAL_RATING_MAP[source],
        row.rating_value,
        row.vote_count
      )
    ]
  })

  return [...local, ...external]
})

const tileClass =
  'bg-default-100 hover:bg-default-200 flex h-full min-w-36 shrink-0 flex-col justify-between gap-1.5 rounded-lg px-3 py-2 text-left transition-colors'

// The five sources have no logo mark between them — VNDB, 批评空间 and DLsite
// are wordmarks and 批评空间 has no mark at all — so the source is identified by
// a wordmark badge. It replaces the full caption that used to sit under the
// score, which is the width the tier chip now occupies.
const markClass =
  'bg-default-200 text-default-600 rounded-md px-1.5 py-0.5 text-[10px] leading-4 font-medium tracking-wide'
</script>

<template>
  <KunScrollShadow
    axis="horizontal"
    shadow-size="1.5rem"
    :wheel="true"
    content-class="flex items-stretch gap-2"
  >
    <button
      v-if="!forumCount"
      type="button"
      :class="tileClass"
      @click="emits('openRating')"
    >
      <span
        class="text-default-600 flex items-center gap-1 text-sm font-medium"
      >
        <KunIcon name="lucide:star" class="text-warning" />
        还没有评分
      </span>
      <span class="text-primary text-xs">来评第一个</span>
    </button>

    <KunTooltip
      v-for="tile in tiles"
      :key="tile.key"
      :text="tile.tooltip"
      class-name="shrink-0"
    >
      <button
        type="button"
        :class="tileClass"
        @click="emits('openDetail', tile.key)"
      >
        <span class="flex items-center justify-between gap-2">
          <span :class="markClass">{{ tile.short }}</span>
          <KunChip
            v-if="tile.tier"
            size="xs"
            variant="flat"
            :color="tile.tier.color"
          >
            {{ tile.tier.label }}
          </KunChip>
        </span>

        <span class="flex items-baseline justify-between gap-2">
          <span class="flex items-baseline gap-0.5">
            <span class="text-2xl leading-none font-semibold tabular-nums">
              {{ tile.score }}
            </span>
            <span class="text-default-500 text-sm leading-none font-medium">
              {{ tile.suffix }}
            </span>
          </span>
          <span class="text-default-500 text-xs whitespace-nowrap tabular-nums">
            {{ tile.votes }}
          </span>
        </span>
      </button>
    </KunTooltip>
  </KunScrollShadow>
</template>
