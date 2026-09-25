<script setup lang="ts">
import type { UserFollowState, UserProfile } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  userId: UserProfile['id']
}>()

const emit = defineEmits<{
  change: [delta: number]
}>()

const api = useApiClient()
const { id: currentUserId } = storeToRefs(usePersistUserStore())
const followState = ref<UserFollowState | null>(null)
const followBusy = ref(false)

const isVisible = computed(
  () => !currentUserId.value || followState.value !== null
)

const followLabel = computed(() => {
  if (!followState.value?.is_following) {
    return '关注'
  }
  return followState.value.is_followed_by ? '互相关注' : '已关注'
})

const loadFollowState = async () => {
  followState.value = null
  if (!currentUserId.value) {
    return
  }
  const result = await settle(
    api.GET('/me/following/{user_id}', {
      params: { path: { user_id: props.userId } }
    })
  )
  if (result.ok) {
    followState.value = result.data
  }
}

const toggleFollow = async () => {
  if (!currentUserId.value) {
    useAuthModal().open()
    return
  }
  if (!followState.value || followBusy.value) {
    return
  }
  followBusy.value = true
  const wasFollowing = followState.value.is_following
  const path = { params: { path: { user_id: props.userId } } }
  const result = await settle(
    wasFollowing
      ? api.DELETE('/me/following/{user_id}', path)
      : api.PUT('/me/following/{user_id}', path)
  )
  followBusy.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  followState.value = result.data
  if (wasFollowing !== result.data.is_following) {
    emit('change', result.data.is_following ? 1 : -1)
  }
}

onMounted(() => {
  void loadFollowState()
})
watch(currentUserId, () => {
  void loadFollowState()
})
</script>

<template>
  <KunButton
    v-if="isVisible"
    variant="flat"
    size="xs"
    :color="followState?.is_following ? 'default' : 'primary'"
    class-name="gap-1"
    :loading="followBusy"
    @click="toggleFollow"
  >
    <KunIcon
      :name="
        followState?.is_following
          ? 'lucide:user-round-check'
          : 'lucide:user-plus'
      "
    />
    {{ followLabel }}
  </KunButton>
</template>
