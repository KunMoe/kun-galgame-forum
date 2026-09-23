<script setup lang="ts">
import {
  kunUserTopicNavItem,
  type KUN_USER_PAGE_TOPIC_TYPE
} from '~/constants/user'
import { settle } from '#shared/utils/api/problem'
import type { PageListUserTopicItem } from '#shared/utils/api/schemas'

const TOPIC_RELATION = {
  topic: 'authored',
  topic_like: 'liked',
  topic_upvote: 'upvoted',
  topic_favorite: 'favorited',
  topic_hide: 'hidden'
} as const

const props = defineProps<{
  userId: number
  type: (typeof KUN_USER_PAGE_TOPIC_TYPE)[number]
}>()

const { id: currentUserId } = usePersistUserStore()
const api = useApiClient()
const canViewHiddenTopic = useCan('topic.view_hidden')
const canSeeHidden = computed(
  () => currentUserId === props.userId || canViewHiddenTopic.value
)
const { allowsNsfw: includeNsfw } = useContentStance()

const activeTab = computed(() => props.type)
const relation = computed(() => TOPIC_RELATION[props.type])
const pageData = reactive({
  page: usePageQuery(),
  limit: 50
})

const { data, status } = await useApi<PageListUserTopicItem>(
  () =>
    `user-topics:${props.userId}:${relation.value}:${pageData.page}:${pageData.limit}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (client, { signal }) =>
    client.GET('/users/{user_id}/topics', {
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

const handleUpdateTopicHideStatus = async (topicId: string) => {
  const res = await useComponentMessageStore().alert(
    '八嘎杂鱼笨蛋萝莉, 你要取消隐藏该话题吗, 取消隐藏后该话题将对所有人可见'
  )
  if (!res) {
    return
  }

  const result = await settle(
    api.PATCH('/topics/{topic_id}', {
      params: { path: { topic_id: topicId } },
      body: { state: 'published' }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('取消隐藏话题成功', 'success')
  await navigateTo(`/topic/${topicId}`)
}
</script>

<template>
  <div class="space-y-3">
    <KunTab
      :items="kunUserTopicNavItem(userId, canSeeHidden)"
      :model-value="activeTab"
      variant="light"
      size="sm"
      scrollable
    />

    <div class="flex flex-col space-y-3" v-if="data && data.items.length">
      <template v-if="relation !== 'hidden'">
        <KunCard
          v-for="(topic, index) in data.items"
          :key="index"
          :href="`/topic/${topic.id}`"
        >
          <div>
            {{ topic.title }}
          </div>
          <div class="text-default-500 text-sm">
            <KunTime :time="topic.created_at" type="date" show-year />
          </div>
        </KunCard>
      </template>

      <template v-else>
        <KunCard
          :is-hoverable="false"
          :is-transparent="true"
          v-for="(topic, index) in data.items"
          :key="index"
        >
          <KunLink :to="`/topic/${topic.id}`">
            {{ topic.title }}
          </KunLink>
          <div
            class="text-default-500 flex items-center justify-between text-sm"
          >
            <KunTime :time="topic.created_at" type="date" show-year />
            <KunButton
              @click="handleUpdateTopicHideStatus(topic.id)"
              size="sm"
              variant="flat"
              color="primary"
            >
              取消隐藏
            </KunButton>
          </div>
        </KunCard>
      </template>
    </div>

    <KunPagination
      v-if="data && data.total > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />

    <KunNull
      v-if="data && !data.items.length"
      description="这只笨蛋萝莉没有发布过任何话题"
    />
  </div>
</template>
