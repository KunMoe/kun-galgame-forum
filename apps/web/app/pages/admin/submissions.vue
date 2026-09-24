<script setup lang="ts">
import type { WorkSubmissionSummary } from '#shared/utils/api/schemas'

definePageMeta({
  middleware: 'permission',
  permissions: ['galgame.claim.review']
})

useKunDisableSeo('Galgame 审核')

const limit = 30

const { items, hasMore, loadingMore, loadMore, problem, refresh } =
  await useCursorList<WorkSubmissionSummary>(
    'work-submissions:queue',
    (api, cursor, { signal }) =>
      api.GET('/work-submissions', {
        params: { query: { state: ['pending'], limit, cursor } },
        signal
      })
  )
const { move } = useWorkSubmission()

const displayName = (row: WorkSubmissionSummary): string =>
  row.display_name || `#${row.id}`

const isActing = ref<Record<string, boolean>>({})

const previewWorkId = ref(0)
const previewState = ref('')
const isPreviewOpen = ref(false)

const openPreview = (row: WorkSubmissionSummary) => {
  previewWorkId.value = Number(row.id)
  previewState.value = row.state
  isPreviewOpen.value = true
}

type ReasonAction = 'declined' | 'hidden'

interface ReasonContext {
  action: ReasonAction
  target: WorkSubmissionSummary
}

const isReasonModalOpen = ref(false)
const reasonContext = ref<ReasonContext | null>(null)
const reasonText = ref('')

const modalTitle = computed(() => {
  if (!reasonContext.value) return ''
  const name = displayName(reasonContext.value.target)
  return reasonContext.value.action === 'declined'
    ? `拒绝《${name}》`
    : `封禁《${name}》`
})

const modalDescription = computed(() => {
  if (!reasonContext.value) return ''
  return reasonContext.value.action === 'declined'
    ? '请填写拒绝原因, 提交者会在站内消息中看到。'
    : '封禁后该条目对所有人不可见。如不填写理由, 仅记录管理员操作。'
})

const modalRequiresReason = computed(
  () => reasonContext.value?.action === 'declined'
)

const openReasonModal = (
  action: ReasonAction,
  target: WorkSubmissionSummary
) => {
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
  row: WorkSubmissionSummary,
  state: 'live' | ReasonAction,
  note: string
) => {
  isActing.value = { ...isActing.value, [row.id]: true }
  const result = await move(row.id, state, note)
  isActing.value = { ...isActing.value, [row.id]: false }
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('已处理', 'success')
  refresh()
}

const handleApprove = async (row: WorkSubmissionSummary) => {
  const ok = await useComponentMessageStore().alert(
    `通过《${displayName(row)}》?`,
    '通过后该 Galgame 立即公开发布, 提交者会收到通知并自动获得 +3 萌萌点 (由资料库事件同步发放)。'
  )
  if (!ok) return
  await applyVerdict(row, 'live', '')
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
  <div class="w-full space-y-4">
    <KunHeader
      name="Galgame 审核"
      description="审核用户提交的新 Galgame。通过后立即公开发布并向提交者发放 +3 萌萌点; 拒绝时需说明原因, 提交者可据此修改后重新提交。"
    />

    <KunDivider />

    <KunInfo
      v-if="problem"
      color="danger"
      title="加载失败"
      description="无法获取审核队列, 请稍后重试。"
    />

    <div v-else-if="items.length" class="flex flex-col gap-3">
      <div
        v-for="row in items"
        :key="row.id"
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
          <div class="text-default-500 text-sm">work_id: {{ row.id }}</div>
        </div>
        <div class="flex shrink-0 flex-wrap gap-2">
          <KunButton size="sm" variant="flat" @click="openPreview(row)">
            预览
          </KunButton>
          <KunButton
            size="sm"
            color="success"
            :loading="isActing[row.id]"
            :disabled="isActing[row.id]"
            @click="handleApprove(row)"
          >
            通过
          </KunButton>
          <KunButton
            size="sm"
            color="danger"
            variant="flat"
            :loading="isActing[row.id]"
            :disabled="isActing[row.id]"
            @click="openReasonModal('declined', row)"
          >
            拒绝
          </KunButton>
          <KunButton
            size="sm"
            color="default"
            variant="light"
            :loading="isActing[row.id]"
            :disabled="isActing[row.id]"
            @click="openReasonModal('hidden', row)"
          >
            封禁
          </KunButton>
        </div>
      </div>
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
            :color="reasonContext?.action === 'declined' ? 'danger' : 'default'"
            @click="handleConfirmReason"
          >
            确认
          </KunButton>
        </div>
      </div>
    </KunModal>
  </div>
</template>
