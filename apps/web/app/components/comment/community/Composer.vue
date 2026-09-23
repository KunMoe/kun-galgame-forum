<script setup lang="ts">
import type { WallComment } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'

const props = withDefaults(
  defineProps<{
    target: CommunityCommentTarget
    parentCommentId?: string | null
    isReply?: boolean
  }>(),
  { parentCommentId: null, isReply: false }
)

const emits = defineEmits<{
  close: []
  submitted: [post: WallComment]
}>()

const surface = communityCommentSurface(props.target)

const { id } = usePersistUserStore()
const api = useApiClient()
const createKey = useIdempotencyKey()

const content = ref('')
const isPublishing = ref(false)

const handlePublish = async () => {
  if (!id) {
    useAuthModal().open()
    return
  }
  const trimmed = content.value.trim()
  if (!trimmed) {
    useMessage(10540, 'warn')
    return
  }
  if ([...trimmed].length > surface.maxLength) {
    useMessage(`评论最大长度为 ${surface.maxLength} 个字符`, 'warn')
    return
  }

  const body = {
    subject_type: surface.subjectType,
    subject_id: surface.subjectId,
    content_markdown: trimmed,
    ...(props.parentCommentId
      ? { parent_comment_id: props.parentCommentId }
      : {})
  }
  isPublishing.value = true
  const result = await settle(
    api.POST('/wall-comments', {
      params: {
        header: { 'Idempotency-Key': createKey.take('/wall-comments', body) }
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
  content.value = ''
  useMessage(result.data.state === 'held' ? '已提交，待审核' : 10542, 'success')
  emits('submitted', result.data)
  emits('close')
}
</script>

<template>
  <div class="space-y-3">
    <KunMilkdownDualEditorProvider
      :value-markdown="content"
      :placeholder="surface.composerPlaceholder"
      @set-markdown="(val) => (content = val)"
    />

    <div class="flex items-center justify-between gap-2">
      <slot />

      <KunButton class="ml-auto" :loading="isPublishing" @click="handlePublish">
        {{ isReply ? '发布回复' : '发布评论' }}
      </KunButton>
    </div>
  </div>
</template>
