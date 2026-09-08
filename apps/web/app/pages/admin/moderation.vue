<script setup lang="ts">
import {
  TRUST_REVIEW_STATUS,
  TRUST_REVIEW_SOURCE,
  TRUST_ACTIONS,
  TRUST_SUBJECT_KIND,
  trustSubjectHref
} from '~/constants/trust'

definePageMeta({
  middleware: 'permission',
  permissions: ['trust.review']
})

useKunDisableSeo('内容审核')

const statusTabs = [
  { value: '0', textValue: '待处理' },
  { value: '1', textValue: '处理中' },
  { value: '2', textValue: '已处置' },
  { value: '3', textValue: '已驳回' },
  { value: '-1', textValue: '全部' }
]
const activeStatus = ref('0')

const pageData = reactive({ status: 0, page: 1, limit: 30 })
watch(activeStatus, (v) => {
  pageData.status = Number(v)
  pageData.page = 1
})

const { data, status, refresh } = await useKunFetch<ReviewItemPage>(
  '/admin/trust/review-items',
  { query: pageData }
)

const kindLabel = (k: string) => TRUST_SUBJECT_KIND[k] ?? k

const isDetailOpen = ref(false)
const detail = ref<ReviewItemDetail | null>(null)
const detailLoading = ref(false)

const decision = ref<'actioned' | 'dismissed'>('actioned')
const action = ref(1)
const reasonCode = ref('')
const statement = ref('')
const isWorking = ref(false)

const openDetail = async (id: number) => {
  isDetailOpen.value = true
  detailLoading.value = true
  detail.value = null
  decision.value = 'actioned'
  action.value = 1
  reasonCode.value = ''
  statement.value = ''
  detail.value =
    (await kunFetch<ReviewItemDetail>(`/admin/trust/review-items/${id}`)) ??
    null
  detailLoading.value = false
}

const isOpen = computed(() => {
  const s = detail.value?.item.status
  return s === 0 || s === 1
})

const subjectHref = computed(() => {
  if (!detail.value) {
    return undefined
  }
  const fromReport = detail.value.reports.find(
    (r) => r.subject_url
  )?.subject_url
  return (
    fromReport ||
    trustSubjectHref(
      detail.value.item.subject_kind,
      detail.value.item.subject_id
    )
  )
})

const claim = async () => {
  if (!detail.value) return
  isWorking.value = true
  const ok = await kunFetch(
    `/admin/trust/review-items/${detail.value.item.id}/claim`,
    { method: 'POST' }
  )
  isWorking.value = false
  if (ok) {
    useMessage('已认领', 'success')
    await openDetail(detail.value.item.id)
    refresh()
  }
}

const decide = async () => {
  if (!detail.value) return
  if (decision.value === 'actioned' && !reasonCode.value.trim()) {
    useMessage('请填写处置理由代码（reason_code）', 'warn')
    return
  }
  isWorking.value = true
  const body: Record<string, unknown> = { decision: decision.value }
  if (decision.value === 'actioned') {
    body.action = action.value
    body.reason_code = reasonCode.value.trim()
    if (statement.value.trim()) body.statement = statement.value.trim()
  }
  const ok = await kunFetch(
    `/admin/trust/review-items/${detail.value.item.id}/decide`,
    { method: 'POST', body }
  )
  isWorking.value = false
  if (ok) {
    useMessage('已处置', 'success')
    isDetailOpen.value = false
    refresh()
  }
}

const actionOptions = TRUST_ACTIONS.map((a) => ({
  value: a.value as number,
  label: a.label
}))
</script>

<template>
  <div class="space-y-4">
    <div>
      <h2 class="text-xl font-bold">内容审核</h2>
      <p class="text-default-500 text-sm">
        来自 Trust &amp; Safety 平台的统一审核队列，按优先级排序。
      </p>
    </div>

    <KunTab
      v-model="activeStatus"
      :items="statusTabs"
      variant="underlined"
      color="primary"
      size="sm"
    />

    <KunLoading v-if="status === 'pending'" />

    <KunNull v-else-if="!data?.items.length" description="暂无审核条目" />

    <div v-else class="space-y-2">
      <button
        v-for="item in data.items"
        :key="item.id"
        class="hover:bg-default-100 border-default-200 w-full rounded-lg border p-3 text-left transition-colors"
        @click="openDetail(item.id)"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex flex-wrap items-center gap-2">
            <KunChip color="secondary">{{
              kindLabel(item.subject_kind)
            }}</KunChip>
            <span class="text-default-500 text-sm">#{{ item.subject_id }}</span>
            <KunChip
              :color="TRUST_REVIEW_STATUS[item.status]?.color ?? 'default'"
            >
              {{ TRUST_REVIEW_STATUS[item.status]?.label ?? item.status }}
            </KunChip>
          </div>
          <span class="text-default-400 shrink-0 text-xs">
            优先级 {{ item.priority.toFixed(1) }}
          </span>
        </div>
        <div
          class="text-default-400 mt-1 flex flex-wrap items-center gap-x-3 text-xs"
        >
          <span
            >来源：{{ TRUST_REVIEW_SOURCE[item.source] ?? item.source }}</span
          >
          <span v-if="item.report_weight_sum"
            >举报权重 {{ item.report_weight_sum.toFixed(1) }}</span
          >
          <KunTime :time="item.created_at" type="datetime" show-year />
        </div>
      </button>
    </div>

    <KunPagination
      v-if="data?.total"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />

    <KunModal v-model="isDetailOpen" inner-class-name="max-w-2xl w-[94vw]">
      <KunLoading v-if="detailLoading" />
      <div
        v-else-if="detail"
        class="max-h-[80dvh] space-y-4 overflow-y-auto p-1"
      >
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-lg font-bold">审核详情</span>
          <KunChip color="secondary">{{
            kindLabel(detail.item.subject_kind)
          }}</KunChip>
          <KunLink
            v-if="subjectHref"
            :to="subjectHref"
            target="_blank"
            class="text-primary text-sm"
          >
            查看内容 #{{ detail.item.subject_id }}
          </KunLink>
          <span v-else class="text-default-500 text-sm"
            >#{{ detail.item.subject_id }}</span
          >
        </div>

        <div class="space-y-2">
          <div
            class="text-default-400 flex flex-wrap items-center gap-x-3 text-xs"
          >
            <span>
              来源：{{
                TRUST_REVIEW_SOURCE[detail.item.source] ?? detail.item.source
              }}
            </span>
            <span v-if="detail.item.severity != null">
              严重度 {{ detail.item.severity }}
            </span>
            <span v-if="detail.item.classifier_score != null">
              分类器 {{ detail.item.classifier_score.toFixed(2) }}
            </span>
            <span v-if="detail.item.subject_reach != null">
              已触达 {{ detail.item.subject_reach }}
            </span>
            <span v-if="detail.item.report_weight_sum">
              举报权重 {{ detail.item.report_weight_sum.toFixed(1) }}
            </span>
          </div>

          <!-- 420 of the 430 review items on prod are ai_text / community_forward:
               they carry no trust_report rows at all and put their evidence in
               context_note. Rendering only `reports` showed 「举报记录（0）」 and
               nothing else for 97.7% of the queue. -->
          <div v-if="detail.item.context_note" class="space-y-1">
            <span class="text-default-600 text-sm font-medium">判定依据</span>
            <p
              class="bg-default-100 text-default-700 rounded-lg p-2 text-sm whitespace-pre-wrap"
            >
              {{ detail.item.context_note }}
            </p>
          </div>
        </div>

        <div class="space-y-2">
          <span class="text-default-600 text-sm font-medium">
            举报记录（{{ detail.reports.length }}）
          </span>
          <p v-if="!detail.reports.length" class="text-default-400 text-sm">
            该条目不是由用户举报产生的{{
              detail.item.context_note
                ? '，依据见上方'
                : '，且上游没有给出依据摘要'
            }}
          </p>
          <div
            v-for="r in detail.reports"
            :key="r.id"
            class="bg-default-100 space-y-1 rounded-lg p-2 text-sm"
          >
            <div class="text-default-400 flex flex-wrap gap-x-3 text-xs">
              <span>举报人 #{{ r.reporter_id }}</span>
              <span>理由 #{{ r.reason_id }}</span>
              <span>权重 {{ r.weight.toFixed(1) }}</span>
              <KunTime :time="r.created_at" type="datetime" show-year />
            </div>
            <p v-if="r.note" class="text-default-700">{{ r.note }}</p>
            <pre
              v-if="r.subject_snapshot"
              class="text-default-500 border-default-200 max-h-32 overflow-y-auto rounded border p-2 text-xs whitespace-pre-wrap"
              >{{ r.subject_snapshot }}</pre
            >
          </div>
        </div>

        <template v-if="isOpen">
          <KunDivider />
          <div class="space-y-3">
            <div class="flex items-center gap-2">
              <KunButton
                v-if="detail.item.status === 0"
                variant="flat"
                color="primary"
                :loading="isWorking"
                @click="claim"
              >
                认领
              </KunButton>
              <span v-else class="text-default-500 text-sm">
                处理中（认领人 #{{ detail.item.claimed_by }}）
              </span>
            </div>

            <div class="flex gap-2">
              <KunButton
                :variant="decision === 'actioned' ? 'flat' : 'light'"
                color="danger"
                size="sm"
                @click="decision = 'actioned'"
              >
                处置
              </KunButton>
              <KunButton
                :variant="decision === 'dismissed' ? 'flat' : 'light'"
                color="default"
                size="sm"
                @click="decision = 'dismissed'"
              >
                驳回
              </KunButton>
            </div>

            <template v-if="decision === 'actioned'">
              <KunSelect
                v-model="action"
                :options="actionOptions"
                label="处置动作"
              />
              <KunInput
                v-model="reasonCode"
                label="处置理由代码 (reason_code)"
                placeholder="例如 spam / abuse"
              />
              <KunTextarea
                name="statement"
                v-model="statement"
                placeholder="面向用户的处置说明（可选）"
                :rows="3"
              />
            </template>

            <div class="flex justify-end">
              <KunButton color="danger" :loading="isWorking" @click="decide">
                确认{{ decision === 'actioned' ? '处置' : '驳回' }}
              </KunButton>
            </div>
          </div>
        </template>
        <p v-else class="text-default-500 text-sm">
          该条目已终结（{{ TRUST_REVIEW_STATUS[detail.item.status]?.label }}）。
        </p>
      </div>
    </KunModal>
  </div>
</template>
