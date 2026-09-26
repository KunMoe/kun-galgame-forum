<script setup lang="ts">
import type { WorkRef } from '#shared/utils/api/schemas'

const props = defineProps<{ work: WorkRef }>()

const nameOf = useWorkName()
const { isBlurred } = useContentStance()
const masked = computed(() => props.work.is_nsfw && isBlurred.value)

const failedSrc = ref('')
const cover = computed(() => {
  const url = props.work.cover?.url
  return url !== failedSrc.value ? url : ''
})
</script>

<template>
  <KunLink :to="`/galgame/${work.id}`" class-name="shrink-0">
    <div
      class="bg-default-100 aspect-[3/4] w-20 overflow-hidden rounded-lg sm:w-24"
    >
      <KunImage
        v-if="cover"
        :src="cover"
        :alt="nameOf(work)"
        loading="lazy"
        aspect-ratio="3 / 4"
        :image-class-name="masked ? 'blur-xl' : undefined"
        @error="failedSrc = cover"
      />
      <KunImage
        v-else
        src="/galgame-no-cover.webp"
        :alt="nameOf(work)"
        loading="lazy"
        aspect-ratio="3 / 4"
        object-fit="contain"
      />
    </div>
  </KunLink>
</template>
