<script setup lang="ts">
import type { Reply } from '#shared/utils/api/schemas'
import { contentPlainText } from '~/utils/contentPlainText'
import { toKunUser } from '~/utils/userRef'

defineProps<{
  title: string
  reply: Reply
}>()

const emits = defineEmits<{
  handleNewComment: []
}>()

const { id } = usePersistUserStore()
const isCommentPanelVisible = ref(false)
const author = (reply: Reply) => toKunUser(reply.author)

const handleClickComment = () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  isCommentPanelVisible.value = !isCommentPanelVisible.value
}

const handleNewComment = () => {
  emits('handleNewComment')
  isCommentPanelVisible.value = false
}
</script>

<template>
  <div class="w-full">
    <div class="flex items-center justify-between gap-1 leading-none">
      <TopicReactionTrigger />

      <div class="flex items-center gap-1">
        <TopicFooterReply
          :target-user-name="author(reply).name"
          :target-user-id="Number(reply.author.id)"
          :target-floor="reply.floor"
          :target-reply-id="Number(reply.id)"
        />
        <KunTooltip text="评论">
          <KunReaction
            :toggle="false"
            icon="uil:comment-dots"
            label="评论"
            @click="handleClickComment"
          />
        </KunTooltip>
        <KunPopover position="top-end">
          <template #trigger>
            <KunReaction :toggle="false" icon="lucide:ellipsis" label="更多" />
          </template>

          <div class="flex w-54 flex-col gap-2 p-2">
            <KunButton
              variant="light"
              color="default"
              size="sm"
              class-name="w-full justify-start gap-2 whitespace-nowrap"
              @click="
                useKunCopy(
                  `${title}: https://www.kungal.com/topic/${reply.topic_id}?reply=${reply.floor}`
                )
              "
            >
              <KunIcon class-name="text-lg" name="lucide:share-2" />
              分享该回复
            </KunButton>
            <UserFollowMenuItem
              v-if="Number(reply.author.id) !== id"
              :user-id="reply.author.id"
            />
            <template v-if="id">
              <TopicReplyRewrite :reply="reply" />
              <TopicReplyPin :reply="reply" />
              <TopicReplyBestAnswer :reply="reply" />
              <TopicReplyDelete :reply="reply" />
            </template>
            <ReportButton
              v-if="Number(reply.author.id) !== id"
              menu
              subject-kind="forum_reply"
              :subject-id="Number(reply.id)"
              :snapshot="contentPlainText(reply.content)"
              :subject-url="`${kungal.domain.main}/topic/${reply.topic_id}?reply=${reply.floor}`"
            />
          </div>
        </KunPopover>
      </div>
    </div>

    <KunFadeCard>
      <LazyTopicCommentPanel
        v-if="isCommentPanelVisible"
        class="mt-4"
        :reply-id="reply.id"
        :target-user="author(reply)"
        @get-comment="handleNewComment"
        @close-panel="isCommentPanelVisible = false"
      />
    </KunFadeCard>
  </div>
</template>
