<script setup lang="ts">
import {
  galgameEditFieldConfig,
  galgameEditLabel
} from '~/constants/galgameEdit'

const props = defineProps<{ activity: ActivityItem }>()
const data = computed(
  () => props.activity.data as GalgameActivityData | undefined
)

const diff = ref<GalgameEditDiff | null>(null)
const isLoading = ref(false)

const loadDiff = async () => {
  const workId = data.value?.galgame_id
  if (!workId || diff.value || isLoading.value) return
  isLoading.value = true
  try {
    let seq = data.value?.revision_number
    if (!seq) {
      const rowId = data.value?.revision_id
      if (!rowId) return
      const history = await kunFetch<GalgameEditRevisionList>(
        `/galgame/${workId}/edit/revisions?limit=200`
      )
      seq = history?.items?.find((r) => (r.legacy_id ?? r.id) === rowId)?.seq
    }
    if (!seq || seq <= 1) return
    const res = await kunFetch<GalgameEditDiff>(
      `/galgame/${workId}/edit/diff?from=${seq - 1}&to=${seq}`
    )
    if (res) diff.value = res
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
  <ActivityCardShell :actor="activity.actor" :timestamp="activity.timestamp">
    <div class="space-y-3">
      <p class="text-default-600 text-sm break-all">
        编辑了《{{ data?.name || activity.content }}》
      </p>

      <ActivityCardGalgameInfo :activity="activity" />

      <div v-if="isLoading" class="text-default-400 text-sm">加载编辑内容…</div>
      <div
        v-else-if="diff && diff.fields.length"
        class="border-default-200 rounded-lg border p-2 text-sm"
      >
        <div ref="diffRef" :style="diffStyle" class="space-y-3">
          <EditkitFieldDiff
            v-for="row in diff.fields"
            :key="row.key"
            :label="galgameEditLabel(row.key)"
            :diff-hint="row.diff_hint"
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
          :to="activity.link"
          class-name="text-default-500 hover:text-primary flex shrink-0 items-center gap-0.5 text-sm"
        >
          查看详情
          <KunIcon name="lucide:chevron-right" class="size-4" />
        </KunLink>
      </div>
    </div>
  </ActivityCardShell>
</template>
