<script setup lang="ts">
import { reactionAsset } from '~/constants/reaction'
import { problemMessage } from '#shared/utils/api/message'
import type { Reaction } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{ topicId?: string; replyId?: string }>()

const isReply = computed(() => !!props.replyId)
const open = defineModel<boolean>({ required: true })

const {
  items: records,
  hasMore,
  loadMore,
  loadingMore,
  status,
  problem
} = await useCursorList<Reaction>(
  () =>
    props.replyId
      ? `reply-reactions:${props.replyId}`
      : `topic-reactions:${props.topicId}`,
  (api, cursor) =>
    props.replyId
      ? api.GET('/replies/{reply_id}/reactions', {
          params: {
            path: { reply_id: props.replyId },
            query: { limit: 20, ...(cursor ? { cursor } : {}) }
          }
        })
      : api.GET('/topics/{topic_id}/reactions', {
          params: {
            path: { topic_id: props.topicId! },
            query: { limit: 20, ...(cursor ? { cursor } : {}) }
          }
        })
)
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-md w-[92vw]">
    <div class="space-y-3">
      <h3 class="text-lg font-bold">
        {{ isReply ? '回复回应历史' : '话题回应历史' }}
      </h3>

      <div
        v-if="status === 'pending' && !records.length"
        class="flex justify-center py-8"
      >
        <KunLoading />
      </div>

      <KunNull v-else-if="problem" :description="problemMessage(problem)" />

      <KunNull
        v-else-if="!records.length"
        :description="isReply ? '还没有人回应这条回复' : '还没有人回应这个话题'"
      />

      <div v-else class="max-h-[60vh] space-y-0.5 overflow-y-auto">
        <KunLink
          v-for="item in records"
          :key="item.id"
          :to="`/user/${item.reactor.id}`"
          underline="none"
          color="default"
          class-name="hover:bg-default-100 flex items-center gap-3 rounded-lg p-2 transition-colors"
          @click="open = false"
        >
          <UserHoverCard :user-id="item.reactor.id">
            <KunAvatar :user="toKunUser(item.reactor)" size="sm" />
          </UserHoverCard>
          <span class="text-foreground min-w-0 flex-1 truncate font-medium">
            {{ toKunUser(item.reactor).name }}
          </span>
          <img
            :src="reactionAsset(item.reaction)"
            :alt="item.reaction"
            class="size-6 max-w-none shrink-0"
          />
          <KunTime
            :time="item.created_at"
            class="text-default-500 shrink-0 text-xs"
          />
        </KunLink>
        <div v-if="hasMore" class="py-2 text-center">
          <KunButton
            size="sm"
            variant="flat"
            :loading="loadingMore"
            @click="loadMore"
          >
            加载更多
          </KunButton>
        </div>
      </div>
    </div>
  </KunModal>
</template>
