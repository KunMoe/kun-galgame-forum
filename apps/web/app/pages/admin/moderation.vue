<script setup lang="ts">
import {
  TRUST_REVIEW_STATE,
  TRUST_REVIEW_ORIGIN,
  TRUST_ACTIONS,
  TRUST_SUBJECT_KIND,
  trustSubjectHref,
  type DispositionAction,
  type ReviewState
} from '~/constants/trust'
import { settle } from '#shared/utils/api/problem'
import type { ReviewItem, ReviewItemPatch } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

definePageMeta({
  middleware: 'permission',
  permissions: ['trust.review']
})

useKunDisableSeo('内容审核')

const statusTabs = [
  { value: 'pending', textValue: '待处理' },
  { value: 'claimed', textValue: '处理中' },
  { value: 'actioned', textValue: '已处置' },
  { value: 'dismissed', textValue: '已驳回' },
  { value: 'all', textValue: '全部' }
]
const activeState = ref('pending')

const pageData = reactive({
  state: 'pending' as ReviewState | '',
  page: 1,
  limit: 30
})
watch(activeState, (v) => {
  pageData.state = v === 'all' ? '' : (v as ReviewState)
  pageData.page = 1
})

const api = useApiClient()

const { data, status, refresh } = await useApi(
  () =>
    `admin-review-items:${pageData.state}:${pageData.page}:${pageData.limit}`,
  (client, { signal }) =>
    client.GET('/admin/review-items', {
      params: {
        query: {
          page: pageData.page,
          limit: pageData.limit,
          ...(pageData.state ? { state: pageData.state } : {})
        }
      },
      signal
    })
)

const items = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)

const kindLabel = (k: string) => TRUST_SUBJECT_KIND[k] ?? k
const originLabel = (o: string) => TRUST_REVIEW_ORIGIN[o] ?? o

const { reasons, load: loadReasons } = useReportReasons()
const reasonOptions = computed(() =>
  reasons.value.map((r) => ({ value: r.key, label: r.display_name }))
)

const isDetailOpen = ref(false)
const detail = ref<ReviewItem | null>(null)
const detailLoading = ref(false)

const decision = ref<'actioned' | 'dismissed'>('actioned')
const action = ref<DispositionAction>('hide')
const reasonCode = ref('')
const statement = ref('')
const isWorking = ref(false)

const openDetail = async (id: string) => {
  isDetailOpen.value = true
  detailLoading.value = true
  detail.value = null
  decision.value = 'actioned'
  action.value = 'hide'
  reasonCode.value = ''
  statement.value = ''
  loadReasons()
  const result = await settle(
    api.GET('/admin/review-items/{review_item_id}', {
      params: { path: { review_item_id: id } }
    })
  )
  detailLoading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    isDetailOpen.value = false
    return
  }
  detail.value = result.data
}

const isOpen = computed(() => {
  const s = detail.value?.state
  return s === 'pending' || s === 'claimed'
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
    trustSubjectHref(detail.value.subject_kind, detail.value.subject_id)
  )
})

const patchItem = async (body: ReviewItemPatch) => {
  if (!detail.value) {
    return false
  }
  isWorking.value = true
  const result = await settle(
    api.PATCH('/admin/review-items/{review_item_id}', {
      params: { path: { review_item_id: detail.value.id } },
      body
    })
  )
  isWorking.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  detail.value = result.data
  refresh()
  return true
}

const claim = async () => {
  if (await patchItem({ state: 'claimed' })) {
    useMessage('已认领', 'success')
  }
}

const decide = async () => {
  if (decision.value === 'actioned' && !reasonCode.value) {
    useMessage('请选择处置理由', 'warn')
    return
  }
  const body: ReviewItemPatch =
    decision.value === 'actioned'
      ? {
          state: 'actioned',
          action: action.value,
          reason_code: reasonCode.value,
          ...(statement.value.trim()
            ? { statement: statement.value.trim() }
            : {})
        }
      : { state: 'dismissed' }
  if (await patchItem(body)) {
    useMessage('已处置', 'success')
    isDetailOpen.value = false
  }
}

const actionOptions = TRUST_ACTIONS.map((a) => ({
  value: a.value,
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
      v-model="activeState"
      :items="statusTabs"
      variant="underlined"
      color="primary"
      size="sm"
    />

    <KunLoading v-if="status === 'pending'" />

    <KunNull v-else-if="!items.length" description="暂无审核条目" />

    <div v-else class="space-y-2">
      <button
        v-for="item in items"
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
            <KunChip :color="TRUST_REVIEW_STATE[item.state].color">
              {{ TRUST_REVIEW_STATE[item.state].label }}
            </KunChip>
          </div>
          <span class="text-default-400 shrink-0 text-xs">
            优先级 {{ item.priority.toFixed(1) }}
          </span>
        </div>
        <div
          class="text-default-400 mt-1 flex flex-wrap items-center gap-x-3 text-xs"
        >
          <span>来源：{{ originLabel(item.opened_by) }}</span>
          <span v-if="item.report_weight_sum"
            >举报权重 {{ item.report_weight_sum.toFixed(1) }}</span
          >
          <KunTime :time="item.created_at" type="datetime" show-year />
        </div>
      </button>
    </div>

    <KunPagination
      v-if="total > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(total / pageData.limit)"
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
            kindLabel(detail.subject_kind)
          }}</KunChip>
          <KunLink
            v-if="subjectHref"
            :to="subjectHref"
            target="_blank"
            class="text-primary text-sm"
          >
            查看内容 #{{ detail.subject_id }}
          </KunLink>
          <span v-else class="text-default-500 text-sm"
            >#{{ detail.subject_id }}</span
          >
        </div>

        <div class="space-y-2">
          <div
            class="text-default-400 flex flex-wrap items-center gap-x-3 text-xs"
          >
            <span> 来源：{{ originLabel(detail.opened_by) }} </span>
            <span v-if="detail.severity != null">
              严重度 {{ detail.severity }}
            </span>
            <span v-if="detail.classifier_score != null">
              分类器 {{ detail.classifier_score.toFixed(2) }}
            </span>
            <span v-if="detail.reach_count != null">
              已触达 {{ detail.reach_count }}
            </span>
            <span v-if="detail.report_weight_sum">
              举报权重 {{ detail.report_weight_sum.toFixed(1) }}
            </span>
          </div>

          <!-- 420 of the 430 review items on prod are ai_text / community_forward:
               they carry no trust_report rows at all and put their evidence in
               context_note. Rendering only `reports` showed 「举报记录（0）」 and
               nothing else for 97.7% of the queue. -->
          <div v-if="detail.context_note" class="space-y-1">
            <span class="text-default-600 text-sm font-medium">判定依据</span>
            <p
              class="bg-default-100 text-default-700 rounded-lg p-2 text-sm whitespace-pre-wrap"
            >
              {{ detail.context_note }}
            </p>
          </div>
        </div>

        <div class="space-y-2">
          <span class="text-default-600 text-sm font-medium">
            举报记录（{{ detail.reports.length }}）
          </span>
          <p v-if="!detail.reports.length" class="text-default-400 text-sm">
            该条目不是由用户举报产生的{{
              detail.context_note ? '，依据见上方' : '，且上游没有给出依据摘要'
            }}
          </p>
          <div
            v-for="r in detail.reports"
            :key="r.id"
            class="bg-default-100 space-y-1 rounded-lg p-2 text-sm"
          >
            <div class="text-default-400 flex flex-wrap gap-x-3 text-xs">
              <UserHoverCard :user-id="r.reporter.id">
                <KunUserChip :user="toKunUser(r.reporter)" size="xs" />
              </UserHoverCard>
              <span
                >理由：{{
                  r.report_reason?.display_name ?? '已停用的理由'
                }}</span
              >
              <span>权重 {{ r.weight.toFixed(1) }}</span>
              <KunTime :time="r.created_at" type="datetime" show-year />
            </div>
            <p v-if="r.note" class="text-default-700">{{ r.note }}</p>
            <pre
              v-if="r.snapshot"
              class="text-default-500 border-default-200 max-h-32 overflow-y-auto rounded border p-2 text-xs whitespace-pre-wrap"
              >{{ r.snapshot }}</pre
            >
          </div>
        </div>

        <template v-if="isOpen">
          <KunDivider />
          <div class="space-y-3">
            <div class="flex items-center gap-2">
              <KunButton
                v-if="detail.state === 'pending'"
                variant="flat"
                color="primary"
                :loading="isWorking"
                @click="claim"
              >
                认领
              </KunButton>
              <span
                v-else-if="detail.claimant"
                class="text-default-500 flex items-center gap-1 text-sm"
              >
                处理中，认领人
                <UserHoverCard :user-id="detail.claimant.id">
                  <KunUserChip :user="toKunUser(detail.claimant)" size="xs" />
                </UserHoverCard>
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
              <KunSelect
                v-model="reasonCode"
                :options="reasonOptions"
                label="处置理由"
                placeholder="请选择处置理由"
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
          该条目已终结（{{ TRUST_REVIEW_STATE[detail.state].label }}）。
        </p>
      </div>
    </KunModal>
  </div>
</template>
