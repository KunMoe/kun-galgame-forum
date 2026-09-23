<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  resourceId: string
  targetUserId: number
  isLiked: boolean
  likeCount: number
}>()

const { id } = usePersistUserStore()
const api = useApiClient()
const isLiked = ref(props.isLiked)
const likeCount = ref(props.likeCount)
const pending = ref(false)

watch(
  () => [props.isLiked, props.likeCount] as const,
  ([liked, count]) => {
    isLiked.value = liked
    likeCount.value = count
  }
)

const revert = (next: boolean) => {
  isLiked.value = !next
  likeCount.value += next ? -1 : 1
}

const onChange = async (next: boolean) => {
  if (!id) {
    useAuthModal().open()
    revert(next)
    return
  }
  if (id === props.targetUserId) {
    useMessage('您不能给自己点赞', 'warn')
    revert(next)
    return
  }
  pending.value = true
  const params = { params: { path: { resource_id: props.resourceId } } }
  const result = await settle(
    next
      ? api.PUT('/galgame-resources/{resource_id}/like', params)
      : api.DELETE('/galgame-resources/{resource_id}/like', params)
  )
  pending.value = false
  if (!result.ok) {
    revert(next)
    reportProblem(result.problem)
    return
  }
  isLiked.value = result.data.viewer?.has_liked ?? next
  likeCount.value = result.data.like_count
  useMessage(next ? '点赞资源成功' : '取消点赞成功', 'success')
}
</script>

<template>
  <KunReaction
    v-model="isLiked"
    v-model:count="likeCount"
    :disabled="pending"
    size="sm"
    icon="lucide:thumbs-up"
    color="primary"
    label="点赞"
    @change="onChange"
  />
</template>
