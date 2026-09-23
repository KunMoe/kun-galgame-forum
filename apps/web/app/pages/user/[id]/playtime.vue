<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

definePageMeta({
  middleware: 'auth'
})

const props = defineProps<{
  user: UserProfile
}>()

const { id } = usePersistUserStore()
const isOwner = computed(() => id === Number(props.user.id))

useKunDisableSeo('游玩时长')
</script>

<template>
  <UserPlaytime v-if="isOwner" />
  <KunNull v-else description="游玩时长只对本人可见" />
</template>
