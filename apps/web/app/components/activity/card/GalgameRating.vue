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
import type {
  Activity,
  ActivityRating,
  WorkRef
} from '#shared/utils/api/schemas'

const props = defineProps<{
  activity: Activity
  work: WorkRef
  rating: ActivityRating
}>()

const nameOf = useWorkName()

const playStatusLabel = computed(
  () =>
    KUN_GALGAME_PLAY_STATE_MAP[
      props.rating.play_status as KunGalgamePlayStateRead
    ] || props.rating.play_status
)
const recommendLabel = computed(
  () =>
    KUN_GALGAME_RATING_RECOMMEND_MAP[props.rating.recommend] ||
    props.rating.recommend
)
const recommendColor = computed(() => {
  switch (KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP[props.rating.recommend]) {
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
const overall = computed(() => props.rating.overall.toFixed(1))
const hasSpoiler = computed(() => props.rating.spoiler_level !== 'none')
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div class="space-y-3">
      <p class="text-default-600 flex items-center gap-1 text-sm">
        <span>评分了一个 Galgame，评分</span>
        <span
          class="text-default-800 inline-flex items-center gap-0.5 font-semibold"
        >
          <KunIcon name="lucide:star" class="text-warning size-4" />
          {{ overall }}
        </span>
      </p>

      <div class="flex items-start gap-3">
        <ActivityCardWorkCover :work="work" />

        <div class="min-w-0 flex-1 space-y-1.5">
          <KunLink
            underline="none"
            color="default"
            :to="`/galgame/${work.id}`"
            class-name="hover:text-primary block"
          >
            <h3 class="line-clamp-2 font-medium break-all">
              {{ nameOf(work) }}
            </h3>
          </KunLink>

          <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
            <span class="text-default-700">{{ playStatusLabel }}</span>
            <span :class="cn('font-medium', recommendColor)">
              {{ recommendLabel }}
            </span>
          </div>

          <p
            v-if="hasSpoiler"
            class="text-default-500 inline-flex items-center gap-1 text-sm"
          >
            <KunIcon name="lucide:eye-off" class="size-4 shrink-0" />
            {{ KUN_GALGAME_RATING_SPOILER_WARNING }}
          </p>
          <p
            v-else-if="rating.short_summary"
            class="text-default-700 line-clamp-3 text-base break-all"
          >
            {{ rating.short_summary }}
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <GalgameRatingDetailLike
          :rating-id="Number(rating.rating_id)"
          :target-user-id="Number(activity.performer?.id ?? 0)"
          :like-count="rating.like_count"
          :is-liked="false"
        />
        <KunLink
          underline="none"
          color="default"
          :to="activity.path"
          class-name="text-default-500 hover:text-primary ml-auto flex items-center gap-0.5 text-sm"
        >
          查看详情
          <KunIcon name="lucide:chevron-right" class="size-4" />
        </KunLink>
      </div>
    </div>
  </ActivityCardShell>
</template>
