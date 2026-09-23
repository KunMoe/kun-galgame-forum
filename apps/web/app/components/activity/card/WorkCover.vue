<script setup lang="ts">
import type { WorkRef } from '#shared/utils/api/schemas'

const props = defineProps<{ work: WorkRef }>()

const nameOf = useWorkName()
const { isBlurred } = useContentStance()
const masked = computed(() => props.work.is_nsfw && isBlurred.value)
</script>

<template>
  <KunLink :to="`/galgame/${work.id}`" class-name="shrink-0">
    <div
      class="bg-default-100 aspect-[3/4] w-20 overflow-hidden rounded-lg sm:w-24"
    >
      <img
        v-if="work.cover"
        :src="work.cover.url"
        :alt="nameOf(work)"
        loading="lazy"
        :class="cn('h-full w-full object-cover', masked && 'blur-xl')"
      />
    </div>
  </KunLink>
</template>
