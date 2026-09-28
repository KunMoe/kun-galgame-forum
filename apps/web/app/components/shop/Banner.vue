<script setup lang="ts">
import { usePreferredReducedMotion } from '@vueuse/core'
import type { ShopItemArtwork } from '#shared/utils/api/schemas'

const props = defineProps<{
  artwork: ShopItemArtwork | null | undefined
  className?: string
}>()

const reduceMotion = usePreferredReducedMotion()

const src = computed(() => {
  const art = props.artwork
  if (!art) return ''
  return reduceMotion.value === 'no-preference' && art.animated_url
    ? art.animated_url
    : art.static_url
})
</script>

<template>
  <div
    :class="cn('bg-default-100 aspect-[3/1] w-full overflow-hidden', className)"
  >
    <KunImage
      v-if="src"
      :src="src"
      alt=""
      class-name="size-full"
      object-fit="cover"
    />
  </div>
</template>
