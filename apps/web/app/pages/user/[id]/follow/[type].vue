<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

definePageMeta({
  validate: (route) => {
    const type = route.params.type
    return type === 'followers' || type === 'following'
  }
})

const props = defineProps<{
  user: UserProfile
}>()

const route = useRoute()
const followType = computed(() =>
  (route.params as { type: string }).type === 'following'
    ? 'following'
    : 'followers'
)

useKunDisableSeo(
  `${props.user.name ?? ''}的${followType.value === 'followers' ? '粉丝' : '关注'}`
)
</script>

<template>
  <UserFollowContainer :key="followType" :user="user" :type="followType" />
</template>
