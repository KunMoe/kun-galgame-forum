<script setup lang="ts">
import type { CollectionSummary } from '#shared/utils/api/schemas'

const props = defineProps<{
  collection: Pick<
    CollectionSummary,
    'cover_work' | 'recent_works' | 'content_rating'
  >
}>()

const { isBlurred } = useContentStance()
const arts = computed(() =>
  collectionCoverArts(props.collection, isBlurred.value)
)
</script>

<template>
  <div
    class="bg-default-100 relative overflow-hidden rounded-md"
    style="aspect-ratio: 5 / 7"
  >
    <img
      v-if="arts.length === 1"
      :src="arts[0]"
      alt=""
      loading="lazy"
      class="size-full object-cover"
    />
    <div
      v-else-if="arts.length > 1"
      class="grid size-full grid-cols-2 grid-rows-2 gap-0.5"
    >
      <img
        v-for="(url, i) in arts"
        :key="url"
        :src="url"
        alt=""
        loading="lazy"
        :class="
          cn(
            'size-full object-cover',
            arts.length === 2 && 'row-span-2',
            arts.length === 3 && i === 0 && 'row-span-2'
          )
        "
      />
    </div>
    <div
      v-else
      class="text-default-300 flex size-full items-center justify-center"
    >
      <KunIcon name="lucide:folder-heart" class="text-3xl" />
    </div>
  </div>
</template>
