<script setup lang="ts">
import type { Activity, ActivityReply } from '#shared/utils/api/schemas'

const props = defineProps<{ activity: Activity; reply: ActivityReply }>()

const link = computed(() =>
  replyPermalink(`/topic/${props.reply.topic_id}`, props.reply.floor)
)
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div
      class="text-success-600 dark:text-success-400 flex items-center gap-1.5 text-sm font-medium"
    >
      <KunIcon name="lucide:bookmark-check" class-name="text-base" />
      采纳了最佳答案
    </div>

    <div class="bg-success-500/10 mt-2 rounded-lg p-3">
      <ActivityCollapse :max-height="96">
        <ContentDocument
          :document="reply.content"
          compact
          class-name="text-default-600 text-sm"
        />
      </ActivityCollapse>
    </div>

    <div
      class="text-default-500 mt-2 flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-sm"
    >
      <KunLink
        underline="hover"
        color="default"
        :to="link"
        class-name="text-default-500 hover:text-primary inline-flex min-w-0 items-center gap-1.5"
      >
        <KunIcon name="icon-park-outline:topic" class-name="shrink-0" />
        <span class="truncate">{{ reply.topic_title }}</span>
      </KunLink>

      <KunLink
        underline="none"
        color="default"
        :to="link"
        class-name="text-default-500 hover:text-primary flex shrink-0 items-center gap-0.5 text-sm"
      >
        查看详情
        <KunIcon name="lucide:chevron-right" class="size-4" />
      </KunLink>
    </div>
  </ActivityCardShell>
</template>
