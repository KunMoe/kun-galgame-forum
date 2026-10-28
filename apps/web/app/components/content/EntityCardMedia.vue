<script setup lang="ts">
import type { EntityCardMedia } from '~/utils/entityCardModel'

defineProps<{
  media: EntityCardMedia
  accent: string
  compact?: boolean
  topAnchored?: boolean
}>()

const BOX = {
  landscape: 'h-30 w-[86px] sm:w-[213px]',
  portrait: 'h-30 w-[86px]',
  photo: 'h-30 w-[86px]',
  logo: 'h-12 w-auto max-w-40 bg-white p-1'
} as const

const STACK_POSE = [
  'left-0 -rotate-6 z-0',
  'left-6 rotate-0 z-10',
  'left-12 rotate-6 z-20'
]
</script>

<template>
  <div
    v-if="media.kind === 'image'"
    :class="
      cn(
        'bg-default-100 shrink-0 overflow-hidden rounded-lg',
        media.shape === 'logo' ? 'kun-entity-logo' : 'kun-entity-fill',
        BOX[media.shape]
      )
    "
  >
    <KunImage
      :src="media.image.url"
      alt=""
      loading="lazy"
      :thumbhash="media.image.thumbhash ?? undefined"
      :object-fit="media.shape === 'logo' ? 'contain' : 'cover'"
      class-name="size-full"
      :image-class-name="
        cn(topAnchored && 'object-top', media.isBlurred && 'blur-lg')
      "
    />
  </div>
  <div
    v-else-if="media.kind === 'stack'"
    class="relative h-30 w-[120px] shrink-0"
  >
    <div
      v-for="(item, i) in media.images.slice(0, 3)"
      :key="item.image.url"
      :class="
        cn(
          'kun-entity-fill bg-default-100 ring-content1 absolute top-2.5 h-[100px] w-[71px] overflow-hidden rounded-md shadow-md ring-2',
          STACK_POSE[i]
        )
      "
    >
      <KunImage
        :src="item.image.url"
        alt=""
        loading="lazy"
        object-fit="cover"
        class-name="size-full"
        :image-class-name="cn(item.isBlurred && 'blur-lg')"
      />
    </div>
  </div>
  <div
    v-else-if="media.kind === 'avatar'"
    class="flex size-30 shrink-0 items-center justify-center"
  >
    <KunAvatar
      :user="media.user"
      :is-navigation="false"
      size="original"
      class-name="size-24 hover:scale-100"
      image-class-name="size-24"
    />
  </div>
  <div
    v-else
    :class="
      cn(
        'bg-default-100 flex shrink-0 items-center justify-center rounded-lg',
        compact ? 'size-12' : 'h-30 w-[86px]',
        accent
      )
    "
  >
    <KunIcon :name="media.icon" :class="compact ? 'size-6' : 'size-8'" />
  </div>
</template>

<style scoped>
/* Prose sets every img to height: auto, unlayered, so it beat the utility
   classes and a landscape cover sat 54px tall in its 120px frame in a
   published post (2026-10-28). The frame decides the size here. */
.kun-entity-fill :deep(img) {
  width: 100%;
  height: 100%;
}

.kun-entity-logo :deep(img) {
  width: auto;
  height: 100%;
}
</style>
