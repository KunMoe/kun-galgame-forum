<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { CreatorStatus } from '#shared/utils/api/schemas'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'

const api = useApiClient()
const applyKey = useIdempotencyKey()

const { showKUNGalgameCreatorApply: isOpen } = storeToRefs(
  useTempSettingStore()
)

const status = ref<CreatorStatus | null>(null)
const statement = ref('')
const loading = ref(false)
const failed = ref(false)
const submitting = ref(false)

const load = async () => {
  loading.value = true
  failed.value = false
  const result = await settle(api.GET('/me/creator-status'))
  if (result.ok) {
    status.value = result.data
  } else {
    reportProblem(result.problem)
    failed.value = true
  }
  loading.value = false
}

watch(isOpen, (open) => {
  if (open) {
    statement.value = ''
    load()
  }
})

const eligibility = computed(() => status.value?.eligibility ?? null)
const application = computed(() => status.value?.application ?? null)
const isCreator = computed(
  () => !!status.value?.is_creator || application.value?.state === 'approved'
)
const isPending = computed(() => application.value?.state === 'pending')
const isDeclined = computed(() => application.value?.state === 'declined')
const isEligible = computed(() => !!eligibility.value?.is_eligible)
const canApply = computed(
  () => isEligible.value && !isPending.value && !isCreator.value
)

const FLOW = [
  { title: '达成条件', icon: 'lucide:target' },
  { title: '提交申请', icon: 'lucide:send' },
  { title: '管理员审核', icon: 'lucide:user-round-check' },
  { title: '成为创作者', icon: 'lucide:party-popper' }
]
const currentStep = computed(() => {
  if (isCreator.value) return 3
  if (isPending.value) return 2
  if (isEligible.value) return 1
  return 0
})

const BENEFITS = [
  '直接发布 Galgame 词条，无需排队等待审核',
  '收录 VNDB 未登录的原创 / 同人 / 独立作品',
  '提交即时生效，编辑已发布条目更自由'
]

const conditions = computed(() => {
  const e = eligibility.value
  if (!e) return []
  return [
    {
      label: '已经被合并的 Galgame 信息更新请求',
      cur: e.merged_pr_count,
      need: e.required_merged_pr_count
    },
    {
      label: '已发布 Galgame',
      cur: e.published_galgame_count,
      need: e.required_published_galgame_count
    },
    {
      label: '百字以上简评',
      cur: e.long_review_count,
      need: e.required_long_review_count
    },
    { label: '萌萌点', cur: e.moemoepoint, need: e.required_moemoepoint }
  ].map((c) => ({
    ...c,
    met: c.cur >= c.need,
    pct: c.need > 0 ? Math.min(100, Math.round((c.cur / c.need) * 100)) : 100
  }))
})

const handleApply = async () => {
  if (!canApply.value) return
  submitting.value = true
  const body = { statement: statement.value }
  const result = await settle(
    api.POST('/me/creator-applications', {
      params: {
        header: {
          'Idempotency-Key': applyKey.take('/me/creator-applications', body)
        }
      },
      body
    })
  )
  submitting.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  applyKey.clear()
  useMessage('申请已提交，等待管理员审核', 'success')
  await load()
}
</script>

<template>
  <KunModal v-model="isOpen" inner-class-name="max-w-xl">
    <div class="space-y-5 p-1">
      <div class="flex items-start gap-3">
        <KunIcon
          class="text-primary mt-0.5 size-8 shrink-0"
          name="lucide:badge-check"
        />
        <div class="space-y-0.5">
          <h2 class="text-foreground text-lg font-semibold">成为创作者</h2>
          <p class="text-default-500 text-sm">
            创作者是社区里值得信赖的发布者，可直接为大家收录 Galgame 词条。
          </p>
        </div>
      </div>

      <div v-if="loading" class="flex justify-center py-12">
        <KunIcon
          class="text-primary size-7 animate-spin"
          name="lucide:loader-circle"
        />
      </div>

      <div v-else-if="failed" class="flex flex-col items-center gap-3 py-10">
        <p class="text-default-500 text-sm">加载失败</p>
        <KunButton variant="flat" size="sm" @click="load">重试</KunButton>
      </div>

      <template v-else-if="status">
        <KunSteps
          :items="FLOW"
          :current="currentStep"
          color="primary"
          size="sm"
        />

        <div
          v-if="isCreator"
          class="bg-success-50 text-success-700 flex items-center gap-2 rounded-xl p-4 text-sm"
        >
          <KunIcon class="size-5 shrink-0" name="lucide:party-popper" />
          你已是创作者，可直接发布 Galgame 词条，无需再次申请。
        </div>

        <template v-else>
          <section class="space-y-2">
            <h3 class="text-default-700 text-sm font-medium">创作者特权</h3>
            <ul class="text-default-700 list-disc space-y-1.5 pl-5 text-sm">
              <li v-for="b in BENEFITS" :key="b">{{ b }}</li>
            </ul>
          </section>

          <KunDivider />

          <section class="space-y-3">
            <div class="flex items-center justify-between">
              <h3 class="text-default-700 text-sm font-medium">申请条件</h3>
              <KunChip
                size="sm"
                variant="flat"
                :color="isEligible ? 'success' : 'default'"
              >
                满足任一即可 · {{ isEligible ? '已满足' : '未满足' }}
              </KunChip>
            </div>
            <div v-for="c in conditions" :key="c.label" class="space-y-1">
              <div class="flex items-center justify-between gap-2 text-sm">
                <span>{{ c.label }}</span>
                <span
                  class="shrink-0"
                  :class="
                    c.met ? 'text-success font-medium' : 'text-default-500'
                  "
                >
                  {{ c.cur }} / {{ c.need }}
                </span>
              </div>
              <KunProgress
                :value="c.pct"
                size="sm"
                :color="c.met ? 'success' : 'primary'"
              />
            </div>
          </section>

          <div
            v-if="isPending"
            class="bg-primary-50 text-primary-700 flex items-center gap-2 rounded-xl p-3 text-sm"
          >
            <KunIcon class="size-5 shrink-0" name="lucide:clock" />
            申请审核中，管理员会尽快处理，结果将通过站内消息通知你。
          </div>
          <div
            v-else-if="isDeclined"
            class="bg-warning-50 text-warning-700 space-y-1 rounded-xl p-3 text-sm"
          >
            <div class="flex items-center gap-2 font-medium">
              <KunIcon class="size-5 shrink-0" name="lucide:circle-x" />
              上次申请未通过
            </div>
            <p v-if="application?.decline_reason" class="text-warning-600 pl-7">
              原因：{{ application.decline_reason }}
            </p>
          </div>

          <KunTextarea
            v-if="canApply"
            name="creator-message"
            placeholder="(可选) 附言：向管理员说明你的情况"
            :rows="2"
            v-model="statement"
          />

          <div class="flex items-center justify-end gap-2 pt-1">
            <KunButton variant="light" @click="isOpen = false">
              稍后再说
            </KunButton>
            <KunButton
              v-if="!isPending"
              color="primary"
              :disabled="!canApply"
              :loading="submitting"
              @click="handleApply"
            >
              {{
                canApply ? (isDeclined ? '重新申请' : '立即申请') : '继续努力'
              }}
            </KunButton>
          </div>
        </template>
      </template>
    </div>
  </KunModal>
</template>
