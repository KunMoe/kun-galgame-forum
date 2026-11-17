<script setup lang="ts">
defineProps<{
  title: string
  cover: AgentCardCover | null
}>()
</script>

<template>
  <div class="bg-default-100 relative size-full overflow-hidden">
    <KunImage
      v-if="cover"
      :src="cover.src"
      :alt="title"
      :thumbhash="cover.thumbhash ?? undefined"
      object-fit="cover"
      :class-name="cn('size-full', cover.blurred && 'blur-md')"
    />
    <span
      v-else
      class="text-default-400 flex size-full items-center justify-center text-lg font-semibold"
      aria-hidden="true"
    >
      {{ (Array.from(title.trim())[0] ?? '?').toUpperCase() }}
    </span>
  </div>
</template>
