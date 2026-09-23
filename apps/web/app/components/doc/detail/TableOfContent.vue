<script setup lang="ts">
import type { DocumentHeading } from '#shared/utils/content/plainText'

defineProps<{
  links: DocumentHeading[]
}>()
</script>

<template>
  <aside class="hidden w-64 shrink-0 space-y-3 lg:block">
    <h3 class="p-3 text-xl font-semibold">文档目录</h3>
    <nav class="scrollbar-hide max-h-[calc(100dvh-12rem)] overflow-y-auto">
      <DocDetailTOCLink :links="links" />
    </nav>
  </aside>
</template>
