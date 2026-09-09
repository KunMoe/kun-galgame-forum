<script setup lang="ts">
import {
  KUN_GALGAME_DIMENSIONS,
  KUN_GALGAME_DIM_LABELS,
  KUN_GALGAME_LOCAL_RATING_META,
  KUN_GALGAME_LOCAL_RATING_SOURCE,
  KUN_GALGAME_RATING_RECOMMEND_MAP,
  KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP,
  KUN_GALGAME_RATING_TIER_INSUFFICIENT,
  KUN_GALGAME_RATING_TIER_SCOPE_HINT,
  kunGalgameRatingTierBadge
} from '~/constants/galgame-rating'
import {
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'
import {
  ratingDimensionMeans,
  ratingHistogram,
  ratingMean,
  ratingTally
} from './_stats'

const props = defineProps<{
  galgame: GalgameDetail
  ratings: GalgameRatingCardOnGalgamePage[]
}>()

const { id: currentUserId } = usePersistUserStore()

const mean = computed(() =>
  ratingMean(props.ratings.map((rating) => rating.overall))
)
const buckets = computed(() => ratingHistogram(props.ratings))
const categories = KUN_GALGAME_LOCAL_RATING_META.histogram.keys.map(
  KUN_GALGAME_LOCAL_RATING_META.histogram.label
)
const dimensionMeans = computed(() => ratingDimensionMeans(props.ratings))
const mine = computed(
  () =>
    props.ratings.find((rating) => rating.user.id === currentUserId)?.overall ??
    null
)
const tier = computed(() =>
  kunGalgameRatingTierBadge(
    KUN_GALGAME_LOCAL_RATING_META,
    props.galgame.rating,
    props.ratings.length
  )
)

const tierTooltip = computed(() =>
  tier.value
    ? `${tier.value.description}, ${KUN_GALGAME_RATING_TIER_SCOPE_HINT}`
    : `本站评分人数不到 ${KUN_GALGAME_LOCAL_RATING_META.tier.minVotes} 人, 分级会被少数几票带偏, 所以这里不给等级`
)

const recommendTally = computed(() =>
  ratingTally(props.ratings.map((rating) => rating.recommend))
)
const playStatusTally = computed(() =>
  ratingTally(props.ratings.map((rating) => rating.play_status))
)
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-wrap items-end gap-x-6 gap-y-3">
      <div>
        <div class="flex items-center gap-2">
          <div class="flex items-baseline gap-1">
            <KunIcon name="lucide:star" class="text-warning" />
            <span class="flex items-baseline gap-0.5">
              <span class="text-3xl leading-none font-semibold tabular-nums">
                {{
                  KUN_GALGAME_LOCAL_RATING_META.formatScore(galgame.rating ?? 0)
                }}
              </span>
              <span class="text-default-500 text-xl leading-none font-medium">
                {{ KUN_GALGAME_LOCAL_RATING_META.scoreSuffix }}
              </span>
            </span>
          </div>
          <KunTooltip :text="tierTooltip">
            <KunChip
              size="sm"
              variant="flat"
              :color="tier ? tier.color : 'default'"
            >
              {{ tier ? tier.label : KUN_GALGAME_RATING_TIER_INSUFFICIENT }}
            </KunChip>
          </KunTooltip>
        </div>
        <KunTooltip
          text="贝叶斯平均: 评分人数少时会向全站均分收敛, 避免一两个满分就冲榜"
        >
          <span class="text-default-500 text-xs">
            贝叶斯平均
            <KunIcon name="lucide:circle-help" class="size-3" />
          </span>
        </KunTooltip>
      </div>

      <div class="text-default-500 flex gap-6 text-sm">
        <span>
          原始均分
          <span class="text-foreground font-medium tabular-nums">
            {{ mean.toFixed(2) }}
          </span>
        </span>
        <span>
          评分人数
          <span class="text-foreground font-medium tabular-nums">
            {{ ratings.length }}
          </span>
        </span>
        <span v-if="mine != null">
          我的评分
          <span class="text-warning font-medium tabular-nums">{{ mine }}</span>
        </span>
      </div>
    </div>

    <div class="space-y-1">
      <h4 class="font-medium">评分分布</h4>
      <GalgameHeaderRatingDistributionChart
        :galgame-id="galgame.id"
        :source="KUN_GALGAME_LOCAL_RATING_SOURCE"
        :buckets="buckets"
        :categories="categories"
        :mine-index="mine == null ? -1 : mine - 1"
      />
    </div>

    <div class="flex flex-col items-center gap-4 sm:flex-row">
      <GalgameRatingRadar
        :model-value="dimensionMeans"
        :size="180"
        :readonly="true"
        label-class="text-[10px]"
        class="shrink-0"
      />
      <div class="grid w-full grid-cols-2 gap-x-4 gap-y-1.5">
        <div v-for="dim in KUN_GALGAME_DIMENSIONS" :key="dim">
          <div class="flex items-baseline justify-between text-xs">
            <span>{{ KUN_GALGAME_DIM_LABELS[dim] }}</span>
            <span class="tabular-nums">{{
              dimensionMeans[dim].toFixed(1)
            }}</span>
          </div>
          <KunProgress :value="dimensionMeans[dim]" :max="10" size="sm" />
        </div>
      </div>
    </div>

    <div class="flex flex-wrap gap-4">
      <div class="space-y-1.5">
        <h4 class="font-medium">推荐度</h4>
        <div class="flex flex-wrap gap-1.5">
          <KunChip
            v-for="[recommend, count] in recommendTally"
            :key="recommend"
            :color="KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP[recommend]"
          >
            {{ KUN_GALGAME_RATING_RECOMMEND_MAP[recommend] }} · {{ count }}
          </KunChip>
        </div>
      </div>

      <div class="space-y-1.5">
        <h4 class="font-medium">游玩进度</h4>
        <div class="flex flex-wrap gap-1.5">
          <KunChip
            v-for="[status, count] in playStatusTally"
            :key="status"
            color="secondary"
          >
            {{ KUN_GALGAME_PLAY_STATE_MAP[status as KunGalgamePlayStateRead] }}
            · {{ count }}
          </KunChip>
        </div>
      </div>
    </div>

    <GalgameHeaderRatingSourceCompare
      :galgame="galgame"
      :highlight="KUN_GALGAME_LOCAL_RATING_SOURCE"
    />

    <GalgameHeaderRatingUserList :ratings="ratings" />
  </div>
</template>
