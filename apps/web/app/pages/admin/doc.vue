<script setup lang="ts">
import { useSortable, moveArrayElement } from '@vueuse/integrations/useSortable'
import { settle } from '#shared/utils/api/problem'
import type { AdminDoc, DocSummary } from '#shared/utils/api/schemas'
import type { DocEditorMode } from '~/components/edit/doc/type'
import { KUN_DOC_CATEGORY_MAP } from '~/constants/doc'

definePageMeta({
  middleware: 'permission',
  permissions: ['doc.create', 'doc.edit', 'doc.delete']
})

useKunDisableSeo('文档管理')

const api = useApiClient()
const { docs, refresh: refetch } = await useAllDocs('position_asc')

const list = ref<DocSummary[]>([...docs.value])

const refresh = async () => {
  await refetch()
  list.value = [...docs.value]
}

const persistOrder = async () => {
  const result = await settle(
    api.PUT('/admin/doc-order', {
      body: { doc_ids: list.value.map((doc) => doc.id) }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    await refresh()
  }
}

const listEl = ref<HTMLElement | null>(null)
useSortable(listEl, list, {
  animation: 150,
  handle: '.doc-drag-handle',
  onUpdate: (e) => {
    if (e.oldIndex == null || e.newIndex == null) return
    moveArrayElement(list, e.oldIndex, e.newIndex)
    nextTick(() => persistOrder())
  }
})

const isModalOpen = ref(false)
const modalMode = ref<DocEditorMode>('create')
const editingDoc = ref<AdminDoc | null>(null)

const openCreate = () => {
  modalMode.value = 'create'
  editingDoc.value = null
  isModalOpen.value = true
}

const openEdit = async (row: DocSummary) => {
  const result = await settle(
    api.GET('/admin/docs/{doc_id}', { params: { path: { doc_id: row.id } } })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  modalMode.value = 'rewrite'
  editingDoc.value = result.data
  isModalOpen.value = true
}

const onSaved = async () => {
  isModalOpen.value = false
  await refresh()
}

const handleDelete = async (row: DocSummary) => {
  const confirmed = await useComponentMessageStore().alert(
    `确认删除文档「${row.title}」吗？`,
    '删除操作不可恢复，请慎重。',
    true
  )
  if (!confirmed) return

  const result = await settle(
    api.DELETE('/admin/docs/{doc_id}', { params: { path: { doc_id: row.id } } })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('删除文档成功', 'success')
  await refresh()
}

const handleTogglePin = async (row: DocSummary, value: boolean) => {
  const prev = row.is_pinned
  row.is_pinned = value
  const result = await settle(
    api.PATCH('/admin/docs/{doc_id}', {
      params: { path: { doc_id: row.id } },
      body: { is_pinned: value }
    })
  )
  if (!result.ok) {
    row.is_pinned = prev
    reportProblem(result.problem)
    return
  }
  useMessage(value ? '已置顶' : '已取消置顶', 'success')
}
</script>

<template>
  <div class="w-full space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold">文档管理</h1>
        <p class="text-default-500 text-sm">
          新建 / 编辑 / 删除文档, 拖拽左侧手柄可调整文档的展示顺序
        </p>
      </div>
      <KunButton color="primary" @click="openCreate">
        <KunIcon name="lucide:plus" />
        新建文档
      </KunButton>
    </div>

    <div ref="listEl" class="space-y-3">
      <KunCard
        v-for="article in list"
        :key="article.id"
        :is-hoverable="false"
        :is-transparent="false"
        padding="sm"
      >
        <div class="flex items-center gap-3">
          <KunIcon
            name="lucide:grip-vertical"
            class="doc-drag-handle text-default-400 shrink-0 cursor-grab active:cursor-grabbing"
          />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="truncate font-medium">{{ article.title }}</span>
            </div>
            <span class="text-default-500 block truncate text-xs">
              {{ KUN_DOC_CATEGORY_MAP[article.doc_category] }}
            </span>
          </div>
          <div class="flex shrink-0 items-center gap-1.5">
            <span class="text-default-500 text-sm">置顶</span>
            <KunSwitch
              :model-value="article.is_pinned"
              color="primary"
              @update:model-value="(v) => handleTogglePin(article, v)"
            />
          </div>
          <KunButton
            size="sm"
            variant="light"
            color="primary"
            @click="openEdit(article)"
          >
            编辑
          </KunButton>
          <KunButton
            size="sm"
            variant="light"
            color="danger"
            @click="handleDelete(article)"
          >
            删除
          </KunButton>
        </div>
      </KunCard>
    </div>

    <KunNull v-if="!list.length" description="暂无文档, 点击右上角新建" />

    <KunModal
      v-model="isModalOpen"
      :is-dismissable="false"
      inner-class-name="max-w-4xl w-full"
      scroll-behavior="inside"
    >
      <EditDocLayout
        v-if="isModalOpen"
        :mode="modalMode"
        :initial-doc="editingDoc"
        :redirect-on-success="false"
        @saved="onSaved"
      />
    </KunModal>
  </div>
</template>
