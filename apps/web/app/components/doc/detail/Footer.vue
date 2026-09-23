<script setup lang="ts">
const route = useRoute()

const { docs } = await useAllDocs('published_desc')

const currentIndex = computed(() =>
  docs.value.findIndex((doc) => `/doc/${doc.slug}` === route.path)
)

const prev = computed(() =>
  currentIndex.value > 0 ? docs.value[currentIndex.value - 1] : null
)

const next = computed(() =>
  currentIndex.value !== -1 && currentIndex.value < docs.value.length - 1
    ? docs.value[currentIndex.value + 1]
    : null
)
</script>

<template>
  <div class="flex items-center justify-between">
    <KunButton
      class-name="mr-auto justify-start text-start gap-2"
      v-if="prev"
      color="default"
      variant="light"
      :href="`/doc/${prev.slug}`"
    >
      <KunIcon name="lucide:chevron-left" />
      {{ prev.title }}
    </KunButton>
    <KunButton
      class-name="ml-auto justify-end text-end gap-2"
      color="default"
      v-if="next"
      variant="light"
      :href="`/doc/${next.slug}`"
    >
      {{ next.title }}
      <KunIcon name="lucide:chevron-right" />
    </KunButton>
  </div>
</template>
