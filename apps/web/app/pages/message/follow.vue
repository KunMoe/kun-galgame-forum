<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { FollowedWall } from '#shared/utils/api/schemas'

definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('关注的评论区')

const WALL_PAGE: Record<
  FollowedWall['subject_type'],
  { label: string; path: string }
> = {
  galgame: { label: 'Galgame', path: '/galgame/' },
  galgame_rating: { label: '游戏评分', path: '/galgame-rating/' },
  galgame_resource: { label: '下载资源', path: '/galgame/resource/' },
  galgame_quiz: { label: '游戏答题', path: '/galgame-quiz/' },
  toolset: { label: 'Gal 工具', path: '/toolset/' },
  website: { label: '网站', path: '/website/' }
}

const api = useApiClient()
const workName = useWorkName()

const wallKey = (item: FollowedWall) =>
  `${item.subject_type}:${item.subject_id}`
const wallLink = (item: FollowedWall) =>
  WALL_PAGE[item.subject_type].path +
  (item.website ? item.website.host : item.subject_id)
const wallTitle = (item: FollowedWall) =>
  item.work
    ? workName(item.work)
    : (item.website?.title ?? WALL_PAGE[item.subject_type].label)

const items = ref<FollowedWall[]>([])
const nextCursor = ref<string | undefined>()
const loadingMore = ref(false)

const { data, status } = await useApi('me-walls', (client, { signal }) =>
  client.GET('/me/walls', { params: { query: { limit: 30 } }, signal })
)

watchEffect(() => {
  if (data.value) {
    items.value = [...data.value.items]
    nextCursor.value = data.value.next_cursor
  }
})

const loadMore = async () => {
  if (!nextCursor.value || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const result = await settle(
    api.GET('/me/walls', {
      params: { query: { cursor: nextCursor.value, limit: 30 } }
    })
  )
  loadingMore.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  const seen = new Set(items.value.map(wallKey))
  items.value = [
    ...items.value,
    ...result.data.items.filter((item) => !seen.has(wallKey(item)))
  ]
  nextCursor.value = result.data.next_cursor
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
  items.value = items.value.filter((row) => wallKey(row) !== wallKey(item))
}
</script>

<template>
  <div class="flex w-full flex-col space-y-3">
    <header class="flex items-center gap-2">
      <KunButton size="lg" :is-icon-only="true" variant="light" href="/message">
        <KunIcon name="lucide:chevron-left" />
      </KunButton>
      <h2 class="text-lg">关注的评论区</h2>
    </header>

    <KunDivider />

    <KunLoading v-if="status === 'pending' && !items.length" />

    <div v-else-if="items.length" class="space-y-2">
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
      v-if="nextCursor"
      variant="light"
      color="primary"
      full-width
      :loading="loadingMore"
      @click="loadMore"
    >
      <KunIcon name="lucide:chevron-down" />
      加载更多
    </KunButton>
  </div>
</template>
