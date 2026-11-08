<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  collectionId: string
  following: boolean
  followerCount: number
}>()

const emits = defineEmits<{
  changed: [following: boolean, followerCount: number]
}>()

const api = useApiClient()
const submitting = ref(false)

const toggle = async () => {
  if (!requireLogin() || submitting.value) {
    return
  }
  submitting.value = true
  const params = { path: { collection_id: props.collectionId } }
  const result = await settle(
    props.following
      ? api.DELETE('/collections/{collection_id}/follow', { params })
      : api.PUT('/collections/{collection_id}/follow', { params })
  )
  submitting.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  const following = result.data.viewer?.is_following ?? !props.following
  const fallback = Math.max(props.followerCount + (following ? 1 : -1), 0)
  emits('changed', following, result.data.follower_count ?? fallback)
  useMessage(following ? '已关注这个收藏夹' : '已取消关注', 'success')
}
</script>

<template>
  <KunButton
    size="sm"
    :variant="following ? 'flat' : 'solid'"
    :color="following ? 'default' : 'primary'"
    :loading="submitting"
    @click="toggle"
  >
    <KunIcon
      :name="following ? 'lucide:bookmark-check' : 'lucide:bookmark-plus'"
    />
    {{ following ? '已关注' : '关注' }}
    <span class="opacity-70">{{ followerCount }}</span>
  </KunButton>
</template>
