<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'
import {
  GALGAME_RESOURCE_NAV_CONFIG,
  type KUN_USER_PAGE_GALGAME_RESOURCE_TYPE
} from '~/constants/user'

const props = defineProps<{
  user: UserProfile
}>()

const route = useRoute()
const resourceType = computed(() => {
  const routeType =
    (route.params as { type: string }).type.replace(/-/g, '_') || 'valid'
  return routeType as (typeof KUN_USER_PAGE_GALGAME_RESOURCE_TYPE)[number]
})

useKunDisableSeo(
  `${props.user.name ?? ''} 的${
    GALGAME_RESOURCE_NAV_CONFIG[resourceType.value].text
  }`
)
</script>

<template>
  <UserResource :user-id="Number(user.id)" :type="resourceType" />
</template>
