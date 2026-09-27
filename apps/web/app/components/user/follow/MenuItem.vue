<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

const props = defineProps<{
  userId: UserProfile['id']
}>()

const { id: currentUserId } = storeToRefs(usePersistUserStore())
const { state, busy, load, toggle } = useFollowState(() => props.userId)

const isFollowing = computed(() => state.value?.is_following ?? false)
const isPending = computed(() => !!currentUserId.value && !state.value)

onMounted(() => {
  void load()
})
</script>

<template>
  <KunButton
    variant="light"
    color="default"
    size="sm"
    class-name="w-full justify-start gap-2 whitespace-nowrap"
    :loading="busy || isPending"
    @click="toggle"
  >
    <KunIcon
      class-name="text-lg"
      :name="isFollowing ? 'lucide:user-round-minus' : 'lucide:user-plus'"
    />
    {{ isFollowing ? '取消关注 TA' : '关注 TA' }}
  </KunButton>
</template>
