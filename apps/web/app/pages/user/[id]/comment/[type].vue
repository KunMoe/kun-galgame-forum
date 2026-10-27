<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'
import {
  COMMENT_NAV_CONFIG,
  KUN_USER_PAGE_COMMENT_TYPE
} from '~/constants/user'

definePageMeta({
  validate: (route) =>
    (KUN_USER_PAGE_COMMENT_TYPE as readonly string[]).includes(
      String(route.params.type).replace(/-/g, '_')
    )
})

const props = defineProps<{
  user: UserProfile
}>()

const route = useRoute()
const commentType = computed(() => {
  const routeType =
    (route.params as { type: string }).type.replace(/-/g, '_') || 'valid'
  return routeType as (typeof KUN_USER_PAGE_COMMENT_TYPE)[number]
})

useKunDisableSeo(
  `${props.user.name ?? ''}${COMMENT_NAV_CONFIG[commentType.value].text}`
)
</script>

<template>
  <UserComment :user-id="Number(user.id)" :type="commentType" />
</template>
