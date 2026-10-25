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

useKunDisableSeo('关注的厂商')
</script>

<template>
  <div>
    <UserFollowedCompaniesContainer v-if="isOwner" />
    <KunNull v-else description="关注的厂商只对本人可见" />
  </div>
</template>
