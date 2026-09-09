<script setup lang="ts">
import {
  KUN_GALGAME_DIMENSIONS,
  KUN_GALGAME_DIM_LABELS,
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

const props = defineProps<{
  ratings: GalgameRatingCardOnGalgamePage[]
}>()

const sortBy = ref<'time' | 'overall'>('time')
const expanded = ref<string[]>([])

const sorted = computed(() =>
  [...props.ratings].sort((a, b) =>
    sortBy.value === 'overall'
      ? b.overall - a.overall
      : new Date(b.created).getTime() - new Date(a.created).getTime()
  )
)
</script>

<template>
  <div class="space-y-2">
    <div class="flex flex-wrap items-baseline justify-between gap-2">
      <h4 class="font-medium">
        每位用户的评分
        <span class="text-default-400 text-xs">({{ ratings.length }})</span>
      </h4>

      <div class="flex items-center gap-1">
        <KunButton
          size="sm"
          :variant="sortBy === 'time' ? 'flat' : 'light'"
          :color="sortBy === 'time' ? 'primary' : 'default'"
          @click="sortBy = 'time'"
        >
          最新
        </KunButton>
        <KunButton
          size="sm"
          :variant="sortBy === 'overall' ? 'flat' : 'light'"
          :color="sortBy === 'overall' ? 'primary' : 'default'"
          @click="sortBy = 'overall'"
        >
          分数
        </KunButton>
      </div>
    </div>

    <KunAccordion v-model="expanded" :multiple="true" variant="splitted">
      <KunAccordionItem
        v-for="rating in sorted"
        :key="rating.id"
        :value="String(rating.id)"
      >
        <template #title>
          <span class="flex min-w-0 flex-wrap items-center gap-2">
            <KunUserChip
              :user="rating.user"
              size="sm"
              :is-navigation="false"
              :disable-floating="true"
              class-name="min-w-0"
            />
            <span class="text-warning shrink-0 font-bold">
              {{ rating.overall }}
              <span class="text-default-400 text-xs font-normal">/10</span>
            </span>
            <KunTime
              :time="rating.created"
              class="text-default-400 ml-auto shrink-0 text-xs"
            />
          </span>
        </template>

        <div class="space-y-3">
          <div class="grid grid-cols-2 gap-x-4 gap-y-1.5 sm:grid-cols-4">
            <div v-for="dim in KUN_GALGAME_DIMENSIONS" :key="dim">
              <div class="flex items-baseline justify-between text-xs">
                <span>{{ KUN_GALGAME_DIM_LABELS[dim] }}</span>
                <span class="tabular-nums">{{ rating[dim] }}</span>
              </div>
              <KunProgress :value="rating[dim]" :max="10" size="sm" />
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <KunChip
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
              :color="
                KUN_GALGAME_RATING_SPOILER_COLOR_MAP[rating.spoiler_level]
              "
            >
              {{ KUN_GALGAME_RATING_SPOILER_MAP[rating.spoiler_level] }}
            </KunChip>
          </div>

          <p
            v-if="rating.short_summary && rating.spoiler_level !== 'none'"
            class="text-default-500 flex items-center gap-1.5"
          >
            <KunIcon name="lucide:triangle-alert" class="text-warning" />
            {{ KUN_GALGAME_RATING_SPOILER_WARNING }}
          </p>
          <KunText
            v-else-if="rating.short_summary"
            :content="rating.short_summary"
          />

          <KunLink :to="`/galgame-rating/${rating.id}`" size="sm">
            阅读完整评分 >
          </KunLink>
        </div>
      </KunAccordionItem>
    </KunAccordion>
  </div>
</template>
