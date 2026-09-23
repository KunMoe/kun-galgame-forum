<script setup lang="ts">
import type { WallComment } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  comment: WallComment
}>()

const { id } = usePersistUserStore()
const api = useApiClient()
const isLiked = ref(props.comment.viewer?.has_liked ?? false)
const likesCount = ref(props.comment.like_count)
const pending = ref(false)

watch(
  () => [props.comment.viewer?.has_liked, props.comment.like_count] as const,
  ([liked, count]) => {
    isLiked.value = liked ?? false
    likesCount.value = count
  }
)

const revert = (next: boolean) => {
  isLiked.value = !next
  likesCount.value += next ? -1 : 1
}

const onChange = async (next: boolean) => {
  if (!id) {
    useAuthModal().open()
    revert(next)
    return
  }
  if (!props.comment.viewer?.can_like) {
    useMessage(10533, 'warn')
    revert(next)
    return
  }

  pending.value = true
  const params = { params: { path: { wall_comment_id: props.comment.id } } }
  const result = await settle(
    next
      ? api.PUT('/wall-comments/{wall_comment_id}/like', params)
      : api.DELETE('/wall-comments/{wall_comment_id}/like', params)
  )
  pending.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    revert(next)
    return
  }
  isLiked.value = result.data.viewer?.has_liked ?? next
  likesCount.value = result.data.like_count
}
</script>

<template>
  <KunTooltip text="点赞">
    <KunReaction
      v-model="isLiked"
      v-model:count="likesCount"
      :disabled="pending"
      size="sm"
      icon="lucide:thumbs-up"
      color="primary"
      label="点赞"
      @change="onChange"
    />
  </KunTooltip>
</template>
