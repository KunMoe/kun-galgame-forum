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
        <GalgameCardCover
          :src="card.work.cover?.url"
          :alt="card.name"
          :thumbhash="card.work.cover?.thumbhash ?? undefined"
          :masked="isBlurred && card.work.is_nsfw"
        />
        <div class="line-clamp-2 p-2 text-sm font-medium">
          <SearchHighlight :text="card.name" :keywords="keywords" />
        </div>
      </KunCard>
    </div>
  </div>
</template>
