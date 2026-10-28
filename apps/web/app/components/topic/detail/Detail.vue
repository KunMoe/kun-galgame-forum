<script setup lang="ts">
import { useTopicReplies } from '~/composables/topic/useTopicReplies'
import { useTopicPage } from '~/composables/topic/useTopicPage'
import { useTopicReadMarker } from '~/composables/topic/useTopicReadMarker'
import { useTopicScroll } from '~/composables/topic/useTopicScroll'
import { TOPIC_TOC, useTopicTOC } from '~/composables/topic/useTopicTOC'
import { problemMessage } from '#shared/utils/api/message'
import type { Reply, Topic } from '#shared/utils/api/schemas'
import { toKunUserWithPoints } from '~/utils/userRef'

const props = defineProps<{
  topic: Topic
}>()

const { id } = usePersistUserStore()
const canCreateAnyPoll = useCan('poll.create_any')
const canEditAnyPoll = useCan('poll.edit_any')
const tempReplyStore = useTempReplyStore()
const { lastSuccessfulReply } = storeToRefs(tempReplyStore)
const authorId = computed(() => Number(props.topic.author.id))
const isTopicAdmin = computed(
  () => authorId.value === id || canCreateAnyPoll.value || canEditAnyPoll.value
)

const {
  replies,
  status,
  isComplete,
  hasEarlier,
  sortOrder,
  problem,
  loadMore,
  loadEarlier,
  setSort,
  addNewReply,
  updateReply,
  removeReply,
  retry
} = useTopicReplies(props.topic.id)

const route = useRoute()
const { scrollToFloor, scrollToComment } = useTopicScroll()
useTopicReadMarker(props.topic.id)
const api = useApiClient()

const targetFloor = Number(route.query.reply) || 0
const targetCommentId = Number(route.query.comment) || 0
const { replyTarget } = useTopicPage()

const activeFloor = ref(replyTarget.value)
watch(
  replyTarget,
  (floor) => {
    activeFloor.value = floor
  },
  { flush: 'sync' }
)
watch(
  () => route.query.reply,
  (value) => {
    const floor = Number(value) || 0
    if (floor > 0) {
      activeFloor.value = floor
      nextTick(() => scrollToFloor(floor, false))
    }
  }
)

const sendViewBeacon = () => {
  void api
    .POST('/topics/{topic_id}/views', {
      params: { path: { topic_id: props.topic.id } }
    })
    .catch(() => undefined)
}

let scrollTimer: ReturnType<typeof setTimeout> | undefined
const scrollToTarget = () => {
  scrollTimer = setTimeout(() => {
    const ok =
      targetCommentId > 0
        ? scrollToComment(targetCommentId, false)
        : scrollToFloor(targetFloor, false)
    if (!ok) {
      useMessage('目标回复或评论可能已被删除', 'info')
    }
  }, 300)
}
onMounted(() => {
  sendViewBeacon()
  if (!targetFloor && !targetCommentId) {
    return
  }
  if (status.value === 'pending') {
    watch(status, scrollToTarget, { once: true })
    return
  }
  scrollToTarget()
})
onBeforeUnmount(() => clearTimeout(scrollTimer))

const isLoadingEarlier = ref(false)
const loadEarlierInPlace = async () => {
  const anchor = document.querySelector<HTMLElement>('[data-reply-list] > *')
  const anchorTop = anchor?.getBoundingClientRect().top
  isLoadingEarlier.value = true
  await loadEarlier()
  isLoadingEarlier.value = false
  await nextTick()
  if (!anchor?.isConnected || anchorTop === undefined) {
    return
  }
  window.scrollBy({
    top: anchor.getBoundingClientRect().top - anchorTop,
    behavior: 'instant'
  })
}

provide('topicUserId', authorId.value)
provide('activeReplyFloor', activeFloor)
provide(
  'pageTopic',
  computed(() => props.topic)
)

const author = computed(() =>
  toKunUserWithPoints(props.topic.author, props.topic.author_moemoepoint)
)

const featuredReplies = computed(() => {
  const items: Reply[] = []
  const seen = new Set<string>()
  for (const reply of [props.topic.pinned_reply, props.topic.best_answer]) {
    if (reply && !seen.has(reply.id)) {
      seen.add(reply.id)
      items.push(reply)
    }
  }
  return items
})

const featuredIds = computed(
  () => new Set(featuredReplies.value.map((reply) => reply.id))
)

const listReplies = computed(() =>
  replies.value.filter((reply) => !featuredIds.value.has(reply.id))
)

const tocReplies = computed(() => {
  const map = new Map<string, Reply>()
  for (const reply of featuredReplies.value) {
    map.set(reply.id, reply)
  }
  for (const reply of replies.value) {
    map.set(reply.id, reply)
  }
  return [...map.values()]
})

provide(
  TOPIC_TOC,
  useTopicTOC({
    getContent: () => props.topic.content,
    getReplies: () => tocReplies.value,
    getTargetFloor: () => activeFloor.value
  })
)

watch(
  () => [props.topic.pinned_reply?.id, props.topic.best_answer?.id] as const,
  ([pinnedId, bestId]) => {
    for (const reply of replies.value) {
      reply.is_pinned = pinnedId === reply.id
      reply.is_best_answer = bestId === reply.id
    }
  }
)

watch(
  lastSuccessfulReply,
  (event) => {
    if (!event) {
      return
    }

    switch (event.type) {
      case 'created':
        addNewReply(event.data)
        nextTick(() => {
          scrollToFloor(event.data.floor)
        })
        break
      case 'updated':
        if (replies.value.some((reply) => reply.id === event.data.id)) {
          updateReply(event.data)
        } else {
          addNewReply(event.data)
        }
        nextTick(() => {
          scrollToFloor(event.data.floor)
        })
        break
      case 'deleted':
        removeReply(event.data.id)
        break
    }

    tempReplyStore.clearSuccessfulReply()
  },
  { deep: true }
)
</script>

<template>
  <div class="flex flex-col gap-4 lg:flex-row lg:items-start xl:gap-6">
    <template v-if="topic.author">
      <TopicDetailMasterUser :user="author" class="xl:hidden" />
      <TopicDetailAuthorRail :user="author" :topic-id="topic.id" />
    </template>

    <div class="min-w-0 flex-1 space-y-4">
      <TopicDetailHiddenNotice
        v-if="topic.state === 'hidden'"
        :hidden-by="topic.hidden_by ?? ''"
      />

      <TopicDetailScopeNotice
        v-if="topic.access_scope && topic.access_scope !== 'public'"
        :scope="topic.access_scope"
      />

      <TopicDetailMaster :topic="topic" />

      <TopicMiniappContainer
        :topic-id="Number(topic.id)"
        :is-topic-admin="isTopicAdmin"
      />

      <div id="comments-anchor" class="scroll-mt-20">
        <TopicDetailTool
          :reply-count="topic.reply_count"
          :status="status"
          :sort-order="sortOrder"
          @set-sort-order="setSort"
        />
      </div>

      <section id="reply-section" class="space-y-4">
        <div v-if="hasEarlier" class="text-center">
          <KunButton
            size="lg"
            variant="flat"
            :loading="isLoadingEarlier"
            :disabled="status === 'pending'"
            @click="loadEarlierInPlace"
          >
            加载更早的回复
          </KunButton>
        </div>

        <TopicReply
          v-for="reply in featuredReplies"
          :key="`featured-${reply.id}`"
          :reply="reply"
          :title="topic.title"
        />

        <template v-if="problem">
          <KunNull :description="problemMessage(problem)" />
          <div class="flex justify-center">
            <KunButton variant="flat" size="sm" @click="retry">重试</KunButton>
          </div>
        </template>

        <div
          v-else-if="status === 'pending' && replies.length === 0"
          class="space-y-4"
        >
          <ActivityCardSkeleton v-for="n in 3" :key="n" />
        </div>

        <TopicReplyList
          v-else-if="listReplies.length > 0"
          :initial-replies="listReplies"
          :topic-id="Number(topic.id)"
          :title="topic.title"
        />

        <div class="py-6 text-center">
          <KunButton
            v-if="!isComplete && status !== 'pending'"
            size="lg"
            variant="flat"
            @click="loadMore"
          >
            加载更多
          </KunButton>
          <KunLoading v-if="status === 'pending' && replies.length > 0" />
          <p v-if="isComplete" class="text-default-500">
            {{ `(｡>︿<｡) 已经一滴回复都不剩了哦~` }}
          </p>
        </div>
      </section>

      <TopicDetailActionBar :topic="topic" />
    </div>

    <TopicDetailTocRail />
  </div>
</template>
