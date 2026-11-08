<script setup lang="ts">
import { useSortable, moveArrayElement } from '@vueuse/integrations/useSortable'
import { settle } from '#shared/utils/api/problem'
import type { CollectionEntry } from '#shared/utils/api/schemas'

const props = defineProps<{
  collectionId: string
  entries: CollectionEntry[]
  canEditNotes: boolean
  canReorder: boolean
  canReport: boolean
  canSetCover?: boolean
  coverWorkId?: string | null
  marks?: Record<string, { collected: boolean; playState: string | null }>
  canMark?: boolean
}>()

const emits = defineEmits<{
  changed: []
  setCover: [workId: string | null]
  marked: [workId: string, playState: string | null]
}>()

const api = useApiClient()
const nameOf = useWorkName()
const catalogName = useCatalogName()

const rows = ref<CollectionEntry[]>([])
watch(
  () => props.entries,
  (value) => {
    rows.value = [...value]
  },
  { immediate: true, flush: 'sync' }
)

const moving = ref(false)
const listEl = ref<HTMLElement | null>(null)

const persistMove = async (from: number, to: number) => {
  const moved = rows.value[to]
  if (!moved) {
    return
  }
  const before = rows.value[to + 1]
  const after = rows.value[to - 1]
  const body = before
    ? { before_work_id: before.work_summary.id }
    : { after_work_id: after?.work_summary.id }
  moving.value = true
  const result = await settle(
    api.PATCH('/collections/{collection_id}/works/{work_id}', {
      params: {
        path: {
          collection_id: props.collectionId,
          work_id: moved.work_summary.id
        }
      },
      body
    })
  )
  moving.value = false
  if (!result.ok) {
    moveArrayElement(rows, to, from)
    reportProblem(result.problem)
    return
  }
  emits('changed')
}

const { option } = useSortable(listEl, rows, {
  animation: 150,
  handle: '.collection-drag-handle',
  disabled: !props.canReorder,
  onUpdate: (e: { oldIndex?: number; newIndex?: number }) => {
    if (e.oldIndex == null || e.newIndex == null || e.oldIndex === e.newIndex) {
      return
    }
    const from = e.oldIndex
    const to = e.newIndex
    moveArrayElement(rows, from, to)
    nextTick(() => persistMove(from, to))
  }
})

watch(
  () => props.canReorder,
  (can) => option('disabled', !can),
  { flush: 'post' }
)

const noteTarget = ref<CollectionEntry | null>(null)
const noteOpen = ref(false)
const openNote = (entry: CollectionEntry) => {
  noteTarget.value = entry
  noteOpen.value = true
}
const onNoteSaved = (workId: string, note: string | null) => {
  rows.value = rows.value.map((row) =>
    row.work_summary.id === workId ? { ...row, note } : row
  )
  emits('changed')
}

const reportTarget = ref<CollectionEntry | null>(null)
const reportOpen = ref(false)
const openReport = (entry: CollectionEntry) => {
  if (!requireLogin()) {
    return
  }
  reportTarget.value = entry
  reportOpen.value = true
}
</script>

<template>
  <div>
    <ol ref="listEl" class="space-y-8">
      <li v-for="entry in rows" :key="entry.work_summary.id" class="flex gap-2">
        <button
          v-if="canReorder"
          type="button"
          :disabled="moving"
          class="collection-drag-handle text-default-400 hover:text-default-700 -ml-1 flex shrink-0 cursor-grab touch-none items-center active:cursor-grabbing"
          aria-label="拖动以调整顺序"
        >
          <KunIcon name="lucide:grip-vertical" />
        </button>

        <div class="min-w-0 flex-1 space-y-3">
          <GalgameCollectionEntryRow
            :entry="entry"
            :mark="marks?.[entry.work_summary.id]"
            :is-cover="coverWorkId === entry.work_summary.id"
          >
            <template v-if="canMark" #mark>
              <GalgamePlayStateMark
                :work-id="entry.work_summary.id"
                :play-state="marks?.[entry.work_summary.id]?.playState ?? null"
                size="xs"
                @changed="
                  (saved) =>
                    emits('marked', entry.work_summary.id, saved.play_state)
                "
              />
            </template>
            <template v-if="canSetCover" #cover>
              <KunTooltip
                :text="
                  coverWorkId === entry.work_summary.id
                    ? '取消封面，改回自动拼图'
                    : '设为收藏夹封面'
                "
              >
                <KunButton
                  :variant="
                    coverWorkId === entry.work_summary.id ? 'solid' : 'flat'
                  "
                  size="xs"
                  is-icon-only
                  :color="
                    coverWorkId === entry.work_summary.id
                      ? 'primary'
                      : 'default'
                  "
                  :class-name="
                    cn(
                      'shadow-sm',
                      coverWorkId !== entry.work_summary.id &&
                        'bg-background/85 backdrop-blur'
                    )
                  "
                  :aria-label="
                    coverWorkId === entry.work_summary.id
                      ? '取消封面'
                      : '设为收藏夹封面'
                  "
                  @click="
                    emits(
                      'setCover',
                      coverWorkId === entry.work_summary.id
                        ? null
                        : entry.work_summary.id
                    )
                  "
                >
                  <KunIcon name="lucide:image" />
                </KunButton>
              </KunTooltip>
            </template>
            <template #actions>
              <KunButton
                v-if="canEditNotes"
                variant="light"
                size="xs"
                is-icon-only
                :aria-label="entry.note ? '编辑备注' : '添加备注'"
                @click="openNote(entry)"
              >
                <KunIcon name="lucide:message-square-text" />
              </KunButton>
              <KunButton
                v-if="canReport && entry.note"
                variant="light"
                size="xs"
                color="danger"
                is-icon-only
                aria-label="举报这条备注"
                @click="openReport(entry)"
              >
                <KunIcon name="lucide:flag" />
              </KunButton>
            </template>
          </GalgameCollectionEntryRow>

          <div v-if="entry.top_tags.length" class="flex flex-wrap gap-1.5">
            <NuxtLink
              v-for="tag in entry.top_tags"
              :key="tag.id"
              :to="`/galgame/tag/${tag.id}`"
              class="bg-default-100 text-default-600 hover:bg-primary/15 hover:text-primary rounded-md px-2 py-0.5 text-sm transition-colors"
            >
              {{ catalogName(tag).name }}
            </NuxtLink>
          </div>

          <blockquote
            v-if="entry.note"
            class="bg-default-100 text-default-700 rounded-lg px-3 py-2 break-words whitespace-pre-wrap"
          >
            {{ entry.note }}
          </blockquote>
        </div>
      </li>
    </ol>

    <GalgameCollectionNoteModal
      v-if="canEditNotes"
      v-model="noteOpen"
      :collection-id="collectionId"
      :work-id="noteTarget?.work_summary.id ?? ''"
      :work-name="noteTarget ? nameOf(noteTarget.work_summary) : ''"
      :note="noteTarget?.note ?? ''"
      @saved="onNoteSaved"
    />
    <GalgameCollectionReportModal
      v-if="canReport"
      v-model="reportOpen"
      :collection-id="collectionId"
      :work-id="reportTarget?.work_summary.id"
      :work-name="reportTarget ? nameOf(reportTarget.work_summary) : ''"
    />
  </div>
</template>
