<script setup lang="ts">
import {
  galgameEditFieldConfig,
  galgameEditLabel
} from '~/constants/galgameEdit'
import { settle } from '#shared/utils/api/problem'
import type { Activity, WorkRef } from '#shared/utils/api/schemas'
import type { EditRevisionDiff } from '~/utils/galgame/editAdapt'

const props = defineProps<{ activity: Activity; work: WorkRef }>()

const nameOf = useWorkName()

const api = useApiClient()
const diff = ref<EditRevisionDiff | null>(null)
const isLoading = ref(false)

const loadDiff = async () => {
  const workId = props.work.id
  const revision = props.activity.work_revision
  if (!revision || diff.value || isLoading.value) return
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
    if (res.ok) diff.value = res.data
  } finally {
    isLoading.value = false
  }
}
onMounted(loadDiff)

const DIFF_COLLAPSED_MAX_HEIGHT = 100
const diffRef = ref<HTMLElement | null>(null)
const isExpanded = ref(false)
const isOverflowing = ref(false)
let resizeObserver: ResizeObserver | null = null

const measureOverflow = () => {
  const el = diffRef.value
  if (!el) {
    isOverflowing.value = false
    return
  }
  isOverflowing.value = el.scrollHeight > DIFF_COLLAPSED_MAX_HEIGHT
}

const diffStyle = computed(() => {
  if (!isOverflowing.value || isExpanded.value) return undefined
  return { maxHeight: `${DIFF_COLLAPSED_MAX_HEIGHT}px`, overflow: 'hidden' }
})

watch(diff, () =>
  nextTick(() => {
    if (diffRef.value && !resizeObserver) {
      resizeObserver = new ResizeObserver(() => measureOverflow())
      resizeObserver.observe(diffRef.value)
    }
    measureOverflow()
  })
)

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
})
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
        v-else-if="diff && diff.field_changes.length"
        class="border-default-200 rounded-lg border p-2 text-sm"
      >
        <div ref="diffRef" :style="diffStyle" class="space-y-3">
          <EditkitFieldDiff
            v-for="row in diff.field_changes"
            :key="row.key"
            :label="galgameEditLabel(row.key)"
            :from="row.from"
            :to="row.to"
            :config="galgameEditFieldConfig(row.key)"
          />
        </div>
        <button
          v-if="isOverflowing"
          type="button"
          class="text-primary mt-1 flex items-center gap-1 text-sm"
          @click="isExpanded = !isExpanded"
        >
          {{ isExpanded ? '收起' : '显示更多' }}
          <KunIcon
            :name="isExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
            class="size-4"
          />
        </button>
      </div>

      <div class="flex items-center justify-between gap-2 text-sm">
        <span class="text-default-500">该更新已经被合并到 鲲Galgame百科</span>
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
