<script setup lang="ts">
const props = defineProps<{
  workId: number
  targetUserId: number
  likeCount: number
  isLiked: boolean
}>()

const { id } = usePersistUserStore()
const isLiked = ref(props.isLiked)
const likesCount = ref(props.likeCount)

watch(
  () => props.isLiked,
  (value) => (isLiked.value = value)
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
  if (id === props.targetUserId) {
    useMessage(10533, 'warn')
    revert(next)
    return
  }
  pending.value = true
  const result = await kunFetch(`/galgame/${props.workId}/like`, {
    method: 'PUT'
  })
  pending.value = false
  if (!result) {
    revert(next)
    return
  }
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
