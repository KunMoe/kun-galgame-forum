<script setup lang="ts">
import { useContentLightbox } from '@kungal/ui-vue'
import type { Image } from '#shared/utils/api/schemas'

type Cover = string | Image

const props = defineProps<{
  images: Cover[]
  meta?: Record<string, KunImageMeta>
  zoomable?: boolean
  nsfw?: boolean
}>()

const { isBlurred } = useContentStance()
const masked = computed(() => !!props.nsfw && isBlurred.value)

const shown = computed(() => props.images.slice(0, 9))
const isSingle = computed(() => shown.value.length === 1)

const resolved = (item: Cover) => {
  if (typeof item === 'string') {
    const m = props.meta?.[item]
    return {
      src: imageTokenUrl(item),
      thumbhash: m?.thumbhash,
      width: m?.width,
      height: m?.height
    }
  }
  return {
    src: item.url,
    thumbhash: item.thumbhash ?? undefined,
    width: item.width ?? undefined,
    height: item.height ?? undefined
  }
}

const aspectOf = (item: Cover): string | undefined => {
  const m = resolved(item)
  return m.width && m.height ? `${m.width} / ${m.height}` : undefined
}

const SINGLE_MAX_HEIGHT_PX = 384

const singleWidth = computed(() => {
  const m = resolved(shown.value[0]!)
  if (!m.width || !m.height) {
    return undefined
  }
  const heightCapped = Math.round((SINGLE_MAX_HEIGHT_PX * m.width) / m.height)
  return { width: `min(${m.width}px, 100%, ${heightCapped}px)` }
})

const root = ref<HTMLElement | null>(null)
const lightboxRoot = props.zoomable ? root : ref<HTMLElement | null>(null)

const {
  isLightboxOpen,
  images: lightboxImages,
  currentImageIndex
} = useContentLightbox(lightboxRoot)
</script>

<template>
  <div v-if="shown.length">
    <div ref="root">
      <KunNsfwMask
        v-if="isSingle"
        :active="masked"
        :style="singleWidth"
        label="成人向封面已模糊"
      >
        <KunImage
          :src="resolved(shown[0]!).src"
          :thumbhash="resolved(shown[0]!).thumbhash"
          :aspect-ratio="aspectOf(shown[0]!)"
          :width="resolved(shown[0]!).width"
          :height="resolved(shown[0]!).height"
          alt="话题封面"
          loading="lazy"
          object-fit="cover"
          :class-name="cn('w-full rounded-lg', zoomable && 'cursor-zoom-in')"
        />
      </KunNsfwMask>

      <KunScrollShadow
        v-else
        axis="horizontal"
        shadow-size="2rem"
        scrollbar="thin"
      >
        <div class="flex gap-1.5">
          <KunNsfwMask
            v-for="(item, idx) in shown"
            :key="`${idx}-${resolved(item).src}`"
            :active="masked"
            class-name="shrink-0 rounded-lg"
            label="成人向封面已模糊"
          >
            <KunImage
              :src="resolved(item).src"
              :thumbhash="resolved(item).thumbhash"
              :aspect-ratio="aspectOf(item)"
              alt="话题封面"
              loading="lazy"
              object-fit="contain"
              :class-name="
                cn(
                  'h-40 w-auto shrink-0 rounded-lg',
                  zoomable && 'cursor-zoom-in'
                )
              "
            />
          </KunNsfwMask>
        </div>
      </KunScrollShadow>
    </div>

    <KunLightbox
      v-if="zoomable"
      v-model:is-open="isLightboxOpen"
      :images="lightboxImages"
      :initial-index="currentImageIndex"
    />
  </div>
</template>
