<script setup lang="ts">
import type { Comment } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { createCommentSchema } from '~/validations/topic'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'

const props = defineProps<{
  replyId: string
  targetUser: KunUser
  parentCommentId?: string
}>()

const emits = defineEmits<{
  getComment: [newComment: Comment]
  closePanel: []
}>()

const { name } = usePersistUserStore()
const api = useApiClient()
const createKey = useIdempotencyKey()
const commentValue = ref('')
const isPublishing = ref(false)

const handlePublishComment = async () => {
  if (isPublishing.value) {
    return
  }

  const body = {
    text: commentValue.value,
    ...(props.parentCommentId
      ? { parent_comment_id: props.parentCommentId }
      : {})
  }
  const parsed = createCommentSchema.safeParse(body)
  if (!parsed.success) {
    const message = JSON.parse(parsed.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }

  isPublishing.value = true
  const result = await settle(
    api.POST('/replies/{reply_id}/comments', {
      params: {
        path: { reply_id: props.replyId },
        header: {
          'Idempotency-Key': createKey.take(
            `/replies/${props.replyId}/comments`,
            body
          )
        }
      },
      body
    })
  )
  isPublishing.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  createKey.clear()
  commentValue.value = ''
  emits('getComment', result.data)
  useMessage(10224, 'success')
  emits('closePanel')
}

const handleClose = () => {
  emits('closePanel')
}
</script>

<template>
  <div class="w-full space-y-3 pt-2">
    <div class="flex items-center gap-1">
      {{ `${name} 评论` }}
      <UserHoverCard :user-id="targetUser.id">
        <KunUserChip size="sm" :user="targetUser" />
      </UserHoverCard>
    </div>

    <KunTextarea
      name="comment"
      placeholder="请输入您的评论, 最大字数为 1007"
      :rows="5"
      v-model="commentValue"
    />

    <div class="flex w-full justify-end space-x-1">
      <KunButton variant="light" color="danger" @click="handleClose">
        关闭
      </KunButton>
      <KunButton
        :disabled="isPublishing"
        :loading="isPublishing"
        @click="handlePublishComment"
      >
        发布评论
      </KunButton>
    </div>
  </div>
</template>
