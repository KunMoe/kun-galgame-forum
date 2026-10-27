<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'
import { TOPIC_NAV_CONFIG, KUN_USER_PAGE_TOPIC_TYPE } from '~/constants/user'

definePageMeta({
  validate: (route) =>
    (KUN_USER_PAGE_TOPIC_TYPE as readonly string[]).includes(
      String(route.params.type).replace(/-/g, '_')
    )
})

const props = defineProps<{
  user: UserProfile
}>()

const route = useRoute()
const topicType = computed(() => {
  const routeType =
    (route.params as { type: string }).type.replace(/-/g, '_') || 'topic'
  return routeType as (typeof KUN_USER_PAGE_TOPIC_TYPE)[number]
})

useKunDisableSeo(
  `${props.user.name ?? ''}${TOPIC_NAV_CONFIG[topicType.value].text}的话题`
)
</script>

<template>
  <UserTopic :user-id="Number(user.id)" :type="topicType" />
</template>
