<script setup lang="ts">
const { data, items, hasMore, isLoadingMore, loadMore, refresh } =
  await useGalgameClaimList('/galgame/mine')

const isWithdrawing = ref<Record<number, boolean>>({})

const handleWithdraw = async (item: UserClaimItem) => {
  const ok = await useComponentMessageStore().alert(
    '确定撤回这条申请吗?',
    '撤回后该条目会退回草稿状态, 不再公开展示。您填写的内容不会丢失, 随时可以重新提交。'
  )
  if (!ok) {
    return
  }
  const workId = item.work_id
  isWithdrawing.value = { ...isWithdrawing.value, [workId]: true }
  const res = await kunFetch<string>(`/galgame/${workId}`, {
    method: 'DELETE'
  })
  isWithdrawing.value = { ...isWithdrawing.value, [workId]: false }
  if (res !== null) {
    useMessage('已撤回', 'success')
    refresh()
  }
}

const previewWorkId = ref(0)
const previewState = ref('')
const previewReason = ref('')
const isPreviewOpen = ref(false)

const openPreview = (item: UserClaimItem) => {
  previewWorkId.value = item.work_id
  previewState.value = item.claim_state
  previewReason.value =
    item.claim_state === CLAIM_STATE_DECLINED ? (item.last_reason ?? '') : ''
  isPreviewOpen.value = true
}

const isResubmitting = ref<Record<number, boolean>>({})

const handleResubmit = async (item: UserClaimItem) => {
  const workId = item.work_id
  isResubmitting.value = { ...isResubmitting.value, [workId]: true }
  const res = await kunFetch<unknown>(`/galgame/${workId}/resubmit`, {
    method: 'POST'
  })
  isResubmitting.value = { ...isResubmitting.value, [workId]: false }
  if (res !== null) {
    useMessage('已重新提交审核', 'success')
    refresh()
  }
}

const isDeleting = ref<Record<number, boolean>>({})

const handleDelete = async (item: UserClaimItem) => {
  const ok = await useComponentMessageStore().alert(
    '确定删除这条草稿吗?',
    '草稿会被彻底移除, 无法恢复。已经发布过资源的条目不会被删除。'
  )
  if (!ok) {
    return
  }
  const workId = item.work_id
  isDeleting.value = { ...isDeleting.value, [workId]: true }
  const res = await kunFetch<string>(`/galgame/${workId}/draft`, {
    method: 'DELETE'
  })
  isDeleting.value = { ...isDeleting.value, [workId]: false }
  if (res !== null) {
    useMessage('已删除', 'success')
    refresh()
  }
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
      v-if="!data"
      color="danger"
      title="加载失败"
      description="无法获取您的提交列表, 可能是后端 / Galgame 资料库暂时不可用, 请稍后重试。"
    />

    <div v-else-if="items.length" class="flex flex-col gap-3">
      <EditGalgameClaimRow
        v-for="item in items"
        :key="item.work_id"
        :item="item"
        time-label="提交于"
      >
        <template #note>
          <div
            v-if="item.claim_state === CLAIM_STATE_DECLINED && item.last_reason"
            class="text-danger bg-danger/10 mt-1 rounded-md px-2 py-1 text-sm"
          >
            被拒原因: {{ item.last_reason }}
          </div>
        </template>

        <template #actions>
          <template v-if="item.work_id">
            <KunButton size="sm" variant="flat" @click="openPreview(item)">
              预览
            </KunButton>
            <KunLink :to="`/galgame/${item.work_id}/edit`">
              <KunButton size="sm" variant="flat">编辑</KunButton>
            </KunLink>
            <KunButton
              v-if="item.claim_state === CLAIM_STATE_DECLINED"
              size="sm"
              color="primary"
              variant="flat"
              :loading="isResubmitting[item.work_id]"
              :disabled="isResubmitting[item.work_id]"
              @click="handleResubmit(item)"
            >
              重新提交
            </KunButton>
            <KunButton
              v-else-if="item.claim_state !== CLAIM_STATE_DRAFT"
              size="sm"
              color="danger"
              variant="flat"
              :loading="isWithdrawing[item.work_id]"
              :disabled="isWithdrawing[item.work_id]"
              @click="handleWithdraw(item)"
            >
              撤回
            </KunButton>
            <KunButton
              v-if="item.claim_state === CLAIM_STATE_DRAFT"
              size="sm"
              color="danger"
              variant="flat"
              :loading="isDeleting[item.work_id]"
              :disabled="isDeleting[item.work_id]"
              @click="handleDelete(item)"
            >
              删除
            </KunButton>
          </template>
        </template>
      </EditGalgameClaimRow>
    </div>

    <KunNull v-else />

    <KunButton
      v-if="hasMore"
      variant="flat"
      :loading="isLoadingMore"
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
