<script setup lang="ts">
import type { Activity, WorkRef } from '#shared/utils/api/schemas'

defineProps<{ activity: Activity; work: WorkRef }>()

const nameOf = useWorkName()
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div class="space-y-1.5">
      <ActivityCollapse :max-height="300">
        <p class="text-default-700 text-base break-all whitespace-pre-line">
          {{ activity.excerpt_markdown }}
        </p>
      </ActivityCollapse>
      <KunLink
        underline="none"
        color="default"
        :to="activity.path"
        class-name="text-default-500 hover:text-primary inline-flex items-center gap-1 text-sm"
      >
        <KunIcon name="lucide:gamepad-2" class="size-3.5 shrink-0" />
        {{ nameOf(work) }}
      </KunLink>
    </div>
  </ActivityCardShell>
</template>
