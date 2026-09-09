<script setup lang="ts">
import {
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'
import {
  KUN_GALGAME_RATING_RECOMMEND_MAP,
  KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP,
  KUN_GALGAME_RATING_SPOILER_WARNING
} from '~/constants/galgame-rating'

const props = defineProps<{
  rating: GalgameRatingCardOnGalgamePage
}>()

const playStatusLabel = computed(
  () =>
    KUN_GALGAME_PLAY_STATE_MAP[
      props.rating.play_status as KunGalgamePlayStateRead
    ] ||
    props.rating.play_status
)

const recommendLabel = computed(
  () =>
    KUN_GALGAME_RATING_RECOMMEND_MAP[props.rating.recommend] ||
    props.rating.recommend
)

const recommendColor = computed(() => {
  const c = KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP[props.rating.recommend]
  switch (c) {
    case 'danger':
      return 'text-danger'
    case 'success':
      return 'text-success'
    case 'warning':
      return 'text-warning'
    case 'secondary':
      return 'text-secondary'
    default:
      return 'text-default-600'
  }
})

const MAX_SUMMARY = 100

const truncatedSummary = computed(() => {
  const s = props.rating.short_summary?.trim()
  if (!s) return ''
  if (s.length <= MAX_SUMMARY) return s
  return s.slice(0, MAX_SUMMARY) + '...'
})

const overall = computed(() => props.rating.overall.toFixed(1))
</script>

<template>
  <NuxtLink
    :to="`/galgame-rating/${rating.id}`"
    class="hover:bg-default-100/50 group block rounded-md px-2 py-2 transition-colors"
  >
    <div class="flex flex-wrap items-center gap-2 text-sm">
      <KunAvatar :user="rating.user" size="sm" :is-navigation="false" />
      <span class="text-default-800 font-medium">{{ rating.user.name }}</span>
      <span class="text-default-500">
        <template v-if="rating.play_status === 'wish'">
          还未开始游玩此游戏
        </template>
        <template v-else>
          <span class="text-default-700">{{ playStatusLabel }}</span>
          了此游戏
        </template>
        ，表示
        <span :class="cn('font-medium', recommendColor)">
          {{ recommendLabel }}
        </span>
        ，评分
        <span class="text-default-800 font-semibold">{{ overall }}</span>
      </span>
    </div>

    <p
      v-if="rating.short_summary && rating.spoiler_level !== 'none'"
      class="text-default-500 mt-1 ml-8 flex items-center gap-1 text-sm"
    >
      <KunIcon name="lucide:triangle-alert" class="text-warning shrink-0" />
      {{ KUN_GALGAME_RATING_SPOILER_WARNING }}
    </p>
    <p v-else-if="truncatedSummary" class="text-default-500 mt-1 ml-8 text-sm">
      {{ truncatedSummary }}
    </p>
  </NuxtLink>
</template>
