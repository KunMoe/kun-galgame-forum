<script setup lang="ts">
defineProps<{
  src: string
  name: string
}>()
</script>

<template>
  <div class="bg-default-100 shrink-0 rounded-xl p-2 sm:p-3">
    <KunImage
      :src="src"
      :alt="`${name} logo`"
      loading="eager"
      object-fit="contain"
      class-name="size-16 sm:size-28 lg:size-36"
    />
  </div>
</template>
