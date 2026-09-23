<script setup lang="ts">
import type { NotificationSummary } from '#shared/utils/api/schemas'
import { markdownToText } from '#shared/utils/markdownToText'

const props = defineProps<{
  summary?: NotificationSummary
}>()

const preview = computed(() =>
  markdownToText(props.summary?.latest?.excerpt_markdown ?? '').trim()
)
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    class-name="hover:bg-primary/20 flex cursor-pointer flex-nowrap gap-3 rounded-lg p-2 transition-colors hover:opacity-80"
    to="/message/notice"
  >
    <KunImage
      src="/apple-touch-icon.png"
      class="h-12 w-12 shrink-0 rounded-full"
    />
    <div class="justify-space flex w-full flex-col">
      <div class="flex items-center justify-between">
        <span class="font-bold">通知</span>
        <span
          class="text-default-500 text-sm"
          v-if="summary?.latest?.created_at"
        >
          <KunTime :time="summary.latest.created_at" />
        </span>
      </div>

      <div class="flex items-center justify-between text-sm">
        <span class="line-clamp-1 break-all">
          {{ preview }}
        </span>
        <KunChip
          class-name="whitespace-nowrap"
          color="primary"
          v-if="summary && summary.unread_count > 0"
        >
          {{ summary.unread_count }}
        </KunChip>
      </div>
    </div>
  </KunLink>
</template>
