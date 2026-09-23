<script setup lang="ts">
import type { NotificationSummary } from '#shared/utils/api/schemas'

defineProps<{
  summary?: NotificationSummary
}>()
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    class-name="hover:bg-primary/20 flex cursor-pointer flex-nowrap gap-3 rounded-lg p-2 transition-colors hover:opacity-80"
    to="/message/muted"
  >
    <div
      class="bg-default-100 flex h-12 w-12 shrink-0 items-center justify-center rounded-full"
    >
      <KunIcon name="lucide:bell-off" class="text-default-500 text-xl" />
    </div>
    <div class="flex w-full flex-col justify-center">
      <span class="font-bold">已静音的消息</span>
      <div class="flex items-center justify-between text-sm">
        <span class="text-default-500 line-clamp-1">已被你静音的通知</span>
        <KunChip
          v-if="summary && summary.muted_unread_count > 0"
          color="default"
          class-name="whitespace-nowrap"
        >
          {{ summary.muted_unread_count }}
        </KunChip>
      </div>
    </div>
  </KunLink>
</template>
