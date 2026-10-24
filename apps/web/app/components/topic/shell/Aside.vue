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
  { title: string; icon: string; items: TopicSummary[]; hot: boolean }[]
>(() => [
  {
    title: '最新话题',
    icon: 'lucide:sparkles',
    items: latest.value?.items ?? [],
    hot: false
  },
  {
    title: '本周热门',
    icon: 'lucide:flame',
    items: hot.value?.items ?? [],
    hot: true
  }
])
</script>

<template>
  <div class="space-y-4">
    <KunCard
      v-for="block in blocks"
      :key="block.title"
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <h2 class="flex items-center gap-2 text-sm font-semibold">
        <KunIcon :name="block.icon" class="text-primary-500" />
        {{ block.title }}
      </h2>
      <ul v-if="block.items.length" class="space-y-2.5">
        <li v-for="topic in block.items" :key="topic.id" class="space-y-0.5">
          <KunLink
            :to="`/topic/${topic.id}`"
            underline="hover"
            class-name="line-clamp-2 text-sm"
          >
            {{ topic.title }}
          </KunLink>
          <p class="text-default-500 flex items-center gap-2 text-xs">
            <span v-if="block.hot" class="flex items-center gap-1">
              <KunIcon name="lucide:eye" class="size-3" />
              {{ formatNumber(topic.view_count) }}
            </span>
            <KunTime v-else :time="topic.created_at" type="relative" />
          </p>
        </li>
      </ul>
      <p v-else class="text-default-400 text-xs">暂时没有话题</p>
    </KunCard>
  </div>
</template>
