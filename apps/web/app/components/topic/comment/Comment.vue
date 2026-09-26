<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import ContentDocument from '~/components/content/Document.vue'
import type { Comment } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { useTopicReplies } from '~/composables/topic/useTopicReplies'
import { contentPlainText } from '~/utils/contentPlainText'
import { toKunUser } from '~/utils/userRef'
import { updateCommentSchema } from '~/validations/topic'
import { threadComments } from './threadComments'

const props = defineProps<{
  replyId: string
  commentsData: Comment[]
}>()

const currentUserId = usePersistUserStore().id
const topicId = inject<number>('topicId', 0)
const api = useApiClient()
const { refreshReply } = useTopicReplies(String(topicId))
const comments = computed(() => props.commentsData)
const activeCommentId = ref<string | null>(null)
const targetUserForPanel = ref<KunUser | null>(null)
const parentCommentIdForPanel = ref<string | null>(null)

const threadedComments = computed(() => threadComments(comments.value))

const editingId = ref<string | null>(null)
const editValue = ref('')
const isSaving = ref(false)

const isMobileQuery = useMediaQuery('(max-width: 767px)')
const mounted = ref(false)
onMounted(() => (mounted.value = true))
const isMobile = computed(() => mounted.value && isMobileQuery.value)
const isCommentPanelOpen = computed({
  get: () => activeCommentId.value !== null && !!targetUserForPanel.value,
  set: (open) => {
    if (!open) {
      activeCommentId.value = null
      targetUserForPanel.value = null
      parentCommentIdForPanel.value = null
    }
  }
})

const handleClickComment = (comment: Comment) => {
  if (!currentUserId) {
    useAuthModal().open()
    return
  }

  if (activeCommentId.value === comment.id) {
    activeCommentId.value = null
    targetUserForPanel.value = null
    parentCommentIdForPanel.value = null
  } else {
    activeCommentId.value = comment.id
    targetUserForPanel.value = toKunUser(comment.author)
    parentCommentIdForPanel.value = comment.id
  }
}

const handleNewComment = () => {
  void refreshReply(props.replyId)
  activeCommentId.value = null
  targetUserForPanel.value = null
  parentCommentIdForPanel.value = null
}

const handleRemoveComment = () => {
  void refreshReply(props.replyId)
}

// The editor writes the stored plain text, not the rendered document: an
// /image/<hash> token resolves to an image node on read, so editing what the
// document renders would drop the token.
const handleStartEdit = async (comment: Comment) => {
  const result = await settle(
    api.GET('/comments/{comment_id}/source', {
      params: { path: { comment_id: comment.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  editValue.value = result.data.text
  editingId.value = comment.id
}

const handleCancelEdit = () => {
  editingId.value = null
  editValue.value = ''
}

const handleSaveEdit = async (comment: Comment) => {
  const body = { text: editValue.value }
  const parsed = updateCommentSchema.safeParse(body)
  if (!parsed.success) {
    const message = JSON.parse(parsed.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }

  isSaving.value = true
  const result = await settle(
    api.PATCH('/comments/{comment_id}', {
      params: { path: { comment_id: comment.id } },
      body
    })
  )
  isSaving.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  await refreshReply(props.replyId)
  editingId.value = null
  useMessage('编辑评论成功', 'success')
}
</script>

<template>
  <div v-if="comments.length" class="bg-default-100 space-y-3 rounded-lg p-3">
    <h3 class="text-lg font-semibold">评论</h3>

    <div class="space-y-3">
      <div
        v-for="{ comment, depth } in threadedComments"
        :id="`comment-${comment.id}`"
        :key="comment.id"
        :class="depth === 1 ? 'ml-9' : ''"
      >
        <div class="flex items-start space-x-3">
          <UserHoverCard :user-id="comment.author.id">
            <KunAvatar :user="toKunUser(comment.author)" />
          </UserHoverCard>

          <div class="flex w-full flex-col space-y-1">
            <div class="text-sm">
              <span>{{ toKunUser(comment.author).name }}</span>
              <span class="text-default-500 mx-1">
                {{ depth === 1 ? '回复' : '评论' }}
              </span>
              <KunLink
                size="sm"
                underline="hover"
                :to="`/user/${comment.in_reply_to_user.id}`"
              >
                {{ toKunUser(comment.in_reply_to_user).name }}
              </KunLink>
            </div>

            <div v-if="editingId === comment.id" class="space-y-2">
              <KunTextarea
                name="edit-comment"
                placeholder="请输入您的评论, 最大字数为 1007"
                :rows="4"
                v-model="editValue"
              />
              <div class="flex justify-end gap-1">
                <KunButton
                  variant="light"
                  color="danger"
                  @click="handleCancelEdit"
                >
                  取消
                </KunButton>
                <KunButton
                  :disabled="isSaving"
                  :loading="isSaving"
                  @click="handleSaveEdit(comment)"
                >
                  保存
                </KunButton>
              </div>
            </div>

            <ContentDocument
              v-else
              compact
              class-name="text-default-700 text-sm"
              :document="comment.content"
            />

            <div class="flex items-center justify-between">
              <span class="text-default-500 text-xs">
                <KunTime :time="comment.created_at" type="datetime" show-year />
                <span v-if="comment.edited_at" class="ml-1">
                  (编辑于
                  <KunTime
                    :time="comment.edited_at"
                    type="datetime"
                    show-year
                  />)
                </span>
              </span>

              <div class="flex items-center gap-1 leading-none">
                <TopicCommentLike :comment="comment" />
                <KunTooltip text="评论">
                  <KunReaction
                    :toggle="false"
                    size="sm"
                    icon="uil:comment-dots"
                    label="评论"
                    @click="handleClickComment(comment)"
                  />
                </KunTooltip>
                <KunPopover position="top-end">
                  <template #trigger>
                    <KunReaction
                      :toggle="false"
                      size="sm"
                      icon="lucide:ellipsis"
                      label="更多"
                    />
                  </template>

                  <div class="flex w-44 flex-col gap-2 p-2">
                    <KunButton
                      v-if="
                        comment.viewer?.can_edit && editingId !== comment.id
                      "
                      variant="light"
                      color="default"
                      size="sm"
                      class-name="w-full justify-start gap-2 whitespace-nowrap"
                      @click="handleStartEdit(comment)"
                    >
                      <KunIcon class-name="text-lg" name="lucide:pencil" />
                      编辑评论
                    </KunButton>
                    <TopicCommentDelete
                      :comment="comment"
                      @remove-comment="handleRemoveComment"
                    />
                    <ReportButton
                      v-if="Number(comment.author.id) !== currentUserId"
                      menu
                      subject-kind="forum_comment"
                      :subject-id="Number(comment.id)"
                      :snapshot="contentPlainText(comment.content)"
                      :subject-url="`${kungal.domain.main}/topic/${topicId}?comment=${comment.id}`"
                    />
                  </div>
                </KunPopover>
              </div>
            </div>
          </div>
        </div>

        <KunFadeCard v-if="!isMobile">
          <LazyTopicCommentPanel
            v-if="activeCommentId === comment.id && targetUserForPanel"
            :reply-id="replyId"
            :target-user="targetUserForPanel"
            :parent-comment-id="parentCommentIdForPanel ?? undefined"
            @get-comment="handleNewComment"
            @close-panel="activeCommentId = null"
          />
        </KunFadeCard>
      </div>
    </div>

    <KunDrawer
      v-if="isMobile"
      v-model="isCommentPanelOpen"
      placement="bottom"
      size="md"
      title="发表评论"
    >
      <LazyTopicCommentPanel
        v-if="targetUserForPanel"
        :reply-id="replyId"
        :target-user="targetUserForPanel"
        :parent-comment-id="parentCommentIdForPanel ?? undefined"
        @get-comment="handleNewComment"
        @close-panel="activeCommentId = null"
      />
    </KunDrawer>
  </div>
</template>
