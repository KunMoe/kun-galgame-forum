<script setup lang="ts">
const props = defineProps<{
  collection: CollectionSummary
  ownerName?: string
}>()

const displayName = computed(() =>
  collectionDisplayName(props.collection, props.ownerName)
)

const visibilityMeta = (v: CollectionVisibility) =>
  v === 'private'
    ? { icon: 'lucide:lock', label: '私密' }
    : { icon: 'lucide:globe', label: '公开' }
</script>

<template>
  <KunCard
    :href="`/galgame/collection/${collection.id}`"
    content-class="space-y-3"
  >
    <div
      class="bg-default-100 grid aspect-video grid-cols-2 grid-rows-2 gap-0.5 overflow-hidden rounded-lg"
    >
      <template v-if="collection.preview_covers.length">
        <img
          v-for="(cover, index) in collection.preview_covers.slice(0, 4)"
          :key="index"
          :src="cover"
          :class="
            cn(
              'size-full object-cover',
              collection.preview_covers.length === 1 && 'col-span-2 row-span-2'
            )
          "
          alt=""
        />
      </template>
      <div
        v-else
        class="text-default-300 col-span-2 row-span-2 flex items-center justify-center"
      >
        <KunIcon name="lucide:heart" class="text-4xl" />
      </div>
    </div>

    <div class="space-y-1">
      <div class="flex items-center gap-2">
        <span class="truncate font-medium">{{ displayName }}</span>
        <span
          v-if="collection.is_default"
          class="text-default-400 shrink-0 text-xs"
        >
          默认
        </span>
      </div>
      <div class="text-default-500 flex items-center gap-1.5 text-xs">
        <KunIcon :name="visibilityMeta(collection.visibility).icon" />
        <span>{{ visibilityMeta(collection.visibility).label }}</span>
        <span class="ml-auto">{{ collection.item_count }} 个游戏</span>
      </div>
    </div>
  </KunCard>
</template>
