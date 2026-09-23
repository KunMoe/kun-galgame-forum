<script setup lang="ts">
import type { PageListUserTopicItem } from '#shared/utils/api/schemas'

const props = defineProps<{
  userId: number
}>()

const page = usePageQuery()
const limit = 24
const { allowsNsfw: includeNsfw } = useContentStance()

const { data, status } = await useApi<PageListUserTopicItem>(
  () =>
    `user-topics:${props.userId}:favorited:${page.value}:${limit}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (client, { signal }) =>
    client.GET('/users/{user_id}/topics', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          relation: 'favorited',
          page: page.value,
          limit,
          include_nsfw: includeNsfw.value
        }
      },
      signal
    })
)
</script>

<template>
  <div class="space-y-3">
    <div v-if="data && data.items.length" class="flex flex-col space-y-3">
      <KunCard
        v-for="topic in data.items"
        :key="topic.id"
        :href="`/topic/${topic.id}`"
      >
        <div>{{ topic.title }}</div>
        <div class="text-default-500 text-sm">
          <KunTime :time="topic.created_at" type="date" show-year />
        </div>
      </KunCard>

      <KunPagination
        v-if="data.total > limit"
        v-model:current-page="page"
        :total-page="Math.ceil(data.total / limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull
      v-if="data && !data.items.length"
      description="这只笨蛋萝莉还没有收藏任何话题"
    />
  </div>
</template>
