<script setup lang="ts">
import type { Activity } from '#shared/utils/api/schemas'

defineProps<{ activity: Activity }>()
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
      class-name="group block space-y-1"
    >
      <span class="text-default-500 flex items-center gap-1.5 text-sm">
        <KunIcon name="lucide:wrench" class="text-secondary size-4 shrink-0" />
        创建了工具
      </span>
      <p
        class="group-hover:text-primary text-lg font-medium break-all transition-colors"
      >
        {{ activity.excerpt_markdown }}
      </p>
    </KunLink>
  </ActivityCardShell>
</template>
