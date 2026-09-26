<script setup lang="ts">
import { useIntersectionObserver, useThrottleFn } from '@vueuse/core'
import type { FollowingActivityGroup } from '#shared/utils/api/schemas'

const { allowsNsfw } = useContentStance()
const { markSeen } = useFollowingUnseen()

const MAX_AUTO_LOADS = 4
const autoLoadCount = ref(0)

const query = computed(() => ({ include_nsfw: allowsNsfw.value, limit: 20 }))

const { items, status, problem, hasMore, loadingMore, loadMore } =
  await useCursorList<FollowingActivityGroup>(
    () => `following-activities:${JSON.stringify(query.value)}`,
    (api, cursor, { signal }) =>
      api.GET('/me/following-activities', {
        params: { query: { ...query.value, ...(cursor ? { cursor } : {}) } },
        signal
      })
  )

watch(
  () => status.value === 'success',
  (loaded) => {
    if (loaded && import.meta.client) {
      markSeen()
    }
  },
  { immediate: true }
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
      v-if="problem && !items.length"
      description="关注动态加载失败，请稍后再试"
    />

    <div v-else-if="status !== 'pending' && !items.length">
      <KunNull description="你关注的人还没有动态" />
      <p class="text-default-500 text-center text-sm">
        在用户主页点击「关注」，TA 在鲲 Galgame 和 NextMoe
        各站的新动态会出现在这里
      </p>
    </div>

    <div v-else class="divide-default-200/60 divide-y">
      <div
        v-for="group in items"
        :key="group.id"
        class="py-5 first:pt-0 last:pb-0"
      >
        <HomeFollowingGroup :group="group" :include-nsfw="allowsNsfw" />
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
