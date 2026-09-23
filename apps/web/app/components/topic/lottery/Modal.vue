<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import {
  formToCreate,
  formToPatch,
  lotteryToForm,
  useLottery
} from '~/composables/topic/useLottery'
import { lotterySchema } from '~/validations/topic-lottery'
import {
  KUN_LOTTERY_DELIVERY_OPTIONS,
  KUN_LOTTERY_DRAW_MODE_OPTIONS,
  KUN_LOTTERY_ENTRY_MODE_OPTIONS,
  KUN_LOTTERY_POINT_MODE_OPTIONS
} from '~/constants/topic'
import type { Lottery } from '#shared/utils/api/schemas'
import type { LotteryFormData, LotteryPrizeFormData } from './types'

const props = defineProps<{
  modelValue: boolean
  topicId: string
  initialData?: Lottery
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  refresh: []
}>()

const isModalOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const isEditing = computed(() => !!props.initialData)
const { createLottery, updateLottery } = useLottery(() => props.topicId)
const isLoading = ref(false)

const emptyPrize = (): LotteryPrizeFormData => ({
  name: '',
  description: '',
  image_hashes: [],
  image_urls: [],
  nsfw_hashes: [],
  machine_nsfw_hashes: [],
  delivery: 'offline',
  point_mode: 'fixed',
  point_amount: 0,
  slots: 1,
  codes: ''
})

const getInitialFormData = (): LotteryFormData =>
  props.initialData
    ? lotteryToForm(props.initialData)
    : {
        title: '',
        description: '',
        entry_mode: 'signup',
        floor_rule: '',
        draw_mode: 'deadline',
        draw_threshold: 0,
        deadline: undefined,
        min_account_age_days: 3,
        min_moemoepoint: 20,
        show_entrants: true,
        prizes: [emptyPrize()]
      }

const formData = reactive<LotteryFormData>(getInitialFormData())
const rewritePrizes = ref(false)

watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen) {
      Object.assign(formData, getInitialFormData())
      rewritePrizes.value = !props.initialData
    }
  }
)

watch(rewritePrizes, (rewrite) => {
  // Codes are never sent back to the browser, so rewriting the prizes starts
  // the code boxes empty rather than pretending to show what is held.
  if (rewrite && props.initialData) {
    formData.prizes = lotteryToForm(props.initialData).prizes
  }
})

const totalSlots = computed(() =>
  formData.prizes.reduce((sum, p) => sum + Number(p.slots || 0), 0)
)

const isFloor = computed(() => formData.entry_mode === 'floor')

watch(isFloor, (floor) => {
  if (floor && formData.draw_mode === 'threshold') {
    formData.draw_mode = 'deadline'
  }
})

const addPrize = () => {
  if (formData.prizes.length >= 10) {
    useMessage('最多添加 10 个奖项', 'warn')
    return
  }
  formData.prizes.push(emptyPrize())
}

const removePrize = (index: number) => {
  if (formData.prizes.length <= 1) {
    useMessage('至少需要一个奖项', 'warn')
    return
  }
  formData.prizes.splice(index, 1)
}

const codeCount = (prize: LotteryPrizeFormData) =>
  prize.codes.split('\n').filter((c) => c.trim()).length

const isPointPool = (prize: LotteryPrizeFormData) =>
  prize.point_mode !== 'fixed'

const prizePointTotal = (prize: LotteryPrizeFormData) => {
  const amount = Number(prize.point_amount || 0)
  return isPointPool(prize) ? amount : amount * Number(prize.slots || 0)
}

const pointTotal = computed(() =>
  formData.prizes
    .filter((prize) => prize.delivery === 'point')
    .reduce((sum, prize) => sum + prizePointTotal(prize), 0)
)

const pointHint = (prize: LotteryPrizeFormData) => {
  const amount = Number(prize.point_amount || 0)
  const slots = Number(prize.slots || 0)
  if (!amount || !slots) {
    return ''
  }
  if (prize.point_mode === 'split') {
    return `奖池 ${amount} 点均分, 名额全中时每人约 ${Math.floor(amount / slots)} 点; 中奖人数不足时每人分得更多。`
  }
  if (prize.point_mode === 'random') {
    return `奖池 ${amount} 点按开奖随机数拼手气分配, 每人至少 1 点, 金额同样可以用公示的随机数验算。`
  }
  return `${slots} 个名额 × ${amount} 点 = 共 ${amount * slots} 萌萌点。`
}

const checkPrizes = () => {
  for (const prize of formData.prizes) {
    if (prize.delivery === 'code' && codeCount(prize) !== prize.slots) {
      useMessage(
        `奖项「${prize.name || '未命名'}」有 ${prize.slots} 个名额, 需要正好 ${prize.slots} 个兑换码`,
        'warn'
      )
      return false
    }
    if (
      prize.delivery === 'point' &&
      isPointPool(prize) &&
      prize.point_amount < prize.slots
    ) {
      useMessage(
        `奖项「${prize.name || '未命名'}」的奖池至少需要 ${prize.slots} 萌萌点, 每个名额至少分到 1 点`,
        'warn'
      )
      return false
    }
  }
  return true
}

const handleSubmit = async () => {
  const result = lotterySchema.safeParse(formData)
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }
  if (rewritePrizes.value && !checkPrizes()) {
    return
  }

  isLoading.value = true
  try {
    const initial = props.initialData
    if (initial) {
      const patch = formToPatch(formData, initial, rewritePrizes.value)
      if (Object.keys(patch).length === 0) {
        isModalOpen.value = false
        return
      }
      const updated = await updateLottery(initial.id, patch)
      if (!updated.ok) {
        reportProblem(updated.problem)
        return
      }
    } else {
      const created = await createLottery(formToCreate(formData))
      if (!created.ok) {
        reportProblem(created.problem)
        return
      }
    }
    emits('refresh')
    isModalOpen.value = false
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <KunModal
    :is-dismissable="false"
    v-model="isModalOpen"
    inner-class-name="max-w-3xl"
  >
    <form @submit.prevent="handleSubmit">
      <h2 class="mb-3 text-2xl font-bold">
        {{ isEditing ? '编辑抽奖' : '创建抽奖' }}
      </h2>

      <p class="text-default-500 mb-6 text-sm">
        每个话题最多 10 个抽奖, 每个抽奖最多 10 个奖项。本站只负责产生中奖名单,
        不担保实物履约, 也不代收收货地址。
      </p>

      <div class="flex flex-col gap-4">
        <KunInput v-model="formData.title" label="抽奖标题" required />
        <KunTextarea
          v-model="formData.description"
          label="抽奖说明 (可选)"
          auto-grow
          :rows="2"
        />

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <KunSelect
            v-model="formData.entry_mode"
            label="参与方式"
            :options="KUN_LOTTERY_ENTRY_MODE_OPTIONS"
          />
          <KunSelect
            v-model="formData.draw_mode"
            label="开奖方式"
            :options="KUN_LOTTERY_DRAW_MODE_OPTIONS"
          />
        </div>

        <KunInput
          v-if="isFloor"
          v-model="formData.floor_rule"
          label="中奖楼层"
          placeholder="8,18,28 或 every:10 (每 10 楼)"
          :description="`楼层数量需要与总名额一致, 当前总名额 ${totalSlots}`"
        />

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <KunDatePicker
            v-if="formData.draw_mode === 'deadline'"
            v-model="formData.deadline"
            label="开奖时间"
          />
          <KunInput
            v-if="formData.draw_mode === 'threshold'"
            v-model.number="formData.draw_threshold"
            label="满多少人开奖"
            type="number"
            :min="totalSlots"
          />
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <KunInput
            v-model.number="formData.min_account_age_days"
            label="参与门槛: 注册满 (天)"
            type="number"
            :min="0"
          />
          <KunInput
            v-model.number="formData.min_moemoepoint"
            label="参与门槛: 萌萌点不低于"
            type="number"
            :min="0"
          />
        </div>

        <KunSwitch v-model="formData.show_entrants" label="公开参与名单" />

        <div>
          <div class="mb-2 flex items-center justify-between">
            <label class="block text-sm font-medium">
              奖项设置 (共 {{ totalSlots }} 个名额)
            </label>
            <KunSwitch
              v-if="isEditing"
              v-model="rewritePrizes"
              label="修改奖项"
            />
          </div>

          <KunInfo v-if="isEditing && !rewritePrizes" title="奖项保持不变">
            <p class="text-sm">
              打开「修改奖项」会整体重写奖项列表, 已托管的兑换码会被清空,
              需要重新填写。已经有人参与后奖项不可再改。
            </p>
          </KunInfo>

          <div v-else class="flex flex-col gap-3">
            <div
              v-for="(prize, index) in formData.prizes"
              :key="index"
              class="border-default-200 space-y-3 rounded-md border p-3"
            >
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium">奖项 {{ index + 1 }}</span>
                <KunButton
                  variant="light"
                  color="danger"
                  size="sm"
                  :is-icon-only="true"
                  @click="removePrize(index)"
                >
                  <KunIcon name="lucide:trash-2" />
                </KunButton>
              </div>

              <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                <KunInput v-model="prize.name" label="奖品名称" />
                <KunInput
                  v-model.number="prize.slots"
                  label="名额"
                  type="number"
                  :min="1"
                />
              </div>

              <KunTextarea
                v-model="prize.description"
                label="奖品描述 (可选)"
                auto-grow
                :rows="2"
              />

              <KunImagesUpload
                v-model="prize.image_hashes"
                v-model:nsfw="prize.nsfw_hashes"
                :locked="prize.machine_nsfw_hashes"
                :preview-urls="prize.image_urls"
                label="奖品图片 (可选)"
                :max="9"
              />

              <KunSelect
                v-model="prize.delivery"
                label="发放方式"
                :options="KUN_LOTTERY_DELIVERY_OPTIONS"
              />

              <div v-if="prize.delivery === 'point'" class="space-y-3">
                <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                  <KunSelect
                    v-model="prize.point_mode"
                    label="萌萌点发放方式"
                    :options="KUN_LOTTERY_POINT_MODE_OPTIONS"
                  />
                  <KunInput
                    v-model.number="prize.point_amount"
                    :label="
                      isPointPool(prize)
                        ? '奖池总额 (萌萌点)'
                        : '每人获得 (萌萌点)'
                    "
                    type="number"
                    :min="1"
                  />
                </div>
                <p v-if="pointHint(prize)" class="text-default-500 text-xs">
                  {{ pointHint(prize) }}
                </p>
              </div>

              <div v-if="prize.delivery === 'code'">
                <KunTextarea
                  v-model="prize.codes"
                  label="兑换码 (每行一个)"
                  auto-grow
                  :rows="4"
                />
                <p class="text-default-500 mt-1 text-xs">
                  已填写 {{ codeCount(prize) }} 个, 需要 {{ prize.slots }} 个。
                  兑换码提交后即加密托管, 页面上永远不会再显示,
                  只有中奖者本人能揭示, 且需在开奖后 7 天内领取, 逾期自动作废。
                </p>
              </div>

              <KunInfo v-if="prize.delivery === 'offline'" title="实物类奖品">
                <p class="text-sm">
                  开奖后请通过私聊与中奖者沟通。请不要要求对方在公开楼层留下地址,
                  本站也不会代为收集收货信息。
                </p>
              </KunInfo>
            </div>

            <KunButton variant="light" size="sm" @click="addPrize">
              <KunIcon name="lucide:plus" class="mr-1" />
              增加奖项
            </KunButton>

            <KunInfo v-if="pointTotal > 0" title="萌萌点奖池由您出资">
              <p class="text-sm">
                本次抽奖最多发放 {{ pointTotal }} 萌萌点,
                发布时会从您的余额中扣除这部分作为奖池, 开奖时发到中奖者账户。
                名额没有发满的部分会在开奖后退回; 取消或在开奖前删除抽奖会全额退回。
              </p>
            </KunInfo>
          </div>
        </div>
      </div>

      <div class="mt-8 flex justify-end gap-3">
        <KunButton variant="light" @click="isModalOpen = false">取消</KunButton>
        <KunButton type="submit" color="primary" :loading="isLoading">
          {{ isEditing ? '保存更改' : '发布抽奖' }}
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
