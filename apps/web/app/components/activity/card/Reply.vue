<script setup lang="ts">
import type { Activity, ActivityReply } from '#shared/utils/api/schemas'

defineProps<{ activity: Activity; reply: ActivityReply }>()
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div class="space-y-2">
      <ActivityCardQuote v-if="reply.quoted" :quoted="reply.quoted" />

      <ContentDocument
        :document="reply.content"
        compact
        class-name="text-base"
      />

      <KunLink
        underline="none"
        color="default"
        :to="replyPermalink(`/topic/${reply.topic_id}`, reply.floor)"
        class-name="text-default-500 hover:text-primary flex items-center gap-1 text-sm"
      >
        <KunIcon name="icon-park-outline:topic" class="size-4 shrink-0" />
        <span class="line-clamp-1">{{ reply.topic_title }}</span>
      </KunLink>
    </div>
  </ActivityCardShell>
</template>
