<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import type {
  ListUserFollowee,
  ListUserFollower,
  UserFollowee,
  UserFollower,
  UserProfile,
  UserRef
} from '#shared/utils/api/schemas'
import type { ApiClient } from '#shared/utils/api/client'
import { settle } from '#shared/utils/api/problem'
import { problemMessage } from '#shared/utils/api/message'
import { toKunUser } from '~/utils/userRef'

type FollowItem = UserFollower | UserFollowee
type FollowPage = ListUserFollower | ListUserFollowee

const PAGE_LIMIT = 30

const props = defineProps<{
  user: UserProfile
  type: 'followers' | 'following'
}>()

const api = useApiClient()

const tabItems = computed<KunTabItem[]>(() => {
  const base = `/user/${props.user.id}/follow`
  return [
    { value: 'followers', textValue: '粉丝', href: `${base}/followers` },
    { value: 'following', textValue: '关注', href: `${base}/following` }
  ]
})

const personOf = (item: FollowItem): UserRef =>
  item.object === 'user_follower' ? item.follower : item.followee

const listFollow = (
  client: ApiClient,
  cursor: string | undefined,
  signal?: AbortSignal
): Promise<{ data?: FollowPage; error?: unknown; response: Response }> => {
  const params = {
    path: { user_id: props.user.id },
    query: {
      limit: PAGE_LIMIT,
      ...(cursor ? { cursor } : {})
    }
  }
  return props.type === 'followers'
    ? client.GET('/users/{user_id}/followers', { params, signal })
    : client.GET('/users/{user_id}/following', { params, signal })
}

const items = ref<FollowItem[]>([])
const nextCursor = ref<string | undefined>()
const loadingMore = ref(false)

const { data, problem, status } = await useApi<FollowPage>(
  () => `user-follow:${props.user.id}:${props.type}`,
  (client, { signal }) => listFollow(client, undefined, signal)
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
  const result = await settle(listFollow(api, nextCursor.value))
  loadingMore.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  const seen = new Set(items.value.map((item) => personOf(item).id))
  items.value = [
    ...items.value,
    ...result.data.items.filter((item) => !seen.has(personOf(item).id))
  ]
  nextCursor.value = result.data.next_cursor
}

const emptyDescription = computed(() =>
  props.type === 'followers' ? '还没有粉丝' : '还没有关注任何人'
)
</script>

<template>
  <div class="space-y-3">
    <KunTab :items="tabItems" :model-value="type" variant="light" size="sm" />

    <KunLoading v-if="status === 'pending' && !items.length" />

    <KunNull v-else-if="problem" :description="problemMessage(problem)" />

    <div v-else-if="items.length" class="space-y-2">
      <KunCard
        v-for="item in items"
        :key="personOf(item).id"
        padding="sm"
        :is-hoverable="false"
      >
        <div class="flex items-center gap-2">
          <UserHoverCard :user-id="personOf(item).id">
            <KunUserChip
              size="sm"
              :user="toKunUser(personOf(item))"
              :is-navigation="Boolean(personOf(item).name)"
            />
          </UserHoverCard>
          <span
            v-if="item.followed_at"
            class="text-default-500 ml-auto shrink-0 text-sm"
          >
            <KunTime :time="item.followed_at" />
          </span>
        </div>
      </KunCard>
    </div>

    <KunNull v-else-if="status !== 'pending'" :description="emptyDescription" />

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
