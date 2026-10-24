<script setup lang="ts">
import type { TopicSummary } from '#shared/utils/api/schemas'

const ASIDE_LIMIT = 6

const { allowsNsfw } = useContentStance()

const list = (sort: 'created_desc' | 'views_7d_desc') =>
  useApi(
    () => `topic-aside:${sort}:${allowsNsfw.value ? 'nsfw' : 'sfw'}`,
    (api, { signal }) =>
      api.GET('/topics', {
        params: {
          query: {
            sort,
            limit: ASIDE_LIMIT,
            hidden_sections: 'exclude',
            include_nsfw: allowsNsfw.value
          }
        },
        signal
      }),
    { server: false, lazy: true }
  )

const { data: latest } = list('created_desc')
const { data: hot } = list('views_7d_desc')

const blocks = computed<
  { title: string; items: TopicSummary[]; hot: boolean }[]
>(() => [
  {
    title: '最新话题',
    items: latest.value?.items ?? [],
    hot: false
  },
  {
    title: '本周热门',
    items: hot.value?.items ?? [],
    hot: true
  }
])
</script>

<template>
  <div class="divide-default-200/60 space-y-4 divide-y">
    <section
      v-for="block in blocks"
      :key="block.title"
      class="space-y-1.5 pb-4 last:pb-0"
    >
      <h2 class="px-2 text-base font-semibold">
        {{ block.title }}
      </h2>
      <ol v-if="block.items.length" class="space-y-0.5">
        <li v-for="topic in block.items" :key="topic.id">
          <NuxtLink
            :to="`/topic/${topic.id}`"
            class="hover:bg-default-100 block rounded-lg px-2 py-1.5 transition-colors"
          >
            <div class="space-y-0.5">
              <p class="line-clamp-2 text-sm leading-snug">
                {{ topic.title }}
              </p>
              <p class="text-default-500 flex items-center gap-1 text-xs">
                <template v-if="block.hot">
                  <KunIcon name="lucide:eye" class="size-3" />
                  {{ formatNumber(topic.view_count) }}
                </template>
                <KunTime v-else :time="topic.created_at" type="relative" />
              </p>
            </div>
          </NuxtLink>
        </li>
      </ol>
      <p v-else class="text-default-400 px-2 text-xs">暂时没有话题</p>
    </section>
  </div>
</template>
