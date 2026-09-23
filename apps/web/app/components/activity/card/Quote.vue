<script setup lang="ts">
import type { QuotedReply } from '#shared/utils/api/schemas'

defineProps<{ quoted: QuotedReply }>()
</script>

<template>
  <div class="text-default-500 flex gap-2 text-sm">
    <div class="bg-default-300 w-1 shrink-0 rounded-full" />
    <div class="min-w-0">
      <span class="text-default-400 mr-1 text-xs font-medium">
        #{{ quoted.floor }}
      </span>
      <span class="text-default-500 line-clamp-2 break-all">
        {{ markdownToText(quoted.excerpt_markdown) }}
      </span>
    </div>
  </div>
</template>
