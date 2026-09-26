<script setup lang="ts">
const props = defineProps<{
  src?: string
  alt: string
  thumbhash?: string
  masked: boolean
}>()

const failedSrc = ref('')
const cover = computed(() => (props.src !== failedSrc.value ? props.src : ''))
</script>

<template>
  <KunNsfwMask v-if="cover" :active="masked" class-name="">
    <KunImage
      :src="cover"
      loading="lazy"
      :alt="alt"
      :thumbhash="thumbhash"
      aspect-ratio="5 / 7"
      @error="failedSrc = cover"
    />
  </KunNsfwMask>
  <KunImage
    v-else
    src="/galgame-no-cover.webp"
    loading="lazy"
    :alt="alt"
    aspect-ratio="5 / 7"
    object-fit="contain"
    class-name="bg-default-100"
  />
</template>
