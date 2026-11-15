<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

const props = defineProps<{
  user: UserProfile
}>()

const { id } = storeToRefs(usePersistUserStore())
const isOwner = computed(() => id.value === Number(props.user.id))

useKunDisableSeo(`${props.user.name ?? ''} 关注的会社`)
</script>

<template>
  <UserFollowedCompaniesContainer :user-id="user.id" :is-owner="isOwner" />
</template>
