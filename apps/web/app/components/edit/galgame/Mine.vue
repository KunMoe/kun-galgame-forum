<script setup lang="ts">
import type { ApiResult } from '#shared/utils/api/problem'
import type { WorkSubmissionSummary } from '#shared/utils/api/schemas'

const { items, hasMore, loadingMore, loadMore, problem, refresh } =
  await useGalgameClaimList('mine')
const { move, remove } = useWorkSubmission()

const busy = ref<Record<string, boolean>>({})

const act = async (
  item: WorkSubmissionSummary,
  run: () => Promise<ApiResult<unknown>>,
  done: string
) => {
  busy.value = { ...busy.value, [item.id]: true }
  const result = await run()
  busy.value = { ...busy.value, [item.id]: false }
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(done, 'success')
  refresh()
}

const handleWithdraw = async (item: WorkSubmissionSummary) => {
  const ok = await useComponentMessageStore().alert(
    '确定撤回这条申请吗?',
    '撤回后该条目会退回草稿状态, 不再公开展示。您填写的内容不会丢失, 随时可以重新提交。'
  )
  if (ok) {
    await act(item, () => move(item.id, 'draft'), '已撤回')
  }
}

const handleSubmit = (item: WorkSubmissionSummary) =>
  act(
    item,
    () => move(item.id, 'pending'),
    item.state === CLAIM_STATE_DECLINED ? '已重新提交审核' : '已提交审核'
  )

const handleDelete = async (item: WorkSubmissionSummary) => {
  const ok = await useComponentMessageStore().alert(
    '确定删除这条草稿吗?',
    '草稿会被彻底移除, 无法恢复。已经发布过资源的条目不会被删除。'
  )
  if (ok) {
    await act(item, () => remove(item.id), '已删除')
  }
}

const previewWorkId = ref(0)
const previewState = ref('')
const previewReason = ref('')
const isPreviewOpen = ref(false)

const openPreview = (item: WorkSubmissionSummary) => {
  previewWorkId.value = Number(item.id)
  previewState.value = item.state
  previewReason.value =
    item.state === CLAIM_STATE_DECLINED ? (item.last_event?.note ?? '') : ''
  isPreviewOpen.value = true
}
</script>

<template>
  <div class="space-y-4">
    <!-- 铁律 #11: keep this template's SINGLE real root element. A sibling
         or a leading comment at the template root is itself a root node,
         trips Nuxt's "does not have a single root node" warning, and
         silently drops the page-transition enter animation. -->
    <KunHeader
      name="我的 Galgame 提交"
      description="您提交的 Galgame 申请, 审核中 / 已拒绝 / 草稿都会显示在此处。审核通过的 Galgame 会成为公开条目, 不再列在这里。"
    >
      <template #endContent>
        <div class="flex gap-2">
          <KunLink to="/edit/galgame/publish">
            <KunButton size="sm">新建提交</KunButton>
          </KunLink>
          <KunLink to="/message/notice">
            <KunButton size="sm" variant="flat">审核通知</KunButton>
          </KunLink>
        </div>
      </template>
    </KunHeader>

    <KunDivider />

    <KunInfo
      v-if="problem"
      color="danger"
      title="加载失败"
      description="无法获取您的提交列表, 可能是后端 / Galgame 资料库暂时不可用, 请稍后重试。"
    />

    <div v-else-if="items.length" class="flex flex-col gap-3">
      <EditGalgameClaimRow
        v-for="item in items"
        :key="item.id"
        :item="item"
        time-label="提交于"
      >
        <template #note>
          <div
            v-if="item.state === CLAIM_STATE_DECLINED && item.last_event?.note"
            class="text-danger bg-danger/10 mt-1 rounded-md px-2 py-1 text-sm"
          >
            被拒原因: {{ item.last_event.note }}
          </div>
        </template>

        <template #actions>
          <KunButton size="sm" variant="flat" @click="openPreview(item)">
            预览
          </KunButton>
          <KunLink :to="`/galgame/${item.id}/edit`">
            <KunButton size="sm" variant="flat">编辑</KunButton>
          </KunLink>
          <KunButton
            v-if="item.viewer?.can_submit"
            size="sm"
            color="primary"
            variant="flat"
            :loading="busy[item.id]"
            :disabled="busy[item.id]"
            @click="handleSubmit(item)"
          >
            {{ item.state === CLAIM_STATE_DECLINED ? '重新提交' : '提交审核' }}
          </KunButton>
          <KunButton
            v-if="item.viewer?.can_withdraw"
            size="sm"
            color="danger"
            variant="flat"
            :loading="busy[item.id]"
            :disabled="busy[item.id]"
            @click="handleWithdraw(item)"
          >
            撤回
          </KunButton>
          <KunButton
            v-if="item.viewer?.can_delete"
            size="sm"
            color="danger"
            variant="flat"
            :loading="busy[item.id]"
            :disabled="busy[item.id]"
            @click="handleDelete(item)"
          >
            删除
          </KunButton>
        </template>
      </EditGalgameClaimRow>
    </div>

    <KunNull v-else />

    <KunButton
      v-if="hasMore"
      variant="flat"
      :loading="loadingMore"
      @click="loadMore"
    >
      加载更多
    </KunButton>

    <GalgamePreviewModal
      v-if="previewWorkId"
      v-model="isPreviewOpen"
      :work-id="previewWorkId"
      :claim-state="previewState"
      :decline-reason="previewReason"
    />
  </div>
</template>
