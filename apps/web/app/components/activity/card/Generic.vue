<script setup lang="ts">
import type { Activity } from '#shared/utils/api/schemas'

defineProps<{ activity: Activity }>()

const nameOf = useWorkName()
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
      class-name="hover:text-primary block break-all transition-colors"
    >
      <KunText
        class-name="whitespace-normal!"
        :content="activitySummaryText(activity, nameOf)"
      />
    </KunLink>
  </ActivityCardShell>
</template>
