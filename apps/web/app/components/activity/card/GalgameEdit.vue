<script setup lang="ts">
import { workEditFieldLabel } from '~/constants/workEditFields'
import type { Activity, WorkRef } from '#shared/utils/api/schemas'

const props = defineProps<{ activity: Activity; work: WorkRef }>()

const nameOf = useWorkName()
const { workHistoryUrl } = useNextmoeEditor()

const changedLabels = computed(() => [
  ...new Set(
    (props.activity.work_revision?.changed_fields ?? []).map(workEditFieldLabel)
  )
])
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div class="space-y-3">
      <p class="text-default-600 text-sm break-all">
        编辑了《{{ nameOf(work) }}》
      </p>

      <ActivityCardGalgameInfo :work="work" :digest="activity.work_digest" />

      <div
        v-if="changedLabels.length"
        class="flex flex-wrap items-center gap-1.5 text-sm"
      >
        <span class="text-default-500">修改了</span>
        <KunChip
          v-for="label in changedLabels"
          :key="label"
          size="sm"
          variant="flat"
        >
          {{ label }}
        </KunChip>
      </div>

      <div class="flex items-center justify-between gap-2 text-sm">
        <span class="text-default-500">该更新已经被合并到 鲲 Galgame 论坛</span>
        <div class="flex shrink-0 items-center gap-3">
          <KunLink
            underline="none"
            color="default"
            :href="workHistoryUrl(work.id, `/galgame/${work.id}`)"
            target="_blank"
            class-name="text-default-500 hover:text-primary flex items-center gap-0.5 text-sm"
          >
            查看修订
            <KunIcon name="lucide:external-link" class="size-3.5" />
          </KunLink>
          <KunLink
            underline="none"
            color="default"
            :to="activity.path"
            class-name="text-default-500 hover:text-primary flex items-center gap-0.5 text-sm"
          >
            查看详情
            <KunIcon name="lucide:chevron-right" class="size-4" />
          </KunLink>
        </div>
      </div>
    </div>
  </ActivityCardShell>
</template>
