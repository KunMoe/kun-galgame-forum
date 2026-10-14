<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

const props = defineProps<{
  user: UserProfile
}>()

const { id } = storeToRefs(usePersistUserStore())
const isOwner = computed(() => id.value === Number(props.user.id))

useKunDisableSeo(`${props.user.name ?? ''} 的游戏库`)
</script>

<template>
  <UserLibraryContainer :user-id="Number(user.id)" :is-owner="isOwner" />
</template>
