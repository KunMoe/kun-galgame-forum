<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = withDefaults(
  defineProps<{
    userId: UserProfile['id']
    block?: boolean
    compact?: boolean
  }>(),
  { block: false, compact: false }
)

const emit = defineEmits<{
  change: [delta: number]
}>()

const api = useApiClient()
const { id: currentUserId } = storeToRefs(usePersistUserStore())
const {
  state: followState,
  busy: followBusy,
  set: setFollowState,
  load: loadFollowState,
  toggle
} = useFollowState(() => props.userId)
const notifyBusy = ref(false)
const followedHere = ref(false)

const isVisible = computed(() => {
  if (!currentUserId.value) {
    return true
  }
  if (!followState.value) {
    return false
  }
  return !props.compact || !followState.value.is_following || followedHere.value
})

const followLabel = computed(() => {
  if (!followState.value?.is_following) {
    return '关注'
  }
  return followState.value.is_followed_by ? '互相关注' : '已关注'
})

const followIcon = computed(() =>
  followState.value?.is_following
    ? 'lucide:user-round-check'
    : 'lucide:user-plus'
)

const toggleFollow = async () => {
  const delta = await toggle()
  followedHere.value = followState.value?.is_following ?? false
  if (delta) {
    emit('change', delta)
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
  setFollowState(result.data)
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
  <KunTooltip v-if="isVisible && compact" :text="followLabel">
    <KunButton
      :is-icon-only="true"
      variant="flat"
      size="sm"
      :color="followState?.is_following ? 'default' : 'primary'"
      :loading="followBusy"
      :aria-label="followLabel"
      @click="toggleFollow"
    >
      <KunIcon :name="followIcon" />
    </KunButton>
  </KunTooltip>

  <div
    v-else-if="isVisible"
    :class="cn('flex items-center gap-1', block && 'w-full')"
  >
    <KunButton
      variant="flat"
      :size="block ? 'sm' : 'xs'"
      :color="followState?.is_following ? 'default' : 'primary'"
      :class-name="cn('gap-1', block && 'flex-1 justify-center')"
      :loading="followBusy"
      @click="toggleFollow"
    >
      <KunIcon :name="followIcon" />
      {{ followLabel }}
    </KunButton>
    <KunTooltip
      v-if="followState?.is_following"
      :text="notifyAll ? 'TA 发布新内容时通知我' : '只在「关注」动态中显示'"
    >
      <KunButton
        :is-icon-only="true"
        variant="flat"
        :size="block ? 'sm' : 'xs'"
        :loading="notifyBusy"
        :aria-label="notifyAll ? '关闭发布通知' : '开启发布通知'"
        @click="toggleNotify"
      >
        <KunIcon :name="notifyAll ? 'lucide:bell-ring' : 'lucide:bell-off'" />
      </KunButton>
    </KunTooltip>
  </div>
</template>
