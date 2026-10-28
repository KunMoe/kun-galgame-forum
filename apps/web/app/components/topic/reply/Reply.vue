<script setup lang="ts">
import ContentDocument from '~/components/content/Document.vue'
import { scrollPage } from '../_helper'
import { useQuoteContent } from '~/composables/topic/useQuoteContent'
import { useTopicReplies } from '~/composables/topic/useTopicReplies'
import type { ComputedRef } from 'vue'
import type {
  Reply,
  ReplyEngagement,
  Topic,
  TopicEngagement
} from '#shared/utils/api/schemas'
import { contentPlainText } from '~/utils/contentPlainText'
import { toKunReactions } from '~/utils/reactionSummary'
import { toKunUserWithPoints } from '~/utils/userRef'

const bannerBaseClasses =
  'flex items-center gap-2 px-4 py-2 mb-3 rounded-lg font-semibold text-sm'

const props = defineProps<{
  reply: Reply
  title: string
}>()

const { scrollToReplyId } = storeToRefs(useTempReplyStore())
const { refreshReply, updateReply } = useTopicReplies(props.reply.topic_id)
const pageTopic = inject<ComputedRef<Topic>>('pageTopic')
const replaceTopic = inject<(topic: Topic) => void>('replaceTopic', () => {})

const activeFloor = inject('activeReplyFloor', ref(0))
const isActive = computed(
  () => activeFloor.value > 0 && activeFloor.value === props.reply.floor
)

const applyEngagement = (engagement: TopicEngagement | ReplyEngagement) => {
  if (engagement.object !== 'reply_engagement') {
    return
  }
  const next: Reply = {
    ...props.reply,
    like_count: engagement.like_count,
    dislike_count: engagement.dislike_count,
    viewer: engagement.viewer,
    reactions: engagement.reactions
  }
  updateReply(next)
  const topic = pageTopic?.value
  if (
    topic &&
    (topic.pinned_reply?.id === next.id || topic.best_answer?.id === next.id)
  ) {
    replaceTopic({
      ...topic,
      pinned_reply:
        topic.pinned_reply?.id === next.id ? next : topic.pinned_reply,
      best_answer: topic.best_answer?.id === next.id ? next : topic.best_answer
    })
  }
}

provide(
  reactionsKey,
  useReactions({
    replyId: props.reply.id,
    targetUserId: Number(props.reply.author.id),
    reactions: toKunReactions(props.reply.reactions),
    sync: () => toKunReactions(props.reply.reactions),
    showReactors: true,
    onEngagement: applyEngagement
  })
)

const contentRef = ref<HTMLElement | null>(null)
const { preview, keepPreview, hidePreview } = useQuoteContent(contentRef)

const replyContent = computed(() =>
  truncateRunes(contentPlainText(props.reply.content), 20)
)

const MOBILE_FLAT =
  'max-sm:rounded-none max-sm:border-x-0 max-sm:border-t-0 max-sm:border-b max-sm:bg-transparent max-sm:px-0 max-sm:pt-0 max-sm:pb-4 max-sm:shadow-none max-sm:backdrop-blur-none'

const cardClasses = computed(() => {
  if (props.reply.is_best_answer) {
    return 'border-l-4 border-success-600 dark:border-success-700 max-sm:border-l-4 max-sm:pl-3'
  }
  if (props.reply.is_pinned) {
    return 'border-l-4 border-secondary-500 dark:border-secondary-600 max-sm:border-l-4 max-sm:pl-3'
  }
  return ''
})

watch(
  () => scrollToReplyId.value,
  async () => {
    if (scrollToReplyId.value !== -1) {
      await nextTick()
      scrollPage(scrollToReplyId.value)
      scrollToReplyId.value = -1
    }
  }
)

const handleNewComment = () => {
  void refreshReply(props.reply.id)
}
</script>

<template>
  <div
    :class="
      cn(
        'outline-primary kun-reply flex justify-between gap-3 outline-offset-2',
        isActive && 'rounded-lg outline-2'
      )
    "
    :id="`${reply.floor}.${replyContent}`"
  >
    <KunCard
      :is-transparent="false"
      :is-hoverable="false"
      :class-name="
        cn('w-full min-w-0 relative overflow-visible', MOBILE_FLAT, cardClasses)
      "
      content-class="gap-3"
    >
      <div
        v-if="reply.is_best_answer"
        :class="
          cn(
            'bg-success-500/20 text-success-700 dark:text-success-300',
            bannerBaseClasses
          )
        "
      >
        <KunIcon class-name="text-xl" name="lucide:bookmark-check" />
        <span>最佳答案</span>
        <KunIcon
          class-name="absolute bottom-3 right-3 text-[10rem] text-success-500/20 select-none -z-1"
          name="lucide:circle-check-big"
        />
      </div>

      <div
        v-else-if="reply.is_pinned"
        :class="
          cn(
            'bg-secondary-500/20 text-secondary-700 dark:text-secondary-300',
            bannerBaseClasses
          )
        "
      >
        <KunIcon class-name="text-xl" name="lucide:pin" />
        <span>置顶回复</span>
        <KunIcon
          class-name="absolute bottom-3 right-3 text-[10rem] text-secondary-500/20 select-none -z-1"
          name="lucide:disc-2"
        />
      </div>

      <TopicDetailUser
        :user="toKunUserWithPoints(reply.author, reply.author_moemoepoint)"
        :created="reply.created_at"
        :edited="reply.edited_at"
        :topic-id="Number(reply.topic_id)"
        :floor="reply.floor"
      />

      <div ref="contentRef">
        <ContentDocument
          v-if="reply.content.children.length"
          compact
          :document="reply.content"
        />
      </div>

      <TopicQuotePreview
        :preview="preview"
        @keep="keepPreview"
        @leave="hidePreview"
      />

      <TopicReactionBar class="mt-2" />

      <TopicReplyFooter
        :reply="reply"
        :title="title"
        @handle-new-comment="handleNewComment"
      />

      <TopicComment :reply-id="reply.id" :comments-data="reply.comments" />
    </KunCard>
  </div>
</template>
