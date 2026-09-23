<script setup lang="ts">
import type { WorkRef } from '#shared/utils/api/schemas'

const props = defineProps<{
  works: WorkRef[]
  keywords?: string
}>()

const { isBlurred } = useContentStance()
const workName = useWorkName()
const { isOpenInNewTab } = storeToRefs(usePersistGalgameCardStore())

const cards = computed(() =>
  props.works.map((work) => ({
    work,
    name: workName(work)
  }))
)
</script>

<template>
  <div class="@container">
    <div
      class="grid grid-cols-2 gap-2 @lg:grid-cols-3 @lg:gap-3 @2xl:grid-cols-4 @4xl:grid-cols-5 @5xl:grid-cols-6"
    >
      <KunCard
        v-for="card in cards"
        :key="card.work.id"
        :is-transparent="true"
        :href="`/galgame/${card.work.id}`"
        :target="isOpenInNewTab ? '_blank' : undefined"
        class-name="p-0 h-full"
      >
        <KunNsfwMask
          v-if="card.work.cover"
          :active="isBlurred && card.work.is_nsfw"
          class-name=""
        >
          <KunImage
            :src="card.work.cover.url"
            loading="lazy"
            :alt="card.name"
            :thumbhash="card.work.cover.thumbhash ?? undefined"
            aspect-ratio="5 / 7"
          />
        </KunNsfwMask>
        <div
          v-else
          class="bg-default-100 text-default-400 flex items-center justify-center"
          style="aspect-ratio: 5 / 7"
        >
          <KunIcon name="lucide:image-off" class="size-6" />
        </div>
        <div class="line-clamp-2 p-2 text-sm font-medium">
          <SearchHighlight :text="card.name" :keywords="keywords" />
        </div>
      </KunCard>
    </div>
  </div>
</template>
