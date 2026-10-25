<script setup lang="ts">
import type { Image } from '#shared/utils/api/schemas'

defineProps<{
  image?: Image
  name: string
  size: 'bust' | 'avatar'
  icon?: string
}>()
</script>

<template>
  <div
    :class="
      cn(
        'bg-default-100 text-default-400 relative flex shrink-0 items-center justify-center overflow-hidden',
        size === 'bust'
          ? 'aspect-[3/4] w-full rounded-xl'
          : 'size-7 rounded-full'
      )
    "
  >
    <KunImage
      v-if="image"
      :src="withImageVariant(image.url, 'mini')"
      :alt="name"
      loading="lazy"
      :thumbhash="image.thumbhash ?? undefined"
      object-fit="cover"
      class-name="absolute inset-0 size-full"
      image-class-name="size-full object-top"
    />
    <KunIcon
      v-else-if="icon"
      :name="icon"
      :class-name="size === 'bust' ? 'size-8' : 'size-4'"
    />
    <span
      v-else
      :class="
        size === 'bust' ? 'text-3xl font-semibold' : 'text-xs font-medium'
      "
    >
      {{ name.slice(0, 1) }}
    </span>
  </div>
</template>
