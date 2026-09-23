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
    <div class="space-y-3">
      <p class="text-default-600 text-sm break-all">
        提出了《{{ nameOf(work) }}》的更新请求
      </p>

      <ActivityCardGalgameInfo :work="work" :digest="activity.work_digest" />

      <div class="flex items-center justify-between gap-2 text-sm">
        <span class="text-warning-600">该更新请求需要被审核</span>
        <KunLink
          underline="none"
          color="default"
          :to="activity.path"
          class-name="text-default-500 hover:text-primary flex shrink-0 items-center gap-0.5 text-sm"
        >
          查看详情
          <KunIcon name="lucide:chevron-right" class="size-4" />
        </KunLink>
      </div>
    </div>
  </ActivityCardShell>
</template>
