<script setup lang="ts">
import { computed, ref } from 'vue'
import { useLottery } from '~/composables/topic/useLottery'
import type { Lottery, LotteryReturnedCode } from '#shared/utils/api/schemas'

const props = defineProps<{
  lottery: Lottery
}>()

const { revealReturnedCodes } = useLottery(() => props.lottery.topic_id)

const isLoading = ref(false)
const codes = ref<LotteryReturnedCode[]>([])

const isOpen = computed({
  get: () => codes.value.length > 0,
  set: (open) => {
    if (!open) {
      codes.value = []
    }
  }
})

const count = computed(() => props.lottery.viewer?.returned_code_count ?? 0)

const prizeTitle = (prizeId: string) =>
  props.lottery.prizes.find((prize) => prize.id === prizeId)?.title ?? ''

const allCodes = computed(() =>
  codes.value.map((code) => code.redemption_code).join('\n')
)

const handleReveal = async () => {
  isLoading.value = true
  try {
    const result = await revealReturnedCodes(props.lottery.id)
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    codes.value = result.data.items
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div v-if="count > 0">
    <KunButton
      variant="flat"
      color="warning"
      :loading="isLoading"
      @click="handleReveal"
    >
      <KunIcon name="lucide:undo-2" class="mr-1" />
      查看退回的兑换码 ({{ count }})
    </KunButton>

    <KunModal v-model="isOpen" inner-class-name="max-w-lg">
      <h3 class="mb-1 text-xl font-bold">退回给您的兑换码</h3>
      <p class="text-default-500 mb-3 text-sm">
        这些兑换码没有送出: 名额没有中奖者,
        或中奖者超过期限未领取。它们不会再被抽给其他人, 请自行保存。
      </p>

      <div class="max-h-96 space-y-2 overflow-y-auto">
        <div
          v-for="code in codes"
          :key="`${code.prize_id}-${code.redemption_code}`"
          class="border-default-200 space-y-1 rounded-lg border p-2"
        >
          <div class="flex items-center justify-between gap-2 text-xs">
            <span class="text-default-500 min-w-0 truncate">
              {{ prizeTitle(code.prize_id) }}
            </span>
            <KunChip
              size="sm"
              variant="flat"
              :color="
                code.return_reason === 'forfeited' ? 'warning' : 'default'
              "
            >
              {{
                code.return_reason === 'forfeited' ? '中奖者未领取' : '无人中奖'
              }}
            </KunChip>
          </div>
          <KunCopy
            :text="code.redemption_code"
            variant="light"
            color="default"
            class-name="font-mono text-sm break-all"
          />
        </div>
      </div>

      <div class="mt-3 flex items-center justify-between gap-2">
        <span class="text-default-500 text-sm">共 {{ codes.length }} 个</span>
        <KunCopy :text="allCodes" name="复制全部" color="primary" size="sm" />
      </div>
    </KunModal>
  </div>
</template>
