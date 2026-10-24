<script setup lang="ts">
import { useTopicEditorStore } from '~/composables/topic/useTopicEditorStore'
import { lotteryMissesCodes } from '~/composables/topic/pendingMiniApps'
import {
  TOPIC_MINI_APPS,
  type TopicMiniAppKey
} from '~/components/topic/miniapp/registry'
import { KUN_LOTTERY_DRAW_MODE_OPTIONS } from '~/constants/topic'
import type { PollFormData } from '~/components/topic/poll/types'
import type { LotteryFormData } from '~/components/topic/lottery/types'

const MAX_COUNT: Record<TopicMiniAppKey, number> = { poll: 30, lottery: 10 }

const { polls, lotteries } = useTopicEditorStore()

const isPollOpen = ref(false)
const isLotteryOpen = ref(false)
const editing = ref<{ key: TopicMiniAppKey; index: number } | null>(null)

const pollForm = computed(() =>
  editing.value?.key === 'poll' ? polls.value[editing.value.index] : undefined
)
const lotteryForm = computed(() =>
  editing.value?.key === 'lottery'
    ? lotteries.value[editing.value.index]
    : undefined
)

const countOf = (key: TopicMiniAppKey) =>
  key === 'poll' ? polls.value.length : lotteries.value.length

const openModal = (key: TopicMiniAppKey, index = -1) => {
  if (index < 0 && countOf(key) >= MAX_COUNT[key]) {
    useMessage(`每个话题最多 ${MAX_COUNT[key]} 个`, 'warn')
    return
  }
  editing.value = index < 0 ? null : { key, index }
  if (key === 'poll') {
    isPollOpen.value = true
  } else {
    isLotteryOpen.value = true
  }
}

const upsert = <T,>(list: T[], form: T, key: TopicMiniAppKey) => {
  const at = editing.value?.key === key ? editing.value.index : -1
  return at < 0
    ? [...list, form]
    : list.map((item, i) => (i === at ? form : item))
}

const onSavePoll = (form: PollFormData) => {
  polls.value = upsert(polls.value, form, 'poll')
}

const onSaveLottery = (form: LotteryFormData) => {
  lotteries.value = upsert(lotteries.value, form, 'lottery')
}

const remove = async (key: TopicMiniAppKey, index: number) => {
  const ok = await useComponentMessageStore().alert(
    '确定移除吗？',
    '移除后需要重新填写。'
  )
  if (!ok) {
    return
  }
  if (key === 'poll') {
    polls.value = polls.value.filter((_, i) => i !== index)
  } else {
    lotteries.value = lotteries.value.filter((_, i) => i !== index)
  }
}

const drawModeLabel = (mode: LotteryFormData['draw_mode']) =>
  KUN_LOTTERY_DRAW_MODE_OPTIONS.find((option) => option.value === mode)
    ?.label ?? ''

const rows = computed(() => [
  ...polls.value.map((form, index) => ({
    key: 'poll' as const,
    index,
    title: form.title,
    meta: [
      `${form.options.length} 个选项`,
      form.choice_type === 'single' ? '单选' : '多选',
      form.closes_at ? `${form.closes_at} 截止` : '不限时'
    ],
    needsCodes: false
  })),
  ...lotteries.value.map((form, index) => ({
    key: 'lottery' as const,
    index,
    title: form.title,
    meta: [
      `${form.prizes.length} 个奖项`,
      `共 ${form.prizes.reduce((sum, p) => sum + Number(p.slots || 0), 0)} 个名额`,
      drawModeLabel(form.draw_mode),
      form.deadline && `${form.deadline} 开奖`
    ],
    needsCodes: lotteryMissesCodes(form)
  }))
])
</script>

<template>
  <section class="space-y-3">
    <div>
      <h3 class="flex items-center gap-2 font-semibold">
        <KunIcon name="lucide:blocks" class="h-5 w-5" />
        投票与抽奖
        <span class="text-default-500 text-sm font-normal">(可选)</span>
      </h3>
      <p class="text-default-500 mt-1 text-sm">
        在这里添加的投票与抽奖会在话题发布时一并创建,
        发布后也可以在话题页继续添加或修改
      </p>
    </div>

    <div v-if="rows.length" class="space-y-2">
      <div
        v-for="row in rows"
        :key="`${row.key}-${row.index}`"
        class="border-default-200 space-y-2 rounded-xl border p-3"
      >
        <TopicMiniappHeader
          :app-key="row.key"
          :title="row.title"
          :meta="row.meta"
          :status="row.needsCodes ? '需重新填写兑换码' : '发布时创建'"
          :status-color="row.needsCodes ? 'warning' : 'default'"
        />
        <div class="flex justify-end gap-1">
          <KunButton
            size="sm"
            variant="light"
            @click="openModal(row.key, row.index)"
          >
            <KunIcon name="lucide:pencil" class="text-inherit" />
            编辑
          </KunButton>
          <KunButton
            size="sm"
            variant="light"
            color="danger"
            @click="remove(row.key, row.index)"
          >
            <KunIcon name="lucide:trash-2" class="text-inherit" />
            移除
          </KunButton>
        </div>
      </div>
    </div>

    <div class="flex flex-wrap gap-2">
      <KunButton
        v-for="app in TOPIC_MINI_APPS"
        :key="app.key"
        size="sm"
        variant="flat"
        :color="app.badgeColor"
        @click="openModal(app.key)"
      >
        <KunIcon :name="app.icon" class="text-inherit" />
        添加{{ app.label }}
      </KunButton>
    </div>

    <TopicPollModal
      v-model="isPollOpen"
      save-only
      :initial-form="pollForm"
      @save="onSavePoll"
    />
    <TopicLotteryModal
      v-model="isLotteryOpen"
      save-only
      :initial-form="lotteryForm"
      @save="onSaveLottery"
    />
  </section>
</template>
