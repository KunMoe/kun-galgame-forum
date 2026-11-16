<script setup lang="ts">
const props = defineProps<{
  src: string
  name: string
}>()

const isOpen = ref(false)
const images = computed(() => [{ src: props.src, alt: `${props.name} logo` }])
</script>

<template>
  <div class="shrink-0">
    <button
      type="button"
      class="bg-default-100 block cursor-zoom-in rounded-xl p-2 sm:p-3"
      :aria-label="`查看 ${name} 的 logo`"
      @click="isOpen = true"
    >
      <KunImage
        :src="src"
        :alt="`${name} logo`"
        loading="eager"
        object-fit="contain"
        class-name="size-16 sm:size-28 lg:size-36"
      />
    </button>
    <KunLightbox v-model:is-open="isOpen" :images="images" />
  </div>
</template>
