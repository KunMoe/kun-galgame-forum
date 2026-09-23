<script setup lang="ts">
import type { Conversation, NotificationSummary } from '#shared/utils/api/schemas'
import { useCursorList } from '~/composables/useCursorList'
import MessageAsideNoticeItem from '~/components/message/aside/NoticeItem.vue'

const routeName = computed(() => useRoute().name)

const noticeEpoch = useState('message-notice-epoch', () => 0)
const conversationEpoch = useState('message-conversation-epoch', () => 0)

const { data: summary, refresh: refreshSummary } = await useApi<NotificationSummary>(
  'me-notification-summary',
  (api, { signal }) => api.GET('/me/notifications/summary', { signal })
)

watch(noticeEpoch, () => {
  void refreshSummary()
})

const {
  items: conversations,
  hasMore,
  loadingMore,
  loadMore,
  refresh: refreshConversations
} = await useCursorList<Conversation>(
  'me-conversations',
  (api, cursor, { signal }) =>
    api.GET('/me/conversations', {
      params: {
        query: {
          limit: 50,
          ...(cursor ? { cursor } : {})
        }
      },
      signal
    })
)

watch(conversationEpoch, () => {
  void refreshConversations()
})
</script>

<template>
  <aside
    :class="
      cn(
        'scrollbar-hide border-default-200/60 flex w-full shrink-0 flex-col space-y-3 overflow-y-auto pr-0 sm:w-88 sm:border-r sm:pr-3',
        routeName !== 'message' ? 'hidden sm:flex' : ''
      )
    "
  >
    <h2 class="px-2 text-2xl">消息</h2>

    <KunDivider />

    <MessageAsideNoticeItem :summary="summary" />

    <MessageAsideFollowItem />

    <MessageAsideMutedItem :summary="summary" />

    <MessageAsideItem
      v-for="conversation in conversations"
      :key="conversation.id"
      :conversation="conversation"
    />

    <div v-if="hasMore" class="flex justify-center">
      <KunButton
        variant="light"
        size="sm"
        :loading="loadingMore"
        @click="loadMore"
      >
        加载更多
      </KunButton>
    </div>

    <div class="block p-2 sm:hidden">
      <h2 class="text-lg">提示</h2>
      <div>本消息系统尚在开发中, 但是功能应该足够用</div>
      <div>如果您有任何问题, 请查看这个话题</div>
      <KunLink
        to="https://www.kungal.com/topic/1650"
        target="_blank"
        class="text-primary underline"
      >
        [公告] 有关论坛消息系统的说明
      </KunLink>
    </div>
  </aside>
</template>
