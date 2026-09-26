<script setup lang="ts">
import { problemMessage } from '#shared/utils/api/message'
import type { PollOption, PollVote } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  pollId: string
  options: PollOption[]
}>()

const open = defineModel<boolean>({ required: true })

const textOf = computed(
  () => new Map(props.options.map((option) => [option.id, option.text]))
)

const {
  items: votes,
  hasMore,
  loadMore,
  loadingMore,
  status,
  problem
} = await useCursorList<PollVote>(
  () => `poll-votes:${props.pollId}`,
  (api, cursor) =>
    api.GET('/polls/{poll_id}/votes', {
      params: {
        path: { poll_id: props.pollId },
        query: { limit: 20, ...(cursor ? { cursor } : {}) }
      }
    })
)
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-md w-[92vw]">
    <div class="space-y-3">
      <h2 class="text-xl font-bold">投票记录</h2>

      <div
        v-if="status === 'pending' && !votes.length"
        class="flex justify-center py-8"
      >
        <KunLoading description="正在加载记录..." />
      </div>

      <KunNull v-else-if="problem" :description="problemMessage(problem)" />

      <KunNull v-else-if="!votes.length" description="还没有人投票" />

      <div v-else class="max-h-[60vh] space-y-4 overflow-y-auto">
        <div
          v-for="vote in votes"
          :key="vote.id"
          class="flex flex-col justify-between gap-2"
        >
          <div class="flex items-center gap-3">
            <KunUserChip :user="toKunUser(vote.voter)" />
            <div class="text-default-500 text-sm">
              <KunTime :time="vote.created_at" type="datetime" show-year />
            </div>
          </div>
          <span class="text-default-700 text-sm">
            投给了「{{ textOf.get(vote.option_id) ?? '已删除的选项' }}」
          </span>
        </div>

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
