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

defineProps<{
  ratings: GalgameRatingCardOnGalgamePage[]
}>()
</script>

<template>
  <KunScrollShadow
    axis="horizontal"
    shadow-size="5rem"
    wheel="contain"
    draggable
    scrollbar="thin"
  >
    <div class="flex" v-for="rating in ratings" :key="rating.id">
      <KunCard
        :is-transparent="false"
        :is-hoverable="false"
        class-name="max-w-80"
      >
        <div class="flex items-center justify-between gap-3">
          <div class="flex min-w-0 items-center gap-3">
            <KunUserChip
              :disable-floating="true"
              :user="rating.user"
              class-name="min-w-0 flex-1"
            />
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
            v-if="rating.short_summary && rating.spoiler_level !== 'none'"
            class="text-default-500 flex max-h-[110px] items-center gap-1.5 text-sm"
          >
            <KunIcon
              name="lucide:triangle-alert"
              class="text-warning shrink-0"
            />
            {{ KUN_GALGAME_RATING_SPOILER_WARNING }}
          </div>
          <KunScrollShadow
            v-else-if="rating.short_summary"
            axis="vertical"
            shadow-size="3rem"
            class-name="max-h-[110px]"
            class="text-default-700 text-sm"
          >
            <KunText :content="rating.short_summary" />
          </KunScrollShadow>
        </div>

        <div class="text-default-500 flex flex-wrap items-center gap-2">
          <KunChip
            class-name="shrink-0"
            :color="KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP[rating.recommend]"
          >
            {{ KUN_GALGAME_RATING_RECOMMEND_MAP[rating.recommend] }}
          </KunChip>
          <KunChip color="primary">
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
      </KunCard>
    </div>
  </KunScrollShadow>
</template>
