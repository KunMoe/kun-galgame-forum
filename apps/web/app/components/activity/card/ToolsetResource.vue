<script setup lang="ts">
import type { Activity, ActivityToolset } from '#shared/utils/api/schemas'

defineProps<{ activity: Activity; toolset: ActivityToolset }>()
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <KunLink
      underline="none"
      color="default"
      :to="activity.path"
      class-name="group block space-y-1.5"
    >
      <span class="text-default-500 flex items-center gap-1.5 text-sm">
        <KunIcon
          name="lucide:package-plus"
          class="text-secondary size-4 shrink-0"
        />
        在工具《{{ toolset.title }}》发布了资源
      </span>
      <p
        class="group-hover:text-primary line-clamp-3 text-base break-all transition-colors"
      >
        {{ markdownToText(activity.excerpt_markdown) }}
      </p>
    </KunLink>
  </ActivityCardShell>
</template>
