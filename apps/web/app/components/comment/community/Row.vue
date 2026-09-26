<script setup lang="ts">
import type { WallComment } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import ContentDocument from '~/components/content/Document.vue'

const props = withDefaults(
  defineProps<{
    comment: WallComment
    target: CommunityCommentTarget
    depth?: number
    replies?: WallComment[]
  }>(),
  { depth: 0, replies: () => [] }
)

const emit = defineEmits<{
  replyAdded: [reply: WallComment]
  updated: [post: WallComment]
  tombstoned: [postId: string]
}>()

const surface = communityCommentSurface(props.target)

const { id } = usePersistUserStore()
const api = useApiClient()
const { open: openFlag } = useGalgameCommentFlag()

const isShowReply = ref(false)
const isEditing = ref(false)
const editingContent = ref('')
const originalContent = ref('')
const isSavingEdit = ref(false)

const isDeleted = computed(() => props.comment.state === 'deleted')
const author = computed(() => toKunUser(props.comment.author))

const isShowEdit = computed(() => props.comment.viewer?.can_edit ?? false)
const isShowDelete = computed(() => props.comment.viewer?.can_delete ?? false)
const isShowFlag = computed(() =>
  props.comment.viewer ? props.comment.viewer.can_flag : !isDeleted.value
)

const isShowMenu = computed(
  () => isShowEdit.value || isShowFlag.value || isShowDelete.value
)

const replyTarget = computed(() =>
  surface.showsReplyTarget && props.comment.addressee
    ? toKunUser(props.comment.addressee)
    : null
)

const editedLabel = computed(() => {
  if (props.comment.edited_at == null && !props.comment.is_edited_by_moderator) {
    return null
  }
  return props.comment.is_edited_by_moderator ? '已编辑（管理）' : '已编辑'
})

const handleFlag = () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  openFlag(props.comment.id)
}

// The editor gets the stored Markdown, not the rendered document: an
// /image/<hash> token becomes an image node on read, so editing what the
// document shows would drop the token.
const handleStartEdit = async () => {
  const result = await settle(
    api.GET('/wall-comments/{wall_comment_id}/source', {
      params: { path: { wall_comment_id: props.comment.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  editingContent.value = result.data.content_markdown
  originalContent.value = result.data.content_markdown
  isEditing.value = true
}

const handleCancelEdit = () => {
  isEditing.value = false
  editingContent.value = ''
}

const handleSubmitEdit = async () => {
  const text = editingContent.value.trim()
  if (!text) {
    useMessage(10540, 'warn')
    return
  }
  if ([...text].length > surface.maxLength) {
    useMessage(`评论最大长度为 ${surface.maxLength} 个字符`, 'warn')
    return
  }
  if (text === originalContent.value) {
    handleCancelEdit()
    return
  }

  isSavingEdit.value = true
  const result = await settle(
    api.PATCH('/wall-comments/{wall_comment_id}', {
      params: { path: { wall_comment_id: props.comment.id } },
      body: { content_markdown: text }
    })
  )
  isSavingEdit.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('评论已更新', 'success')
  emit('updated', result.data)
  handleCancelEdit()
}

const handleDelete = async () => {
  const ok = await useComponentMessageStore().alert(
    '删除后此楼保留占位，回复不受影响，确定删除吗？'
  )
  if (!ok) {
    return
  }

  const result = await settle(
    api.DELETE('/wall-comments/{wall_comment_id}', {
      params: { path: { wall_comment_id: props.comment.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(10538, 'success')
  emit('tombstoned', props.comment.id)
}

const handleReplyAdded = (reply: WallComment) => {
  isShowReply.value = false
  emit('replyAdded', reply)
}
</script>

<template>
  <div :id="`${surface.anchorPrefix}-${comment.id}`" class="flex gap-3">
    <UserHoverCard :user-id="author.id">
      <KunAvatar :user="author" :size="depth === 0 ? 'md' : 'sm'" />
    </UserHoverCard>

    <div class="min-w-0 flex-1">
      <div
        class="flex flex-wrap items-baseline gap-x-2 gap-y-1 text-xs leading-5"
      >
        <span class="text-default-800 text-sm font-medium">
          {{ author.name }}
        </span>
        <template v-if="replyTarget">
          <KunIcon name="lucide:arrow-right" class="text-default-400" />
          <KunLink underline="hover" size="sm" :to="`/user/${replyTarget.id}`">
            {{ replyTarget.name }}
          </KunLink>
        </template>
        <span class="text-default-400">
          <KunTime :time="comment.created_at" />
        </span>
        <span v-if="editedLabel" class="text-default-400 italic">
          {{ editedLabel }}
        </span>
        <span
          v-if="comment.state === 'held'"
          class="bg-warning-100 text-warning-600 rounded-full px-2 py-0.5"
        >
          审核中
        </span>
      </div>

      <p v-if="isDeleted" class="text-default-400 mt-2 text-sm italic">
        [已删除]
      </p>

      <ContentDocument
        v-else-if="!isEditing"
        class-name="mt-2"
        compact
        :document="comment.content"
      />

      <div v-else class="mt-2 space-y-2">
        <KunMilkdownDualEditorProvider
          :value-markdown="editingContent"
          @set-markdown="(val) => (editingContent = val)"
        />
        <div class="flex justify-end gap-2">
          <KunButton
            variant="light"
            color="default"
            size="sm"
            @click="handleCancelEdit"
          >
            取消
          </KunButton>
          <KunButton
            size="sm"
            :loading="isSavingEdit"
            @click="handleSubmitEdit"
          >
            保存
          </KunButton>
        </div>
      </div>

      <div
        v-if="!isDeleted && !isEditing"
        class="mt-2.5 flex items-center gap-1"
      >
        <KunTooltip text="回复">
          <KunReaction
            :toggle="false"
            icon="lucide:reply"
            label="回复"
            @click="isShowReply = !isShowReply"
          />
        </KunTooltip>

        <CommentCommunityLike :comment="comment" />

        <KunPopover v-if="isShowMenu" position="bottom-start">
          <template #trigger>
            <KunReaction :toggle="false" icon="lucide:ellipsis" label="更多" />
          </template>

          <div class="flex w-44 flex-col gap-2 p-2">
            <KunButton
              v-if="isShowEdit"
              variant="light"
              color="default"
              size="sm"
              class-name="w-full justify-start gap-2 whitespace-nowrap"
              @click="handleStartEdit"
            >
              <KunIcon class-name="text-lg" name="lucide:pencil" />
              编辑评论
            </KunButton>

            <KunButton
              v-if="isShowFlag"
              variant="light"
              color="danger"
              size="sm"
              class-name="w-full justify-start gap-2 whitespace-nowrap"
              @click="handleFlag"
            >
              <KunIcon class-name="text-lg" name="lucide:flag" />
              举报评论
            </KunButton>

            <KunButton
              v-if="isShowDelete"
              variant="light"
              color="danger"
              size="sm"
              class-name="w-full justify-start gap-2 whitespace-nowrap"
              @click="handleDelete"
            >
              <KunIcon class-name="text-lg" name="lucide:trash-2" />
              删除评论
            </KunButton>
          </div>
        </KunPopover>
      </div>

      <KunFadeCard>
        <CommentCommunityComposer
          v-if="isShowReply"
          class="mt-3"
          :target="target"
          :parent-comment-id="comment.id"
          :is-reply="true"
          @close="isShowReply = false"
          @submitted="handleReplyAdded"
        />
      </KunFadeCard>

      <div
        v-if="depth === 0 && replies.length"
        class="mt-4 space-y-4"
      >
        <CommentCommunityRow
          v-for="reply in replies"
          :key="reply.id"
          :comment="reply"
          :target="target"
          :depth="1"
          @reply-added="(r) => emit('replyAdded', r)"
          @updated="(u) => emit('updated', u)"
          @tombstoned="(pid) => emit('tombstoned', pid)"
        />
      </div>
    </div>
  </div>
</template>
