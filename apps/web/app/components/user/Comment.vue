<script setup lang="ts">
import {
  kunUserCommentNavItem,
  type KUN_USER_PAGE_COMMENT_TYPE
} from '~/constants/user'
import type { PageListUserCommentItem } from '#shared/utils/api/schemas'

const COMMENT_RELATION = {
  comment_created: 'authored',
  comment_target: 'received',
  comment_like: 'liked'
} as const

const props = defineProps<{
  userId: number
  type: (typeof KUN_USER_PAGE_COMMENT_TYPE)[number]
}>()

const { allowsNsfw: includeNsfw } = useContentStance()

const activeTab = computed(() => props.type)
const relation = computed(() => COMMENT_RELATION[props.type])
const pageData = reactive({
  page: usePageQuery(),
  limit: 50
})

const { data, status } = await useApi<PageListUserCommentItem>(
  () =>
    `user-comments:${props.userId}:${relation.value}:${pageData.page}:${pageData.limit}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (client, { signal }) =>
    client.GET('/users/{user_id}/comments', {
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
      :items="kunUserCommentNavItem(userId)"
      :model-value="activeTab"
      variant="light"
      size="sm"
      scrollable
    />

    <div class="flex flex-col space-y-3" v-if="data && data.items.length">
      <KunCard
        v-for="(comment, index) in data.items"
        :key="index"
        :href="commentPermalink(`/topic/${comment.topic_id}`, comment.id)"
      >
        <div>
          {{ comment.excerpt }}
        </div>
        <div class="text-default-500 text-sm">
          <KunTime :time="comment.created_at" type="date" show-year />
        </div>
      </KunCard>

      <KunPagination
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull
      v-if="data && !data.items.length"
      description="这只笨蛋萝莉没有发布过任何评论"
    />
  </div>
</template>
