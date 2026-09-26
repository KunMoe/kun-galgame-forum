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
const notifyBusy = ref(false)

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

const notifyAll = computed(() => followState.value?.notify_level !== 'feed')

const toggleNotify = async () => {
  if (!followState.value?.is_following || notifyBusy.value) {
    return
  }
  notifyBusy.value = true
  const notify = notifyAll.value ? 'feed' : 'all'
  const result = await settle(
    api.PATCH('/me/following/{user_id}', {
      params: { path: { user_id: props.userId } },
      body: { notify }
    })
  )
  notifyBusy.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  followState.value = result.data
  useMessage(
    notify === 'all'
      ? 'TA 发布新内容时会通知你'
      : '不再通知, TA 的动态仍会出现在「关注」里',
    'success'
  )
}

onMounted(() => {
  void loadFollowState()
})
watch(currentUserId, () => {
  void loadFollowState()
})
</script>

<template>
  <div v-if="isVisible" class="flex items-center gap-1">
    <KunButton
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
    <KunTooltip
      v-if="followState?.is_following"
      :text="notifyAll ? 'TA 发布新内容时通知我' : '只在「关注」动态中显示'"
    >
      <KunButton
        :is-icon-only="true"
        variant="flat"
        size="xs"
        :loading="notifyBusy"
        :aria-label="notifyAll ? '关闭发布通知' : '开启发布通知'"
        @click="toggleNotify"
      >
        <KunIcon :name="notifyAll ? 'lucide:bell-ring' : 'lucide:bell-off'" />
      </KunButton>
    </KunTooltip>
  </div>
</template>
