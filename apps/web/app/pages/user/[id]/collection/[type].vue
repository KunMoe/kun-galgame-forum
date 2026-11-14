<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'
import { KUN_USER_PAGE_COLLECTION_TYPE } from '~/constants/user'

definePageMeta({
  validate: (route) =>
    (KUN_USER_PAGE_COLLECTION_TYPE as readonly string[]).includes(
      String(route.params.type)
    )
})

const props = defineProps<{
  user: UserProfile
}>()

const route = useRoute()
const collectionType = computed(() => {
  const t = (route.params as { type: string }).type
  return KUN_USER_PAGE_COLLECTION_TYPE.find((type) => type === t) ?? 'galgame'
})

useKunDisableSeo(`${props.user.name ?? ''}的收藏`)
</script>

<template>
  <div>
    <UserCollectionGalgame
      v-if="collectionType === 'galgame'"
      :user-id="Number(user.id)"
      :owner-name="user.name"
    />
    <UserCollectionTopic
      v-else-if="collectionType === 'topic'"
      :user-id="Number(user.id)"
    />
    <UserToolset v-else :user-id="Number(user.id)" relation="favorited" />
  </div>
</template>
