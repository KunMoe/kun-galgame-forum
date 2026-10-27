<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { SubscribedTopic } from '#shared/utils/api/schemas'

defineProps<{
  items: SubscribedTopic[]
  failed: boolean
  hasMore: boolean
  loadingMore: boolean
}>()

const emit = defineEmits<{ more: []; unfollowed: [id: string] }>()

const api = useApiClient()

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
  emit('unfollowed', item.id)
}
</script>

<template>
  <section class="space-y-2">
    <h3 class="text-default-600 text-sm font-medium">关注的话题</h3>

    <KunNull v-if="failed" description="关注的话题加载失败，请稍后再试" />

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
      v-if="hasMore"
      variant="light"
      color="primary"
      full-width
      :loading="loadingMore"
      @click="emit('more')"
    >
      <KunIcon name="lucide:chevron-down" />
      加载更多话题
    </KunButton>
  </section>
</template>
