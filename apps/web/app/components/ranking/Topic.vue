<script setup lang="ts">
import { topicSortItem } from '~/constants/ranking'
import { toKunUser } from '~/utils/userRef'
import { RANKING_LIMIT, topicRankingPageData, getRankClasses } from './pageData'

const { allowsNsfw, stanceKey } = useContentStance()

const { data } = await useApi(
  () =>
    `ranking-topics:${topicRankingPageData.sort}:${stanceKey.value}`,
  (client, { signal }) =>
    client.GET('/rankings/topics', {
      params: {
        query: {
          sort: topicRankingPageData.sort,
          limit: RANKING_LIMIT,
          include_nsfw: allowsNsfw.value
        }
      },
      signal
    })
)

const entries = computed(() =>
  (data.value?.items ?? []).flatMap((entry) =>
    entry.topic ? [{ ...entry, topic: entry.topic }] : []
  )
)

const icon = computed(
  () =>
    topicSortItem.find((i) => i.sort === topicRankingPageData.sort)?.icon ?? ''
)
</script>

<template>
  <ul v-if="data" class="space-y-3">
    <li v-for="(entry, index) in entries" :key="entry.topic.id">
      <KunLink
        color="default"
        underline="none"
        :to="`/topic/${entry.topic.id}`"
        :class-name="
          cn(
            'relative flex items-center gap-3 rounded-xl border p-3 transition-colors',
            getRankClasses(index)
          )
        "
      >
        <RankingMedal :index="index" />

        <div class="flex-1">
          <h3 class="truncate font-semibold">{{ entry.topic.title }}</h3>
          <div class="mt-1 flex items-center gap-2">
            <KunAvatar
              :user="toKunUser(entry.topic.author)"
              size="sm"
              :is-navigation="false"
            />
            <span class="text-default-500 text-sm">
              {{ toKunUser(entry.topic.author).name }}
            </span>

            <div class="flex shrink-0 items-center gap-2 sm:hidden">
              <KunIcon :name="icon" class="text-primary h-5 w-5" />
              <span class="text-default-500 text-sm">
                {{ entry.metric_value }}
              </span>
            </div>
          </div>
        </div>

        <div class="hidden shrink-0 items-center gap-2 sm:flex">
          <KunIcon :name="icon" class="text-primary h-5 w-5" />
          <span class="text-lg font-medium">{{ entry.metric_value }}</span>
        </div>
      </KunLink>
    </li>
  </ul>
</template>
