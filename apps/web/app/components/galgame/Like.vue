<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  workId: number
  likeCount: number
  isLiked: boolean
}>()

const { id } = usePersistUserStore()
const api = useApiClient()
const isLiked = ref(props.isLiked)
const likesCount = ref(props.likeCount)

watch(
  () => [props.isLiked, props.likeCount] as const,
  ([liked, count]) => {
    isLiked.value = liked
    likesCount.value = count
  }
)

const pending = ref(false)
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
  pending.value = true
  const params = { params: { path: { work_id: String(props.workId) } } }
  const result = await settle(
    next
      ? api.PUT('/works/{work_id}/like', params)
      : api.DELETE('/works/{work_id}/like', params)
  )
  pending.value = false
  if (!result.ok) {
    revert(next)
    reportProblem(result.problem)
    return
  }
  isLiked.value = result.data.viewer?.has_liked ?? next
  likesCount.value = result.data.like_count
  useMessage(next ? 10530 : 10531, 'success')
}
</script>

<template>
  <KunTooltip text="点赞">
    <span class="flex">
      <KunReaction
        v-model="isLiked"
        v-model:count="likesCount"
        :disabled="pending"
        icon="lucide:thumbs-up"
        color="primary"
        label="点赞"
        @change="onChange"
      />
    </span>
  </KunTooltip>
</template>
