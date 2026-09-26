<script setup lang="ts">
import { useIntersectionObserver, useThrottleFn } from '@vueuse/core'
import type { Activity, FollowingActivity } from '#shared/utils/api/schemas'
import { feedTabQuery } from '~/utils/activity'

const props = defineProps<{
  tabId: string
  types: string
  following?: boolean
}>()

const settings = usePersistSettingsStore()
const { allowsNsfw } = useContentStance()
const { markSeen } = useFollowingUnseen()

const MAX_AUTO_LOADS = 4
const autoLoadCount = ref(0)

const query = computed(() => ({
  ...feedTabQuery(props.types),
  include_nsfw: props.tabId === 'all' ? false : allowsNsfw.value,
  include_galgames_without_resources: settings.showKUNGalgameNoResource,
  limit: 30
}))

const followingQuery = computed(() => {
  const { sort: _sort, ...rest } = query.value
  return rest
})

const unwrapFollowing = <T extends { items: FollowingActivity[] }>(
  data: T | undefined
) =>
  data && {
    ...data,
    items: data.items.flatMap((entry) =>
      entry.activity ? [entry.activity] : []
    )
  }

const { items, status, problem, hasMore, loadingMore, loadMore } =
  await useCursorList<Activity>(
    () =>
      `${props.following ? 'following-activities' : 'activities'}:${JSON.stringify(query.value)}`,
    (api, cursor, { signal }) =>
      props.following
        ? api
            .GET('/me/following-activities', {
              params: {
                query: {
                  ...followingQuery.value,
                  ...(cursor ? { cursor } : {})
                }
              },
              signal
            })
            .then((r) => ({ ...r, data: unwrapFollowing(r.data) }))
        : api.GET('/activities', {
            params: {
              query: { ...query.value, ...(cursor ? { cursor } : {}) }
            },
            signal
          })
  )

watch(
  () => (props.following && status.value === 'success' ? items.value[0] : null),
  (newest) => {
    if (newest && import.meta.client) {
      markSeen(newest.occurred_at)
    }
  },
  { immediate: true }
)

watch(
  () => props.tabId,
  () => {
    autoLoadCount.value = 0
  }
)

const loadNext = (auto: boolean) => {
  if (auto) {
    if (autoLoadCount.value >= MAX_AUTO_LOADS) {
      return
    }
    autoLoadCount.value++
  } else {
    autoLoadCount.value = 0
  }
  return loadMore()
}

const autoLoad = useThrottleFn(() => loadNext(true), 600)

const sentinel = ref<HTMLElement | null>(null)
useIntersectionObserver(
  sentinel,
  ([entry]) => {
    if (entry?.isIntersecting && hasMore.value && !loadingMore.value) {
      autoLoad()
    }
  },
  { rootMargin: '150px' }
)
</script>

<template>
  <KunLoadingDim
    class="min-w-0"
    :loading="status === 'pending' && items.length > 0"
  >
    <KunNull
      v-if="following && problem && !items.length"
      description="关注动态加载失败，请稍后再试"
    />

    <div v-else-if="status !== 'pending' && !items.length">
      <KunNull :description="following ? '你关注的人还没有动态' : '暂无动态'" />
      <p v-if="following" class="text-default-500 text-center text-sm">
        在用户主页点击「关注」，TA 的新话题、资源和评论会出现在这里
      </p>
    </div>

    <div v-else class="divide-default-200/60 divide-y">
      <div
        v-for="activity in items"
        :key="activity.id"
        class="py-5 first:pt-0 last:pb-0"
      >
        <ActivityCard :activity="activity" />
      </div>
      <template v-if="loadingMore">
        <div v-for="n in 3" :key="`skeleton-${n}`" class="py-5">
          <ActivityCardSkeleton />
        </div>
      </template>
    </div>

    <div v-if="items.length" ref="sentinel" class="flex justify-center pt-4">
      <KunButton
        v-if="hasMore && !loadingMore"
        variant="light"
        @click="loadNext(false)"
      >
        加载更多
      </KunButton>
      <span v-else-if="!hasMore" class="text-default-400 text-sm">
        没有更多动态了
      </span>
    </div>
  </KunLoadingDim>
</template>
