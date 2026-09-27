<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { SubscribedTopic } from '#shared/utils/api/schemas'

const PAGE_SIZE = 30

const api = useApiClient()
const { allowsNsfw } = useContentStance()

const items = ref<SubscribedTopic[]>([])
const nextCursor = ref<string | undefined>()
const status = ref<'pending' | 'success' | 'error'>('pending')
const loadingMore = ref(false)

const fetchPage = (cursor?: string) =>
  settle(
    api.GET('/me/topic-subscriptions', {
      params: {
        query: {
          include_nsfw: allowsNsfw.value,
          limit: PAGE_SIZE,
          ...(cursor ? { cursor } : {})
        }
      }
    })
  )

const loadFirst = async () => {
  status.value = 'pending'
  const result = await fetchPage()
  if (!result.ok) {
    status.value = 'error'
    return
  }
  items.value = result.data.items
  nextCursor.value = result.data.next_cursor
  status.value = 'success'
}

const loadMore = async () => {
  if (!nextCursor.value || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const result = await fetchPage(nextCursor.value)
  loadingMore.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  const seen = new Set(items.value.map((item) => item.id))
  items.value = [
    ...items.value,
    ...result.data.items.filter((item) => !seen.has(item.id))
  ]
  nextCursor.value = result.data.next_cursor
}

const unfollow = async (item: SubscribedTopic) => {
  const result = await settle(
    api.PUT('/topics/{topic_id}/subscription', {
      params: { path: { topic_id: item.id } },
      body: { notification_level: 'normal' }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  items.value = items.value.filter((row) => row.id !== item.id)
}

onMounted(() => {
  void loadFirst()
})
watch(allowsNsfw, () => {
  void loadFirst()
})
</script>

<template>
  <section class="space-y-2">
    <h3 class="text-default-600 text-sm font-medium">关注的话题</h3>

    <KunLoading v-if="status === 'pending'" />

    <KunNull
      v-else-if="status === 'error'"
      description="关注的话题加载失败，请稍后再试"
    />

    <div v-else-if="items.length" class="space-y-2">
      <KunCard
        v-for="item in items"
        :key="item.id"
        padding="sm"
        :is-hoverable="true"
      >
        <div class="flex w-full items-center gap-2">
          <KunLink
            color="default"
            underline="none"
            :to="`/topic/${item.id}`"
            class-name="min-w-0 flex-1 truncate font-medium"
          >
            {{ item.topic?.title ?? '话题' }}
          </KunLink>
          <KunChip
            v-if="item.unread_reply_count"
            size="sm"
            color="primary"
            variant="flat"
            class-name="shrink-0"
          >
            {{ item.unread_reply_count }} 条新回复
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

    <p v-else class="text-default-500 text-sm">
      还没有关注任何话题。在话题底部的铃铛里选择「关注」，有新回复时会通知你。
    </p>

    <KunButton
      v-if="nextCursor"
      variant="light"
      color="primary"
      full-width
      :loading="loadingMore"
      @click="loadMore"
    >
      <KunIcon name="lucide:chevron-down" />
      加载更多话题
    </KunButton>
  </section>
</template>
