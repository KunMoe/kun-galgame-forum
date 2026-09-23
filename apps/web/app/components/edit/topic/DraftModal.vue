<script setup lang="ts">
import { useTopicDraft } from '~/composables/topic/useTopicDraft'
import type { TopicDraftSummary } from '#shared/utils/api/schemas'

const open = defineModel<boolean>({ required: true })

const {
  listDrafts,
  saveCurrentAsDraft,
  loadDraft,
  deleteDraft,
  hasCurrentContent
} = useTopicDraft()

const drafts = ref<TopicDraftSummary[]>([])
const isLoading = ref(false)
const isSaving = ref(false)

const refresh = async () => {
  isLoading.value = true
  drafts.value = await listDrafts()
  isLoading.value = false
}

watch(open, (isOpen) => {
  if (isOpen) {
    refresh()
  }
})

const onSave = async () => {
  if (isSaving.value) {
    return
  }
  if (!hasCurrentContent.value) {
    useMessage('当前没有可保存的内容', 'warn')
    return
  }
  isSaving.value = true
  const saved = await saveCurrentAsDraft()
  isSaving.value = false
  if (saved) {
    useMessage('已保存为草稿', 'success')
    refresh()
  }
}

const onLoad = async (draft: TopicDraftSummary) => {
  if (hasCurrentContent.value) {
    const ok = await useComponentMessageStore().alert(
      '载入草稿会覆盖当前正在编写的内容，确定吗？'
    )
    if (!ok) {
      return
    }
  }
  if (await loadDraft(draft.id)) {
    useMessage('草稿已载入', 'success')
    open.value = false
  }
}

const onDelete = async (draft: TopicDraftSummary) => {
  const ok = await useComponentMessageStore().alert('确定删除这份草稿吗？')
  if (!ok) {
    return
  }
  if (!(await deleteDraft(draft.id))) {
    return
  }
  useMessage('草稿已删除', 'success')
  refresh()
}

const displayTitle = (draft: TopicDraftSummary) =>
  draft.title.trim() ||
  truncateRunes(markdownToText(draft.summary), 40) ||
  '(无标题草稿)'
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-lg">
    <div class="flex max-h-[80vh] flex-col">
      <header class="mb-4 flex items-center gap-2">
        <KunIcon name="lucide:notebook-pen" class="text-primary-500 h-5 w-5" />
        <h2 class="text-xl font-bold">话题草稿</h2>
      </header>

      <KunButton
        color="primary"
        :disabled="isSaving"
        class="mb-4 shrink-0"
        @click="onSave"
      >
        <KunIcon
          v-if="isSaving"
          name="lucide:loader-2"
          class="mr-1 h-4 w-4 animate-spin"
        />
        <KunIcon v-else name="lucide:save" class="mr-1 h-4 w-4" />
        保存当前为草稿
      </KunButton>

      <div class="text-default-500 mb-2 text-sm">
        已保存的草稿（仅自己可见，最多 30 份）
      </div>

      <div class="min-h-0 flex-1 space-y-2 overflow-y-auto">
        <div v-if="isLoading" class="text-default-400 py-8 text-center text-sm">
          加载中…
        </div>
        <div
          v-else-if="!drafts.length"
          class="text-default-400 py-8 text-center text-sm"
        >
          还没有草稿
        </div>
        <div
          v-for="draft in drafts"
          v-else
          :key="draft.id"
          class="border-default-200 hover:border-primary-500 flex items-center gap-3 rounded-lg border p-3 transition-colors"
        >
          <div class="min-w-0 flex-1">
            <div class="truncate font-medium">{{ displayTitle(draft) }}</div>
            <div class="text-default-400 text-xs">
              <KunTime :time="draft.updated_at" type="datetime" />
            </div>
          </div>
          <KunButton size="sm" variant="light" @click="onLoad(draft)">
            载入
          </KunButton>
          <KunButton
            size="sm"
            variant="light"
            color="danger"
            :is-icon-only="true"
            @click="onDelete(draft)"
          >
            <KunIcon name="lucide:trash-2" />
          </KunButton>
        </div>
      </div>
    </div>
  </KunModal>
</template>
