<script setup lang="ts">
import { workEditFieldLabel } from '~/constants/workEditFields'
import { settle } from '#shared/utils/api/problem'
import type { Activity, WorkRef } from '#shared/utils/api/schemas'

const props = defineProps<{ activity: Activity; work: WorkRef }>()

const nameOf = useWorkName()
const { workHistoryUrl } = useNextmoeEditor()

const api = useApiClient()
const changedLabels = ref<string[]>([])
const isLoading = ref(false)

const loadChanges = async () => {
  const workId = props.work.id
  const revision = props.activity.work_revision
  if (!revision || isLoading.value) return
  isLoading.value = true
  try {
    let seq = revision.revision_number ?? undefined
    if (!seq) {
      const rowId = revision.legacy_revision_id
      const history = await settle(
        api.GET('/works/{work_id}/edit-revisions', {
          params: { path: { work_id: workId }, query: { limit: 100 } }
        })
      )
      seq = history.ok
        ? history.data.items.find((r) => r.id === rowId)?.seq
        : undefined
    }
    if (!seq || seq <= 1) return
    const res = await settle(
      api.GET('/works/{work_id}/edit-revisions/diff', {
        params: {
          path: { work_id: workId },
          query: { from_seq: seq - 1, to_seq: seq }
        }
      })
    )
    if (res.ok) {
      changedLabels.value = [
        ...new Set(
          res.data.field_changes.map((row) => workEditFieldLabel(row.key))
        )
      ]
    }
  } finally {
    isLoading.value = false
  }
}
onMounted(loadChanges)
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

      <div v-if="isLoading" class="text-default-400 text-sm">加载编辑内容…</div>
      <div
        v-else-if="changedLabels.length"
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
        <span class="text-default-500">该更新已经被合并到 鲲Galgame百科</span>
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
