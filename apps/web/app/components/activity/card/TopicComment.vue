<script setup lang="ts">
import type { Activity, ActivityComment } from '#shared/utils/api/schemas'

defineProps<{ activity: Activity; comment: ActivityComment }>()
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div class="space-y-2">
      <ActivityCardQuote v-if="comment.quoted" :quoted="comment.quoted" />

      <ActivityCollapse :max-height="300">
        <p class="text-default-700 text-base break-all whitespace-pre-line">
          {{
            markdownToText(activity.excerpt_markdown, {
              preserveNewlines: true
            })
          }}
        </p>
      </ActivityCollapse>

      <KunLink
        underline="none"
        color="default"
        :to="
          commentPermalink(
            `/topic/${comment.topic_id}`,
            Number(comment.comment_id)
          )
        "
        class-name="text-default-500 hover:text-primary flex items-center gap-1 text-sm"
      >
        <KunIcon name="icon-park-outline:topic" class="size-4 shrink-0" />
        <span class="line-clamp-1">{{ comment.topic_title }}</span>
      </KunLink>
    </div>
  </ActivityCardShell>
</template>
