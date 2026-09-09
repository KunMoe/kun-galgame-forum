<script setup lang="ts">
import { calcGalgameRating } from '~~/algorithms/GalgameRatingAlg'
import {
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayState
} from '~/constants/galgame-playtime'
import {
  KUN_GALGAME_RATING_SPOILER_COLOR_MAP,
  KUN_GALGAME_RATING_SPOILER_MAP
} from '~/constants/galgame-rating'

withDefaults(
  defineProps<{
    ratings: GalgameRatingCard[]
    isTransparent?: boolean
  }>(),
  { isTransparent: true }
)

const systemRating = (rating: GalgameRatingCard) => {
  const res = calcGalgameRating(
    { ...rating },
    rating.overall,
    rating.play_status as KunGalgamePlayState,
    rating.recommend as 'no'
  )
  return res
}
</script>

<template>
  <div
    class="grid grid-cols-2 gap-2 sm:grid-cols-2 sm:gap-3 lg:grid-cols-3 xl:grid-cols-4"
  >
    <KunCard
      v-for="rating in ratings"
      :key="rating.id"
      :is-transparent="isTransparent"
      :is-hoverable="true"
      :href="`/galgame-rating/${rating.id}`"
    >
      <div class="flex justify-between">
        <GalgameRatingRadar
          :model-value="{ ...rating }"
          :size="100"
          :readonly="true"
          label-class="text-[10px]"
        />

        <div class="flex flex-col items-end gap-1">
          <span class="text-warning flex items-center gap-1">
            <KunIcon class="text-xl" name="lucide:lollipop" />
            <span class="text-lg font-bold">
              {{ systemRating(rating) }}
            </span>
          </span>

          <span class="text-default-500 text-xs">
            初始分
            <span>
              {{ rating.overall }}
            </span>
          </span>
        </div>
      </div>

      <div class="flex flex-col gap-2">
        <h2
          class="hover:text-primary line-clamp-2 font-medium transition-colors"
        >
          {{ rating.galgame.name }}
        </h2>

        <div class="text-default-700 flex items-center gap-1 text-sm">
          <KunAvatar
            :disable-floating="true"
            :user="rating.user"
            size="xs"
            :is-navigation="false"
          />
          {{ rating.user.name }} - <KunTime :time="rating.created" />
        </div>

        <div class="flex items-center justify-between text-sm">
          <div class="text-default-500 flex gap-3">
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:eye" />
              <span>{{ rating.view }}</span>
            </span>

            <span class="flex items-center gap-1">
              <KunIcon name="lucide:thumbs-up" />
              <span>{{ rating.like_count }}</span>
            </span>
          </div>

          <KunChip variant="solid" color="success">
            {{
              KUN_GALGAME_PLAY_STATE_MAP[
                rating.play_status as KunGalgamePlayState
              ]
            }}
          </KunChip>
        </div>

        <KunChip
          :color="KUN_GALGAME_RATING_SPOILER_COLOR_MAP[rating.spoiler_level]"
        >
          {{ KUN_GALGAME_RATING_SPOILER_MAP[rating.spoiler_level] }}
        </KunChip>
      </div>
    </KunCard>
  </div>
</template>
