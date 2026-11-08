<script setup lang="ts">
import type { CollectionSummary } from '#shared/utils/api/schemas'
import { deletedUserName } from '#shared/utils/deletedUser'

const props = defineProps<{
  collection: CollectionSummary
}>()

const ownerName = computed(() => props.collection.owner.name ?? deletedUserName)
</script>

<template>
  <NuxtLink
    :to="`/collection/${collection.id}`"
    class="hover:bg-default-100 block space-y-2 rounded-lg p-2.5 transition-colors"
  >
    <p class="line-clamp-2 leading-snug font-semibold">
      {{ collectionDisplayName(collection, ownerName) }}
    </p>
    <div class="text-default-500 flex min-w-0 items-center gap-1.5 text-sm">
      <KunAvatar
        :user="toKunUser(collection.owner)"
        size="xs"
        :is-navigation="false"
        class-name="shrink-0"
      />
      <span class="truncate">{{ ownerName }}</span>
    </div>
    <GalgameCollectionStats :collection="collection" size="md" />
  </NuxtLink>
</template>
