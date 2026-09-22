<script setup lang="ts">
import { computed, ref } from 'vue'
import { useLottery } from '~/composables/topic/useLottery'
import {
  KUN_LOTTERY_DELIVERY,
  KUN_LOTTERY_DRAW_MODE,
  KUN_LOTTERY_ENTRY_MODE,
  KUN_LOTTERY_STATUS
} from '~/constants/topic'

const props = defineProps<{
  lottery: TopicLottery
  isTopicAdmin: boolean
}>()

const emits = defineEmits<{
  edit: [lottery: TopicLottery]
  refresh: []
}>()

const { id: currentUserId } = usePersistUserStore()
const {
  enter,
  withdraw,
  drawNow,
  cancel,
  deleteLottery,
  getEntrants,
  claimCode
} = useLottery(props.lottery.topic_id)

const isLoading = ref(false)
const isEntrantsOpen = ref(false)
const entrants = ref<TopicLotteryEntrant[]>([])
const revealedCode = ref('')
const isFairnessOpen = ref(false)

const nowMs = useState(`kun-lottery-now-${useId()}`, () => Date.now())
onMounted(() => {
  nowMs.value = Date.now()
})

const isAuthor = computed(() => props.lottery.user.id === currentUserId)
const canManage = computed(() => isAuthor.value || props.isTopicAdmin)
const isOpen = computed(() => props.lottery.status === 'open')
const isDrawn = computed(() => props.lottery.status === 'drawn')
const isFloor = computed(() => props.lottery.entry_mode === 'floor')

const statusColor = computed(() => {
  switch (props.lottery.status) {
    case 'open':
      return 'success'
    case 'drawing':
      return 'warning'
    case 'drawn':
      return 'primary'
    default:
      return 'default'
  }
})

const countdown = computed(() => {
  if (!props.lottery.deadline || !isOpen.value) {
    return ''
  }
  const remain = new Date(props.lottery.deadline).getTime() - nowMs.value
  if (remain <= 0) {
    return '即将开奖'
  }
  const days = Math.floor(remain / 86400000)
  const hours = Math.floor((remain % 86400000) / 3600000)
  const minutes = Math.floor((remain % 3600000) / 60000)
  if (days > 0) {
    return `还剩 ${days} 天 ${hours} 小时`
  }
  if (hours > 0) {
    return `还剩 ${hours} 小时 ${minutes} 分`
  }
  return `还剩 ${minutes} 分钟`
})

const thresholdProgress = computed(() => {
  if (
    props.lottery.draw_mode !== 'threshold' ||
    !props.lottery.draw_threshold
  ) {
    return 0
  }
  return Math.min(
    100,
    (props.lottery.entry_count / props.lottery.draw_threshold) * 100
  )
})

const anyPrizeImage = computed(() =>
  props.lottery.prizes.some((prize) => prize.image_hashes.length > 0)
)

// image_urls is parallel to image_hashes and the server blanks the entries this
// reader may not see, so a missing URL is the whole withholding signal.
const visibleImages = (prize: TopicLotteryPrize) =>
  prize.image_hashes
    .map((hash, index) => ({
      hash,
      url: prize.image_urls[index] ?? '',
      isNSFW:
        prize.nsfw_hashes.includes(hash) ||
        prize.machine_nsfw_hashes.includes(hash)
    }))
    .filter((image) => !!image.url)

const hiddenCount = (prize: TopicLotteryPrize) =>
  prize.image_hashes.length - visibleImages(prize).length

const hiddenImageTotal = computed(() =>
  props.lottery.prizes.reduce((sum, prize) => sum + hiddenCount(prize), 0)
)

const { isSignedIn, setAnonymousNsfw } = useContentStance()
const { open: openSettingPanel } = useSettingPanel()

const enableNsfw = () => {
  if (isSignedIn.value) {
    openSettingPanel('content')
    return
  }
  setAnonymousNsfw(true)
  location.reload()
}

const pointLine = (prize: TopicLotteryPrize) => {
  if (prize.delivery !== 'point') {
    return ''
  }
  if (prize.point_mode === 'split') {
    return `奖池 ${prize.point_amount} 萌萌点 · 中奖者均分`
  }
  if (prize.point_mode === 'random') {
    return `奖池 ${prize.point_amount} 萌萌点 · 拼手气`
  }
  return `每人 ${prize.point_amount} 萌萌点`
}

const hasRandomPointPrize = computed(() =>
  props.lottery.prizes.some(
    (prize) => prize.delivery === 'point' && prize.point_mode === 'random'
  )
)

const isThresholdOpen = computed(
  () => props.lottery.draw_mode === 'threshold' && isOpen.value
)

const myWin = computed(() =>
  props.lottery.my_prize_id > 0 ? props.lottery.my_prize_name : ''
)

const metaLine = computed(() => [
  KUN_LOTTERY_ENTRY_MODE[props.lottery.entry_mode],
  KUN_LOTTERY_DRAW_MODE[props.lottery.draw_mode],
  isFloor.value
    ? `中奖楼层 ${props.lottery.floor_rule}`
    : `${props.lottery.entry_count} 人参与 / ${props.lottery.total_slots} 个名额`,
  props.lottery.min_moemoepoint > 0
    ? `门槛 ${props.lottery.min_moemoepoint} 萌萌点`
    : '',
  props.lottery.min_account_age_days > 0
    ? `需注册满 ${props.lottery.min_account_age_days} 天`
    : ''
])

const run = async (task: () => Promise<unknown>) => {
  isLoading.value = true
  try {
    await task()
  } finally {
    isLoading.value = false
  }
}

const handleEnter = async () => {
  if (!requireLogin()) return
  await run(async () => {
    await enter(props.lottery.id)
    emits('refresh')
  })
}

const handleWithdraw = async () => {
  await run(async () => {
    await withdraw(props.lottery.id)
    emits('refresh')
  })
}

const handleDraw = async () => {
  await run(async () => {
    if (await drawNow(props.lottery.id)) {
      emits('refresh')
    }
  })
}

const handleCancel = async () => {
  await run(async () => {
    if (await cancel(props.lottery.id)) {
      emits('refresh')
    }
  })
}

const handleDelete = async () => {
  await run(async () => {
    if (await deleteLottery(props.lottery.id)) {
      emits('refresh')
    }
  })
}

const handleShowEntrants = async () => {
  await run(async () => {
    entrants.value = (await getEntrants(props.lottery.id)) ?? []
    isEntrantsOpen.value = true
  })
}

const handleClaim = async () => {
  await run(async () => {
    const res = await claimCode(props.lottery.id)
    if (!res?.code) {
      return
    }
    revealedCode.value = res.code
    emits('refresh')
  })
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-3"
  >
    <TopicMiniappHeader
      app-key="lottery"
      :title="lottery.title"
      :meta="metaLine"
      :status="KUN_LOTTERY_STATUS[lottery.status] ?? '进行中'"
      :status-color="statusColor"
    />

    <p v-if="lottery.description" class="text-default-500 text-sm">
      {{ lottery.description }}
    </p>

    <div class="grid gap-2 sm:grid-cols-2">
      <div
        v-for="prize in lottery.prizes"
        :key="prize.id"
        class="border-default-200 flex items-start gap-3 rounded-lg border p-3"
      >
        <div
          v-if="prize.image_hashes.length"
          class="flex shrink-0 items-start gap-1"
        >
          <KunLightboxGallery v-if="visibleImages(prize).length">
            <div class="flex shrink-0 gap-1">
              <KunLightboxGalleryItem
                v-for="(image, imageIndex) in visibleImages(prize)"
                :key="image.hash"
                :src="image.url"
                :alt="prize.name"
                :wrap="false"
              >
                <template #default="{ open }">
                  <button
                    v-if="imageIndex < 2"
                    type="button"
                    class="relative size-14 shrink-0 cursor-zoom-in overflow-hidden rounded-md"
                    :aria-label="`查看 ${prize.name} 的图片`"
                    @click="open"
                  >
                    <KunImage
                      :src="image.url"
                      :alt="prize.name"
                      class="size-full object-cover"
                    />
                    <span
                      v-if="image.isNSFW"
                      class="bg-danger absolute top-0.5 left-0.5 rounded px-1 py-0.5 text-[10px] leading-none font-medium text-white"
                    >
                      R18
                    </span>
                    <span
                      v-if="imageIndex === 1 && visibleImages(prize).length > 2"
                      class="absolute inset-0 flex items-center justify-center bg-black/55 text-xs font-medium text-white"
                    >
                      +{{ visibleImages(prize).length - 2 }}
                    </span>
                  </button>
                </template>
              </KunLightboxGalleryItem>
            </div>
          </KunLightboxGallery>

          <span
            v-if="hiddenCount(prize)"
            class="border-default-300 text-default-400 flex size-14 shrink-0 flex-col items-center justify-center gap-0.5 rounded-md border border-dashed"
            :title="`${hiddenCount(prize)} 张成人内容图片已按您的内容设置隐藏`"
          >
            <KunIcon name="lucide:eye-off" class="size-4 text-inherit" />
            <span class="text-[10px] leading-none tabular-nums">
              {{ hiddenCount(prize) }} 张
            </span>
          </span>
        </div>
        <span
          v-else-if="anyPrizeImage"
          class="bg-default-100 text-default-400 flex size-14 shrink-0 items-center justify-center rounded-md"
        >
          <KunIcon name="lucide:package" class="text-xl text-inherit" />
        </span>

        <div class="min-w-0 flex-1">
          <div class="flex items-start justify-between gap-2">
            <span class="min-w-0 font-medium">{{ prize.name }}</span>
            <KunChip size="sm" variant="flat" color="secondary">
              {{ prize.slots }} 名
            </KunChip>
          </div>
          <p class="text-default-500 text-xs">
            {{ KUN_LOTTERY_DELIVERY[prize.delivery] }}
            <template v-if="prize.delivery === 'point'">
              · {{ pointLine(prize) }}
            </template>
          </p>
          <p v-if="prize.description" class="text-default-500 mt-1 text-xs">
            {{ prize.description }}
          </p>
        </div>
      </div>
    </div>

    <p v-if="hiddenImageTotal" class="text-default-500 text-xs">
      该抽奖有 {{ hiddenImageTotal }} 张奖品图片被标记为成人内容,
      已按您的内容设置隐藏。
      <button
        type="button"
        class="text-primary cursor-pointer underline-offset-2 hover:underline"
        @click="enableNsfw"
      >
        开启 NSFW 模式
      </button>
      后可以看到它们。
    </p>

    <div
      v-if="countdown || isThresholdOpen"
      :class="
        cn(
          'bg-default-100 space-y-2 rounded-lg px-3 py-2',
          !isThresholdOpen && 'w-fit'
        )
      "
    >
      <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
        <span
          v-if="countdown"
          class="text-warning-600 dark:text-warning-400 flex items-center gap-1 font-medium"
        >
          <KunIcon name="lucide:timer" class="text-inherit" />
          {{ countdown }}
        </span>
        <span v-if="isThresholdOpen" class="text-default-500 ml-auto text-xs">
          满 {{ lottery.draw_threshold }} 人开奖, 已有
          {{ lottery.entry_count }} 人
        </span>
      </div>

      <KunProgress
        v-if="isThresholdOpen"
        :value="thresholdProgress"
        color="secondary"
        size="sm"
        :aria-label="`满 ${lottery.draw_threshold} 人开奖`"
      />
    </div>

    <div v-if="lottery.seed_hash" class="text-default-500 space-y-2 text-xs">
      <KunButton
        size="sm"
        variant="light"
        color="default"
        :aria-expanded="isFairnessOpen"
        @click="isFairnessOpen = !isFairnessOpen"
      >
        <KunIcon name="lucide:shield-check" class="text-inherit" />
        公平性凭据
        <KunIcon
          :name="isFairnessOpen ? 'lucide:chevron-up' : 'lucide:chevron-down'"
          class="text-inherit"
        />
      </KunButton>

      <div
        v-if="isFairnessOpen"
        class="border-default-200 space-y-1 rounded-lg border p-3"
      >
        <p>
          随机数承诺 (创建时公示):
          <code class="break-all">{{ lottery.seed_hash }}</code>
        </p>
        <p v-if="lottery.seed">
          随机数 (开奖时公布):
          <code class="break-all">{{ lottery.seed }}</code>
        </p>
        <p v-else>随机数将在开奖后公布, 届时可自行验算中奖名单。</p>
        <p v-if="lottery.seed">
          验算方式: sha256(随机数) 应等于上面的承诺; 每位参与者的排序值 =
          HMAC-SHA256(随机数, "{{ lottery.id }}:用户 ID"), 取最小的若干名。
        </p>
        <p v-if="lottery.seed && hasRandomPointPrize">
          拼手气奖池的分配权重 = HMAC-SHA256(随机数, "point:奖项 ID:用户 ID")
          的前 4 字节, 每人先保底 1 点, 余下的按权重比例分配。
        </p>
      </div>
    </div>

    <TopicLotteryResult
      v-if="isDrawn"
      :lottery="lottery"
      :can-manage="canManage"
      @refresh="emits('refresh')"
    />

    <div
      class="border-default-200 flex flex-wrap items-center justify-between gap-2 border-t pt-3"
    >
      <div class="flex flex-wrap items-center gap-2">
        <KunButton
          v-if="lottery.can_enter"
          color="primary"
          :loading="isLoading"
          @click="handleEnter"
        >
          参与抽奖
        </KunButton>

        <KunButton
          v-else-if="lottery.has_entered && isOpen"
          variant="bordered"
          :loading="isLoading"
          @click="handleWithdraw"
        >
          退出抽奖
        </KunButton>

        <span
          v-else-if="isOpen && lottery.enter_blocked"
          class="text-default-500 text-sm"
        >
          {{ lottery.enter_blocked }}
        </span>

        <KunButton
          v-if="
            myWin &&
            lottery.my_code_ready &&
            lottery.my_fulfillment !== 'forfeited' &&
            !revealedCode
          "
          color="success"
          :loading="isLoading"
          @click="handleClaim"
        >
          <KunIcon name="lucide:key-round" class="mr-1" />
          领取兑换码
        </KunButton>

        <KunButton
          v-if="lottery.show_entrants && !isFloor && lottery.entry_count > 0"
          variant="light"
          size="sm"
          :loading="isLoading"
          @click="handleShowEntrants"
        >
          参与名单
        </KunButton>
      </div>

      <div v-if="canManage" class="flex items-center gap-1">
        <KunTooltip v-if="isOpen" text="立即开奖">
          <KunButton
            variant="light"
            color="default"
            size="sm"
            :is-icon-only="true"
            :loading="isLoading"
            @click="handleDraw"
          >
            <KunIcon name="lucide:dices" />
          </KunButton>
        </KunTooltip>
        <KunTooltip v-if="isOpen" text="编辑抽奖">
          <KunButton
            variant="light"
            color="default"
            size="sm"
            :is-icon-only="true"
            @click="emits('edit', lottery)"
          >
            <KunIcon name="lucide:pencil" />
          </KunButton>
        </KunTooltip>
        <KunTooltip v-if="isOpen" text="取消抽奖">
          <KunButton
            variant="light"
            color="warning"
            size="sm"
            :is-icon-only="true"
            @click="handleCancel"
          >
            <KunIcon name="lucide:ban" />
          </KunButton>
        </KunTooltip>
        <KunTooltip text="删除抽奖">
          <KunButton
            variant="light"
            color="danger"
            size="sm"
            :is-icon-only="true"
            @click="handleDelete"
          >
            <KunIcon name="lucide:trash-2" />
          </KunButton>
        </KunTooltip>
      </div>
    </div>

    <KunInfo v-if="revealedCode" title="您的兑换码">
      <div class="flex flex-wrap items-center gap-3">
        <code class="text-base break-all">{{ revealedCode }}</code>
        <KunCopy :text="revealedCode" color="primary" size="sm" />
      </div>
      <p class="text-default-500 mt-2 text-sm">
        请立即保存, 关闭后需要重新点击「领取兑换码」才能再次查看。
      </p>
    </KunInfo>

    <KunModal v-model="isEntrantsOpen" inner-class-name="max-w-lg">
      <h3 class="mb-3 text-xl font-bold">参与名单</h3>
      <div class="max-h-96 space-y-2 overflow-y-auto">
        <div
          v-for="entrant in entrants"
          :key="entrant.user.id"
          class="flex items-center gap-2"
        >
          <KunAvatar :user="entrant.user" size="sm" />
          <span class="text-sm">{{ entrant.user.name }}</span>
        </div>
        <p v-if="!entrants.length" class="text-default-500 text-sm">
          还没有人参与
        </p>
      </div>
    </KunModal>
  </KunCard>
</template>
