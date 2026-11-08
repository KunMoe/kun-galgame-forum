<script setup lang="ts">
import type { CollectionSummary } from '#shared/utils/api/schemas'
import { deletedUserName } from '#shared/utils/deletedUser'

const props = withDefaults(
  defineProps<{
    collection: CollectionSummary
    showOwner?: boolean
    showNewItems?: boolean
  }>(),
  { showOwner: true, showNewItems: false }
)

const nameOf = useWorkName()

const ownerName = computed(() => props.collection.owner.name ?? deletedUserName)
const title = computed(() =>
  collectionDisplayName(props.collection, ownerName.value)
)
const blurb = computed(() => collectionBlurb(props.collection, nameOf))
const visibility = computed(() =>
  collectionVisibilityMeta(props.collection.visibility)
)
const newItems = computed(() => props.collection.viewer?.new_item_count ?? 0)
</script>

<template>
  <NuxtLink
    :to="`/collection/${collection.id}`"
    class="hover:bg-default-100 flex h-full gap-4 rounded-lg p-2 transition-colors"
  >
    <GalgameCollectionCover
      :collection="collection"
      class="w-24 shrink-0 self-start sm:w-28"
    />

    <div class="flex min-w-0 flex-1 flex-col gap-2">
      <div class="flex items-start gap-2">
        <h3 class="line-clamp-2 flex-1 text-lg leading-snug font-semibold">
          {{ title }}
        </h3>
        <KunChip
          v-if="showNewItems && newItems > 0"
          color="danger"
          variant="flat"
          size="sm"
          class-name="shrink-0"
        >
          +{{ newItems >= 100 ? '99+' : newItems }} 部新作
        </KunChip>
      </div>
      <div
        v-if="
          !showOwner &&
          (collection.visibility !== 'public' || collection.is_default)
        "
        class="text-default-500 flex items-center gap-3 text-sm"
      >
        <span
          v-if="collection.visibility !== 'public'"
          class="inline-flex items-center gap-1 leading-none"
        >
          <KunIcon :name="visibility.icon" class="size-4" />
          {{ visibility.label }}
        </span>
        <span v-if="collection.is_default">默认收藏夹</span>
      </div>
      <div
        v-if="showOwner"
        class="text-default-500 flex min-w-0 items-center gap-1.5 text-sm"
      >
        <KunAvatar
          :user="toKunUser(collection.owner)"
          size="xs"
          :is-navigation="false"
          class-name="shrink-0"
        />
        <span class="truncate">{{ ownerName }}</span>
      </div>
      <p
        v-if="blurb"
        class="text-default-600 line-clamp-3 break-words whitespace-pre-line"
      >
        {{ blurb }}
      </p>
      <div v-if="collection.tag_names.length" class="flex flex-wrap gap-1.5">
        <span
          v-for="tag in collection.tag_names"
          :key="tag"
          class="bg-default-100 text-default-600 rounded-md px-2 py-0.5 text-xs"
        >
          #{{ tag }}
        </span>
      </div>
      <GalgameCollectionStats
        :collection="collection"
        size="md"
        class="mt-auto pt-1"
      />
    </div>
  </NuxtLink>
</template>
