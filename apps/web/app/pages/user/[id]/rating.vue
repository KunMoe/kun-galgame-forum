<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

const props = defineProps<{
  user: UserProfile
}>()

useKunDisableSeo(`${props.user.name ?? ''} 的评分`)

useHead({
  link: [
    {
      rel: 'canonical',
      href: `${kungal.domain.main}/user/${props.user.id}/rating`
    }
  ]
})
</script>

<template>
  <UserRating :user-id="Number(user.id)" />
</template>
