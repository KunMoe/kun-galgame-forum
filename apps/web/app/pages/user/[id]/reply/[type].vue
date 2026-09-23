<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'
import {
  REPLY_NAV_CONFIG,
  type KUN_USER_PAGE_REPLY_TYPE
} from '~/constants/user'

const props = defineProps<{
  user: UserProfile
}>()

const route = useRoute()
const replyType = computed(() => {
  const routeType =
    (route.params as { type: string }).type.replace(/-/g, '_') || 'valid'
  return routeType as (typeof KUN_USER_PAGE_REPLY_TYPE)[number]
})

useKunDisableSeo(
  `${props.user.name ?? ''}${REPLY_NAV_CONFIG[replyType.value].text}`
)
</script>

<template>
  <UserReply :user-id="Number(user.id)" :type="replyType" />
</template>
