<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { FollowedWall, SubscribedTopic } from '#shared/utils/api/schemas'

definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('关注的话题与评论区')

const WALL_PAGE: Record<
  FollowedWall['subject_type'],
  { label: string; path: string }
> = {
  galgame: { label: 'Galgame', path: '/galgame/' },
  galgame_rating: { label: '游戏评分', path: '/galgame-rating/' },
  galgame_resource: { label: '下载资源', path: '/galgame/resource/' },
  galgame_quiz: { label: '游戏答题', path: '/galgame-quiz/' },
  galgame_walkthrough: { label: '游戏攻略', path: '/galgame-walkthrough/' },
  toolset: { label: 'Gal 工具', path: '/toolset/' },
  website: { label: '网站', path: '/website/' }
}

const api = useApiClient()
const workName = useWorkName()
const { allowsNsfw } = useContentStance()

const wallKey = (item: FollowedWall) =>
  `${item.subject_type}:${item.subject_id}`
const wallLink = (item: FollowedWall) =>
  WALL_PAGE[item.subject_type].path +
  (item.website ? item.website.host : item.subject_id)
const wallTitle = (item: FollowedWall) =>
  item.work
    ? workName(item.work)
    : (item.website?.title ?? WALL_PAGE[item.subject_type].label)

type WallItem = FollowedWall & { id: string }
type WallPage = { items: WallItem[]; next_cursor?: string }

const topics = useCursorList<SubscribedTopic>(
  () => `me-topic-subscriptions:${allowsNsfw.value}`,
  (client, cursor, { signal }) =>
    client.GET('/me/topic-subscriptions', {
      params: {
        query: {
          include_nsfw: allowsNsfw.value,
          limit: 30,
          ...(cursor ? { cursor } : {})
        }
      },
      signal
    })
)

const {
  items,
  hasMore,
  loadingMore,
  loadMore,
  pending: wallsPending
} = useCursorList<WallItem, WallPage>(
  'me-walls',
  async (client, cursor, { signal }) => {
    const res = await client.GET('/me/walls', {
      params: { query: { limit: 30, ...(cursor ? { cursor } : {}) } },
      signal
    })
    if (!res.data) {
      return res
    }
    const withIds = res.data.items.map((item) => ({
      ...item,
      id: wallKey(item)
    }))
    return { ...res, data: { ...res.data, items: withIds } }
  }
)

const topicItems = topics.items
const topicsHasMore = topics.hasMore
const topicsLoadingMore = topics.loadingMore
const topicsFailed = computed(() => topics.problem.value !== null)
const pending = computed(() => topics.pending.value || wallsPending.value)
const dropTopic = (id: string) => {
  topicItems.value = topicItems.value.filter((item) => item.id !== id)
}

const unfollow = async (item: FollowedWall) => {
  const result = await settle(
    api.DELETE('/me/walls/{subject_type}/{subject_id}/follow', {
      params: {
        path: { subject_type: item.subject_type, subject_id: item.subject_id }
      }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  items.value = items.value.filter((row) => row.id !== wallKey(item))
}
</script>

<template>
  <div class="flex w-full flex-col space-y-3">
    <header class="flex items-center gap-2">
      <KunButton size="lg" :is-icon-only="true" variant="light" href="/message">
        <KunIcon name="lucide:chevron-left" />
      </KunButton>
      <h2 class="text-lg">关注的话题与评论区</h2>
    </header>

    <KunDivider />

    <template v-if="pending">
      <UserListSkeleton :rows="3" />
      <KunDivider />
      <UserListSkeleton :rows="3" />
    </template>

    <template v-else>
      <MessageFollowTopics
        :items="topicItems"
        :failed="topicsFailed"
        :has-more="topicsHasMore"
        :loading-more="topicsLoadingMore"
        @more="topics.loadMore"
        @unfollowed="dropTopic"
      />

      <KunDivider />

      <h3 class="text-default-600 text-sm font-medium">关注的评论区</h3>

      <div v-if="items.length" class="space-y-2">
        <KunCard
          v-for="item in items"
          :key="wallKey(item)"
          padding="sm"
          :is-hoverable="true"
        >
          <div class="flex w-full items-center gap-2">
            <KunLink
              color="default"
              underline="none"
              :to="wallLink(item)"
              class-name="min-w-0 flex-1 truncate font-medium"
            >
              {{ wallTitle(item) }}
            </KunLink>
            <KunChip size="sm" color="default" class-name="shrink-0">
              {{ WALL_PAGE[item.subject_type].label }}
            </KunChip>
            <KunButton
              size="sm"
              variant="light"
              color="danger"
              @click="unfollow(item)"
            >
              取消关注
            </KunButton>
          </div>
        </KunCard>
      </div>

      <KunNull v-else description="还没有关注任何评论区, 杂鱼~♡" />

      <KunButton
        v-if="hasMore"
        variant="light"
        color="primary"
        full-width
        :loading="loadingMore"
        @click="loadMore"
      >
        <KunIcon name="lucide:chevron-down" />
        加载更多
      </KunButton>
    </template>
  </div>
</template>
