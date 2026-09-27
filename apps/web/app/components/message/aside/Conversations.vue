<script setup lang="ts">
import type { Conversation } from '#shared/utils/api/schemas'
import { useCursorList } from '~/composables/useCursorList'

const conversationEpoch = useState('message-conversation-epoch', () => 0)

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
  <div class="space-y-3">
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
  </div>
</template>
