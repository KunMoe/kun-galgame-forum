<script setup lang="ts">
const props = defineProps<{
  workId: number
  galgameResourceId: number
  targetUserId: number
  isLiked: boolean
  likeCount: number
}>()

const { id } = usePersistUserStore()
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
    useMessage('您不能给自己点赞', 'warn')
    revert(next)
    return
  }
  pending.value = true
  const result = await kunFetch(`/galgame/${props.workId}/resource/like`, {
    method: 'PUT',
    body: { galgame_resource_id: props.galgameResourceId }
  })
  pending.value = false
  if (!result) {
    revert(next)
    return
  }
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
