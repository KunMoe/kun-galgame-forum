<script setup lang="ts">
import {
  kunUserReplyNavItem,
  type KUN_USER_PAGE_REPLY_TYPE
} from '~/constants/user'
import type { PageListUserReplyItem } from '#shared/utils/api/schemas'

const REPLY_RELATION = {
  reply_created: 'authored',
  reply_target: 'received',
  reply_like: 'liked'
} as const

const props = defineProps<{
  userId: number
  type: (typeof KUN_USER_PAGE_REPLY_TYPE)[number]
}>()

const { allowsNsfw: includeNsfw } = useContentStance()

const activeTab = computed(() => props.type)
const relation = computed(() => REPLY_RELATION[props.type])
const pageData = reactive({
  page: usePageQuery(),
  limit: 50
})

const { data, status } = await useApi<PageListUserReplyItem>(
  () =>
    `user-replies:${props.userId}:${relation.value}:${pageData.page}:${pageData.limit}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (client, { signal }) =>
    client.GET('/users/{user_id}/replies', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          relation: relation.value,
          page: pageData.page,
          limit: pageData.limit,
          include_nsfw: includeNsfw.value
        }
      },
      signal
    })
)
</script>

<template>
  <div class="space-y-6">
    <KunTab
      :items="kunUserReplyNavItem(userId)"
      :model-value="activeTab"
      variant="light"
      size="sm"
      scrollable
    />

    <div v-if="data" class="flex flex-col space-y-3">
      <KunCard
        v-for="(reply, index) in data.items"
        :key="index"
        :href="replyPermalink(`/topic/${reply.topic_id}`, reply.floor)"
      >
        <div>
          {{ reply.excerpt }}
        </div>
        <div class="text-default-500 text-sm">
          <KunTime :time="reply.created_at" type="date" show-year />
        </div>
      </KunCard>

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull
      v-if="data && !data.items.length"
      description="这只笨蛋萝莉没有发布过任何回复"
    />
  </div>
</template>
