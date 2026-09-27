<script setup lang="ts">
import type { Topic } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  topic: Topic
}>()

const { id } = usePersistUserStore()
const { isEdit } = storeToRefs(useTempReplyStore())

const handleOpenReply = () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  isEdit.value = true
}

const scrollToComments = () => {
  document
    .getElementById('comments-anchor')
    ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const handleShare = () => {
  useKunCopy(
    `${props.topic.title}: https://www.kungal.com/topic/${props.topic.id}`
  )
}
</script>

<template>
  <div
    class="border-default/20 bg-background/70 sticky bottom-4 z-30 mx-2 flex items-center gap-1 rounded-full border px-2 py-1.5 shadow-lg backdrop-blur-xl md:hidden"
  >
    <button
      class="bg-default-100 text-default-500 hover:bg-default-200 min-w-0 flex-1 truncate rounded-full px-4 py-2 text-left text-sm transition-colors"
      @click="handleOpenReply"
    >
      发布一条可爱的回复吧～
    </button>

    <TopicFooterFavorite :topic="topic" />

    <TopicFooterSubscribe :topic-id="topic.id" />

    <KunTooltip text="跳到评论区">
      <KunReaction
        :toggle="false"
        icon="uil:comment-dots"
        label="跳到评论区"
        @click="scrollToComments"
      />
    </KunTooltip>

    <KunPopover position="top-end" inner-class="p-2 w-56">
      <template #trigger>
        <KunReaction :toggle="false" icon="lucide:ellipsis" label="更多" />
      </template>

      <div class="flex flex-col gap-1">
        <TopicFooterUpvote menu :topic="topic" />
        <KunButton
          variant="light"
          color="default"
          size="sm"
          class-name="w-full justify-start gap-2 whitespace-nowrap"
          @click="handleShare"
        >
          <KunIcon class-name="text-lg" name="lucide:share-2" />
          分享
        </KunButton>
        <TopicFooterRewrite menu :topic="topic" />
        <TopicFooterHide :topic="topic" />
        <ReportButton
          v-if="Number(topic.author.id) !== id"
          menu
          subject-kind="forum_topic"
          :subject-id="Number(topic.id)"
          :snapshot="topic.title"
          :subject-url="`${kungal.domain.main}/topic/${topic.id}`"
        />
      </div>
    </KunPopover>
  </div>
</template>
