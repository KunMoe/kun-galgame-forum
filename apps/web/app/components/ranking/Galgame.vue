<script setup lang="ts">
import { galgameSortItem } from '~/constants/ranking'
import { toKunUser } from '~/utils/userRef'
import { catalogNameText } from '~/utils/catalogName'
import {
  RANKING_LIMIT,
  galgameRankingPageData,
  getRankClasses
} from './pageData'

const settings = usePersistSettingsStore()
const { allowsNsfw, stanceKey } = useContentStance()

const { data } = await useApi(
  () =>
    `ranking-works:${galgameRankingPageData.sort}:${stanceKey.value}:${settings.showKUNGalgameNoResource ? 'all' : 'resourced'}`,
  (client, { signal }) =>
    client.GET('/rankings/works', {
      params: {
        query: {
          sort: galgameRankingPageData.sort,
          limit: RANKING_LIMIT,
          include_nsfw: allowsNsfw.value,
          include_resourceless: settings.showKUNGalgameNoResource
        }
      },
      signal
    })
)

const entries = computed(() =>
  (data.value?.items ?? []).flatMap((entry) =>
    entry.work ? [{ ...entry, work: entry.work }] : []
  )
)

const icon = computed(
  () =>
    galgameSortItem.find((i) => i.sort === galgameRankingPageData.sort)?.icon ??
    ''
)
</script>

<template>
  <ul v-if="data" class="space-y-3">
    <li v-for="(entry, index) in entries" :key="entry.work.id">
      <KunLink
        color="default"
        underline="none"
        :to="`/galgame/${entry.work.id}`"
        :class-name="
          cn(
            'relative flex border border-default/20 items-center gap-3 rounded-xl p-3 transition-colors',
            getRankClasses(index)
          )
        "
      >
        <RankingMedal :index="index" />

        <div
          class="bg-default-100 aspect-5/7 h-16 shrink-0 overflow-hidden rounded-md bg-cover bg-center"
          :style="
            entry.work.cover
              ? { backgroundImage: `url(${entry.work.cover.url})` }
              : undefined
          "
        />
        <div class="flex-1">
          <div class="flex flex-col items-start justify-between gap-3">
            <h2 class="font-semibold">
              {{
                catalogNameText(
                  entry.work,
                  settings.showKUNGalgamePreferOriginalName
                )
              }}
            </h2>
            <div class="mt-1 flex items-center gap-2">
              <template v-if="entry.creator">
                <KunAvatar
                  :user="toKunUser(entry.creator)"
                  size="sm"
                  :is-navigation="false"
                />
                <span class="text-default-500 text-sm">
                  {{ toKunUser(entry.creator).name }}
                </span>
              </template>

              <div class="flex shrink-0 items-center gap-2 sm:hidden">
                <KunIcon :name="icon" class="text-primary" />
                <span class="text-default-500 text-sm">
                  {{ entry.metric_value }}
                </span>
              </div>
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
