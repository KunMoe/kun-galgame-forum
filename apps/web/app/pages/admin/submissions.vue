<script setup lang="ts">
definePageMeta({
  middleware: 'permission',
  permissions: ['galgame.claim.review']
})

useKunDisableSeo('Galgame 审核')

interface PendingClaim {
  gid: number
  name: string
  state: string
  updated?: string
}

interface PendingQueueEnvelope {
  items: PendingClaim[]
  next_cursor: string | null
}

const limit = 30

const items = ref<PendingClaim[]>([])
const nextCursor = ref('')
const isLoadingMore = ref(false)

const { data, refresh } = await useKunFetch<PendingQueueEnvelope>(
  '/admin/galgame/submissions',
  { query: { limit } }
)

watch(
  data,
  (page) => {
    if (!page) {
      return
    }
    items.value = page.items
    nextCursor.value = page.next_cursor ?? ''
  },
  { immediate: true }
)

const loadMore = async () => {
  if (isLoadingMore.value || !nextCursor.value) {
    return
  }
  isLoadingMore.value = true
  const next = await kunFetch<PendingQueueEnvelope>(
    '/admin/galgame/submissions',
    { query: { cursor: nextCursor.value, limit } }
  )
  isLoadingMore.value = false
  if (!next) {
    return
  }
  items.value.push(...next.items)
  nextCursor.value = next.next_cursor ?? ''
}

const displayName = (row: PendingClaim): string => row.name || `#${row.gid}`

const isActing = ref<Record<number, boolean>>({})

const previewWorkId = ref(0)
const previewState = ref('')
const isPreviewOpen = ref(false)

const openPreview = (row: PendingClaim) => {
  previewWorkId.value = row.gid
  previewState.value = row.state
  isPreviewOpen.value = true
}

type ReasonAction = 'decline' | 'ban'

interface ReasonContext {
  action: ReasonAction
  target: PendingClaim
}

const isReasonModalOpen = ref(false)
const reasonContext = ref<ReasonContext | null>(null)
const reasonText = ref('')

const modalTitle = computed(() => {
  if (!reasonContext.value) return ''
  const name = displayName(reasonContext.value.target)
  return reasonContext.value.action === 'decline'
    ? `拒绝《${name}》`
    : `封禁《${name}》`
})

const modalDescription = computed(() => {
  if (!reasonContext.value) return ''
  return reasonContext.value.action === 'decline'
    ? '请填写拒绝原因, 提交者会在站内消息中看到。'
    : '封禁后该条目对所有人不可见。如不填写理由, 仅记录管理员操作。'
})

const modalRequiresReason = computed(
  () => reasonContext.value?.action === 'decline'
)

const openReasonModal = (action: ReasonAction, target: PendingClaim) => {
  reasonContext.value = { action, target }
  reasonText.value = ''
  isReasonModalOpen.value = true
}

const closeReasonModal = () => {
  isReasonModalOpen.value = false
  setTimeout(() => {
    reasonContext.value = null
    reasonText.value = ''
  }, 300)
}

const applyVerdict = async (
  row: PendingClaim,
  action: 'approve' | 'decline' | 'ban',
  reason: string
) => {
  const workId = row.gid
  isActing.value = { ...isActing.value, [workId]: true }
  const res = await kunFetch<unknown>(`/admin/galgame/${workId}/review`, {
    method: 'POST',
    body: { action, reason }
  })
  isActing.value = { ...isActing.value, [workId]: false }
  if (res !== null) {
    useMessage('已处理', 'success')
    refresh()
  }
}

const handleApprove = async (row: PendingClaim) => {
  const ok = await useComponentMessageStore().alert(
    `通过《${displayName(row)}》?`,
    '通过后该 Galgame 立即公开发布, 提交者会收到通知并自动获得 +3 萌萌点 (由资料库事件同步发放)。'
  )
  if (!ok) return
  await applyVerdict(row, 'approve', '')
}

const handleConfirmReason = async () => {
  const ctx = reasonContext.value
  if (!ctx) return
  const trimmed = reasonText.value.trim()
  if (modalRequiresReason.value && !trimmed) {
    useMessage('拒绝原因不能为空', 'warn')
    return
  }
  closeReasonModal()
  await applyVerdict(ctx.target, ctx.action, trimmed)
}
</script>

<template>
  <div v-if="data" class="w-full space-y-4">
    <KunHeader
      name="Galgame 审核"
      description="审核用户提交的新 Galgame。通过后立即公开发布并向提交者发放 +3 萌萌点; 拒绝时需说明原因, 提交者可据此修改后重新提交。"
    />

    <KunDivider />

    <div v-if="items.length" class="flex flex-col gap-3">
      <div
        v-for="row in items"
        :key="row.gid"
        class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-start"
      >
        <div class="min-w-0 flex-1 space-y-1">
          <div class="flex flex-wrap items-center gap-2">
            <h3 class="truncate text-lg font-medium">
              {{ displayName(row) }}
            </h3>
            <KunChip
              size="xs"
              variant="flat"
              :color="galgameClaimStateBadge(row.state).color"
            >
              {{ galgameClaimStateBadge(row.state).label }}
            </KunChip>
          </div>
          <div
            class="text-default-500 flex flex-wrap items-center gap-2 text-sm"
          >
            <span v-if="row.updated"><KunTime :time="row.updated" /></span>
            <span v-if="row.updated">·</span>
            <span>galgame_id: {{ row.gid }}</span>
          </div>
        </div>
        <div class="flex shrink-0 flex-wrap gap-2">
          <KunButton size="sm" variant="flat" @click="openPreview(row)">
            预览
          </KunButton>
          <KunButton
            size="sm"
            color="success"
            :loading="isActing[row.gid]"
            :disabled="isActing[row.gid]"
            @click="handleApprove(row)"
          >
            通过
          </KunButton>
          <KunButton
            size="sm"
            color="danger"
            variant="flat"
            :loading="isActing[row.gid]"
            :disabled="isActing[row.gid]"
            @click="openReasonModal('decline', row)"
          >
            拒绝
          </KunButton>
          <KunButton
            size="sm"
            color="default"
            variant="light"
            :loading="isActing[row.gid]"
            :disabled="isActing[row.gid]"
            @click="openReasonModal('ban', row)"
          >
            封禁
          </KunButton>
        </div>
      </div>
    </div>

    <KunNull v-if="!items.length" />

    <KunButton
      v-if="nextCursor"
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
    />

    <KunModal
      :model-value="isReasonModalOpen"
      inner-class-name="w-full max-w-lg"
      :is-dismissable="false"
      @update:model-value="closeReasonModal"
    >
      <div class="space-y-4">
        <h3 class="text-xl font-medium">{{ modalTitle }}</h3>
        <p class="text-default-500 text-sm">{{ modalDescription }}</p>

        <KunTextarea
          v-model="reasonText"
          :placeholder="
            modalRequiresReason ? '请填写拒绝原因 (必填)' : '可选填理由'
          "
          :rows="4"
          :maxlength="1007"
          show-char-count
          :required="modalRequiresReason"
        />

        <div class="flex justify-end gap-2">
          <KunButton variant="light" @click="closeReasonModal">取消</KunButton>
          <KunButton
            :color="reasonContext?.action === 'decline' ? 'danger' : 'default'"
            @click="handleConfirmReason"
          >
            确认
          </KunButton>
        </div>
      </div>
    </KunModal>
  </div>
</template>
