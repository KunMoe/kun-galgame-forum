<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

const props = defineProps<{
  user: UserProfile
}>()

const { id } = storeToRefs(usePersistUserStore())
const isOwner = computed(() => id.value === Number(props.user.id))

useKunDisableSeo('关注的收藏夹')
</script>

<template>
  <div>
    <GalgameCollectionFollowedList v-if="isOwner" />
    <KunNull v-else description="关注的收藏夹只有本人可以看到" />
  </div>
</template>
