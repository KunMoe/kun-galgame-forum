<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    companyId: number
    followerCount?: number | null
  }>(),
  { followerCount: null }
)

const emit = defineEmits<{
  change: [delta: number]
}>()

const { id: currentUserId } = storeToRefs(usePersistUserStore())
const {
  state: followState,
  busy: followBusy,
  load: loadFollowState,
  toggle
} = useCompanyFollowState(() => props.companyId)

const shownCount = ref<number | null>(props.followerCount)

watch(
  () => props.followerCount,
  (count) => {
    shownCount.value = count
  }
)

const visible = computed(
  () => !currentUserId.value || followState.value !== null
)

const followLabel = computed(() =>
  followState.value?.is_following ? '已关注' : '关注'
)

const followIcon = computed(() =>
  followState.value?.is_following ? 'lucide:bookmark-check' : 'lucide:bookmark'
)

const toggleFollow = async () => {
  const delta = await toggle()
  if (delta && shownCount.value !== null) {
    shownCount.value = Math.max(0, shownCount.value + delta)
  }
  if (delta) {
    emit('change', delta)
  }
}

const settled = ref(false)
onMounted(async () => {
  await loadFollowState()
  settled.value = true
})
watch(currentUserId, () => {
  void loadFollowState()
})
</script>

<template>
  <div
    v-if="visible || shownCount !== null || (currentUserId && !settled)"
    :class="cn('flex shrink-0 items-center gap-2')"
  >
    <KunSkeleton
      v-if="!visible && currentUserId && !settled"
      width="4.5rem"
      height="2rem"
    />
    <KunButton
      v-else-if="visible"
      variant="flat"
      size="sm"
      :color="followState?.is_following ? 'default' : 'primary'"
      class-name="gap-1"
      :loading="followBusy"
      @click="toggleFollow"
    >
      <KunIcon :name="followIcon" />
      {{ followLabel }}
    </KunButton>
    <span v-if="shownCount !== null" class="text-default-500 text-sm">
      {{ shownCount }} 人关注
    </span>
  </div>
</template>
