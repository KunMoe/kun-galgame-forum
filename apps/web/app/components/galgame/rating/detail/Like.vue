<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  ratingId?: number
  targetUserId: number
  likeCount: number
  isLiked: boolean
}>()

const { id } = usePersistUserStore()
const api = useApiClient()
const isLiked = ref(props.isLiked)
const likeCount = ref(props.likeCount)
const pending = ref(false)

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
    useMessage(10236, 'warn')
    revert(next)
    return
  }
  pending.value = true
  const params = { path: { rating_id: String(props.ratingId) } }
  const result = await settle(
    next
      ? api.PUT('/ratings/{rating_id}/like', { params })
      : api.DELETE('/ratings/{rating_id}/like', { params })
  )
  pending.value = false
  if (!result.ok) {
    revert(next)
    reportProblem(result.problem)
    return
  }
  isLiked.value = result.data.viewer.has_liked
  likeCount.value = result.data.like_count
  useMessage(next ? 10233 : 10234, 'success')
}
</script>

<template>
  <KunReaction
    v-model="isLiked"
    v-model:count="likeCount"
    :disabled="pending"
    icon="lucide:thumbs-up"
    color="primary"
    label="点赞"
    @change="onChange"
  />
</template>
