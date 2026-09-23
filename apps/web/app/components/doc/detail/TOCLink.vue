<script setup lang="ts">
import type { DocumentHeading } from '#shared/utils/content/plainText'

defineProps<{
  links: DocumentHeading[]
}>()

const DEPTH_PADDING: Record<number, string> = {
  2: 'pl-4',
  3: 'pl-8',
  4: 'pl-12',
  5: 'pl-16',
  6: 'pl-20'
}
</script>

<template>
  <div class="flex flex-col space-y-2">
    <div
      v-for="(link, index) in links"
      :key="`${link.anchor}-${index}`"
      :class="
        cn(
          'flex max-w-56 py-1 text-sm transition-all duration-200',
          DEPTH_PADDING[link.depth]
        )
      "
    >
      {{ link.text }}
    </div>
  </div>
</template>
