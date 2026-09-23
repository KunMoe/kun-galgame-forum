<script setup lang="ts">
import { KUN_UPDATE_LOG_CHANGE_TYPE_LABEL } from '~/constants/update'
import { problemMessage } from '#shared/utils/api/message'
import { settle } from '#shared/utils/api/problem'
import type { UpdateLog, UpdateLogCreate } from '#shared/utils/api/schemas'

const api = useApiClient()
const createKey = useIdempotencyKey()
const canCreateUpdateLog = useCan('update_log.create')

const {
  items: logs,
  hasMore,
  loadMore,
  loadingMore,
  status,
  problem,
  refresh
} = await useCursorList<UpdateLog>('update-logs', (api, cursor, { signal }) =>
  api.GET('/update-logs', {
    params: { query: cursor ? { cursor } : {} },
    signal
  })
)

const showModal = ref(false)
const editing = ref<UpdateLog | null>(null)
const isSubmitting = ref(false)

const initialData = computed<UpdateLogCreate | undefined>(() =>
  editing.value
    ? {
        change_type: editing.value.change_type,
        release_version: editing.value.release_version,
        text: editing.value.text
      }
    : undefined
)

const openCreate = () => {
  editing.value = null
  showModal.value = true
}

const openEdit = (log: UpdateLog) => {
  editing.value = log
  showModal.value = true
}

const submit = async (body: UpdateLogCreate) => {
  if (isSubmitting.value) {
    return
  }
  isSubmitting.value = true
  const target = editing.value
  const result = target
    ? await settle(
        api.PATCH('/update-logs/{update_log_id}', {
          params: { path: { update_log_id: target.id } },
          body
        })
      )
    : await settle(
        api.POST('/update-logs', {
          params: {
            header: { 'Idempotency-Key': createKey.take('/update-logs', body) }
          },
          body
        })
      )
  isSubmitting.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  if (!target) {
    createKey.clear()
  }
  useMessage(target ? '更新成功' : '发布更新日志成功', 'success')
  showModal.value = false
  await refresh()
}

const remove = async (log: UpdateLog) => {
  const ok = await useComponentMessageStore().alert(
    `确定删除 ${log.release_version} 的这条更新日志吗？`
  )
  if (!ok) {
    return
  }
  const result = await settle(
    api.DELETE('/update-logs/{update_log_id}', {
      params: { path: { update_log_id: log.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('更新日志已删除', 'success')
  await refresh()
}
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="更新日志"
      description="本页面记录了网站所有的更新日志, 新特性, BUG 修复, 功能更改, 性能优化等等"
    >
      <template #endContent>
        <div v-if="canCreateUpdateLog" class="flex justify-end">
          <KunButton @click="openCreate">创建更新日志</KunButton>
        </div>
      </template>
    </KunHeader>

    <div
      v-if="status === 'pending' && !logs.length"
      class="flex justify-center py-8"
    >
      <KunLoading description="正在加载更新日志..." />
    </div>

    <KunNull v-else-if="problem" :description="problemMessage(problem)" />

    <template v-else>
      <KunCard
        :is-hoverable="false"
        :is-transparent="false"
        v-for="log in logs"
        :key="log.id"
      >
        <div class="mb-3 flex items-center justify-between">
          <div class="flex items-center gap-3">
            <KunChip color="primary">
              {{ KUN_UPDATE_LOG_CHANGE_TYPE_LABEL[log.change_type] }}
            </KunChip>
            <span class="text-default-500 text-sm">
              <KunTime :time="log.created_at" type="date" show-year /> - Version
              {{ log.release_version }}
            </span>
          </div>

          <div class="flex items-center gap-2">
            <KunButton
              v-if="log.viewer?.can_edit"
              variant="flat"
              size="sm"
              @click="openEdit(log)"
            >
              编辑
            </KunButton>
            <KunButton
              v-if="log.viewer?.can_delete"
              variant="flat"
              size="sm"
              color="danger"
              @click="remove(log)"
            >
              删除
            </KunButton>
          </div>
        </div>
        <pre
          class="bg-default-100 rounded-md p-4 font-mono text-sm break-all whitespace-pre-line"
          >{{ log.text }}</pre
        >
      </KunCard>

      <div v-if="hasMore" class="text-center">
        <KunButton variant="flat" :loading="loadingMore" @click="loadMore">
          加载更多
        </KunButton>
      </div>
    </template>

    <UpdateHistoryModal
      v-model="showModal"
      :initial-data="initialData"
      :is-editing="!!editing"
      @submit="submit"
    />
  </div>
</template>
