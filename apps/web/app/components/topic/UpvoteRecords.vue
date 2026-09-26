<script setup lang="ts">
import { randomUpvoteDescription } from '~/constants/upvote'
import type { TopicUpvote, UpvoteExcerpt } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  topicId?: string | number
  records?: UpvoteExcerpt[]
}>()

const { lastCreated } = useUpvoteModal()
const extras = ref<TopicUpvote[]>([])

watch(
  lastCreated,
  (upvote) => {
    if (!upvote || !props.topicId) {
      return
    }
    if (upvote.topic_id !== String(props.topicId)) {
      return
    }
    extras.value = [
      upvote,
      ...extras.value.filter((item) => item.id !== upvote.id)
    ]
  },
  { immediate: true }
)

const fetched =
  props.records || !props.topicId
    ? null
    : await useCursorList<TopicUpvote>(
        () => `topic-upvotes:${props.topicId}`,
        (api, cursor) =>
          api.GET('/topics/{topic_id}/upvotes', {
            params: {
              path: { topic_id: String(props.topicId) },
              query: {
                limit: 20,
                ...(cursor ? { cursor } : {})
              }
            }
          })
      )

type Row = {
  id: string
  user: KunUser
  text: string
  created: string | Date
}

const rows = computed<Row[]>(() => {
  if (props.records) {
    return props.records.map((record, index) => ({
      id: `record-${index}`,
      user: toKunUser(record.upvoter),
      text: record.note ?? randomUpvoteDescription(Number(props.topicId ?? 0)),
      created: record.upvoted_at ?? ''
    }))
  }
  const items = [...extras.value, ...(fetched?.items.value ?? [])]
  const seen = new Set<string>()
  const unique: TopicUpvote[] = []
  for (const item of items) {
    if (seen.has(item.id)) {
      continue
    }
    seen.add(item.id)
    unique.push(item)
  }
  return unique.map((item) => ({
    id: item.id,
    user: toKunUser(item.upvoter),
    text: item.note ?? randomUpvoteDescription(Number(item.id)),
    created: item.created_at
  }))
})

const hasMore = computed(() => fetched?.hasMore.value ?? false)
const loadingMore = computed(() => fetched?.loadingMore.value ?? false)
const loadMore = () => fetched?.loadMore() ?? Promise.resolve()
</script>

<template>
  <div
    v-if="rows.length"
    class="max-h-56 space-y-2 overflow-y-auto pr-1 text-sm"
  >
    <div
      v-for="r in rows"
      :key="r.id"
      class="flex items-center justify-between gap-2"
    >
      <div class="flex min-w-0 items-center gap-1.5">
        <UserHoverCard :user-id="r.user.id">
          <KunAvatar :user="r.user" size="sm" />
        </UserHoverCard>
        <span class="text-default-700 shrink-0 font-medium">
          {{ r.user.name }}
        </span>
        <span class="text-default-500 truncate">
          推了这个话题，<span class="text-secondary font-bold">{{
            r.text
          }}</span>
        </span>
      </div>
      <span class="text-default-400 shrink-0 whitespace-nowrap">
        {{ formatTimeDifference(r.created) }}
      </span>
    </div>
    <div v-if="hasMore" class="pt-1 text-center">
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
</template>
