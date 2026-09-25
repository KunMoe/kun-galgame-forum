<script setup lang="ts">
import type { GalgameArtMeta } from '~~/shared/types/galgame'

const props = defineProps<{
  src: string
  alt: string
  label: string
  meta?: GalgameArtMeta
  maxWidth: number
  maxHeight: number
}>()

defineEmits<{ open: [] }>()

const box = computed(() => artBox(props.meta, props.maxWidth, props.maxHeight))
</script>

<template>
  <button
    type="button"
    class="bg-default-100 block max-w-full shrink-0 cursor-zoom-in overflow-hidden rounded-xl"
    :style="{ width: `${box.width}px`, height: `${box.height}px` }"
    :aria-label="label"
    @click="$emit('open')"
  >
    <KunImage
      :src="src"
      :alt="alt"
      loading="eager"
      :thumbhash="meta?.thumbhash || undefined"
      object-fit="contain"
      class-name="size-full"
      image-class-name="size-full"
    />
  </button>
</template>
