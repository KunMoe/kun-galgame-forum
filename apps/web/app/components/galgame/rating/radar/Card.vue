<script setup lang="ts">
import {
  KUN_GALGAME_RATING_RECOMMEND_MAP,
  KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP,
  KUN_GALGAME_RATING_SPOILER_MAP,
  KUN_GALGAME_RATING_SPOILER_COLOR_MAP,
  KUN_GALGAME_RATING_SPOILER_WARNING
} from '~/constants/galgame-rating'
import {
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'
import { isRatingSummaryShown } from '~/utils/galgame/ratingCard'

const props = withDefaults(
  defineProps<{
    ratings: GalgameRatingCardOnGalgamePage[]
    moreCount?: number
  }>(),
  { moreCount: 0 }
)

const emit = defineEmits<{
  more: []
}>()

const isMoreTeaser = (index: number) =>
  props.moreCount > 0 && index === props.ratings.length - 1
</script>

<template>
  <KunScrollShadow
    axis="horizontal"
    shadow-size="5rem"
    wheel="contain"
    draggable
    scrollbar="thin"
  >
    <div class="flex" v-for="(rating, index) in ratings" :key="rating.id">
      <KunCard
        :is-transparent="false"
        :is-hoverable="false"
        class-name="max-w-80"
      >
        <div class="flex items-center justify-between gap-3">
          <div class="flex min-w-0 items-center gap-3">
            <KunUserChip :user="rating.user" class-name="min-w-0 flex-1" />
            <span class="text-default-500 shrink-0 text-sm">
              <KunTime :time="rating.created" />
            </span>
          </div>

          <div class="flex items-center gap-2">
            <span
              class="text-warning flex shrink-0 items-center text-xl font-bold"
            >
              {{ `${rating.overall}` }}
              <span class="text-default-500 ml-0.5 text-sm">/10</span>
            </span>
          </div>
        </div>

        <div class="flex gap-2">
          <GalgameRatingRadar
            :model-value="{ ...rating }"
            :size="100"
            :readonly="true"
            label-class="text-[10px]"
          />
          <div
            v-if="rating.short_summary_html && !isRatingSummaryShown(rating)"
            class="text-default-500 flex max-h-[110px] items-center gap-1.5 text-sm"
          >
            <KunIcon
              name="lucide:triangle-alert"
              class="text-warning shrink-0"
            />
            {{ KUN_GALGAME_RATING_SPOILER_WARNING }}
          </div>
          <KunScrollShadow
            v-else-if="rating.short_summary_html"
            axis="vertical"
            shadow-size="3rem"
            class-name="max-h-[110px]"
            class="text-default-700 text-sm"
          >
            <KunContent :content="rating.short_summary_html" compact />
          </KunScrollShadow>
        </div>

        <div class="text-default-500 flex flex-wrap items-center gap-2">
          <KunChip
            class-name="shrink-0"
            :color="KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP[rating.recommend]"
          >
            {{ KUN_GALGAME_RATING_RECOMMEND_MAP[rating.recommend] }}
          </KunChip>
          <KunChip v-if="rating.play_status" color="primary">
            {{
              KUN_GALGAME_PLAY_STATE_MAP[
                rating.play_status as KunGalgamePlayStateRead
              ]
            }}
          </KunChip>
          <KunChip
            :color="KUN_GALGAME_RATING_SPOILER_COLOR_MAP[rating.spoiler_level]"
          >
            {{ KUN_GALGAME_RATING_SPOILER_MAP[rating.spoiler_level] }}
          </KunChip>
        </div>

        <div class="text-default-500 flex flex-wrap items-center gap-3 text-xs">
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:eye" />
            {{ rating.view }}
          </span>
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:thumbs-up" />
            {{ rating.like_count }}
          </span>
          <KunLink
            :to="`/galgame-rating/${rating.id}`"
            size="sm"
            class-name="ml-auto"
          >
            阅读详情 >
          </KunLink>
        </div>

        <!-- 铁律 #1 sanctioned exception (e): the last preview rating fades
             into the card so it reads as "more behind this". Listed in
             CLAUDE.md; do NOT remove it in a no-gradient sweep. -->
        <button
          v-if="isMoreTeaser(index)"
          type="button"
          class="text-primary absolute inset-0 z-10 flex cursor-pointer items-center justify-end rounded-[inherit] bg-gradient-to-r from-transparent from-10% to-[oklch(var(--content1))] to-70% pr-6 text-sm font-medium"
          @click="emit('more')"
        >
          <span class="flex items-center gap-1">
            查看全部 {{ moreCount }} 条评分
            <KunIcon name="lucide:chevron-right" />
          </span>
        </button>
      </KunCard>
    </div>
  </KunScrollShadow>
</template>
