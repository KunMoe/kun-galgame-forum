<script setup lang="ts">
import {
  KUN_GALGAME_PLAYTIME_HOURS_MAX,
  KUN_GALGAME_PLAYTIME_MINUTES_FLOOR,
  KUN_GALGAME_PLAY_STATE_CONST,
  KUN_GALGAME_PLAY_STATE_DONE,
  KUN_GALGAME_PLAY_STATE_OPTIONS,
  type KunGalgamePlayState
} from '~/constants/galgame-playtime'

const props = defineProps<{
  galgame: GalgameDetail
  mine: GalgameMyPlaytime | null
}>()

const emits = defineEmits<{
  saved: [GalgameMyPlaytime | null]
  finished: [KunGalgamePlayState]
}>()

const open = defineModel<boolean>({ required: true })

const hours = ref(0)
const status = ref<KunGalgamePlayState | ''>('')

watch(open, (isOpen) => {
  if (!isOpen) return
  hours.value =
    props.mine && props.mine.minutes > 0
      ? Math.round((props.mine.minutes / 60) * 10) / 10
      : 0
  const current = props.mine?.status ?? ''
  const allowed = KUN_GALGAME_PLAY_STATE_CONST as readonly string[]
  if (current === 'done') {
    status.value = 'done_one_route'
  } else if (allowed.includes(current)) {
    status.value = current as KunGalgamePlayState
  } else {
    status.value = ''
  }
})

const minutes = computed(() => Math.round((Number(hours.value) || 0) * 60))

const tooShort = computed(
  () => minutes.value > 0 && minutes.value < KUN_GALGAME_PLAYTIME_MINUTES_FLOOR
)
const tooLong = computed(
  () => minutes.value > KUN_GALGAME_PLAYTIME_HOURS_MAX * 60
)

const pending = ref(false)

const canSave = computed(() => {
  if (pending.value) return false
  if (status.value) return true
  return minutes.value > 0 && !tooShort.value && !tooLong.value
})

const submit = async (body: { minutes?: number; status?: string }) => {
  pending.value = true
  let failed = false
  const result = await kunFetch<GalgameMyPlaytime>(
    `/galgame/${props.galgame.id}/playtime`,
    {
      method: 'PUT',
      body,
      onApiError: () => {
        failed = true
        return false
      }
    }
  )
  pending.value = false
  const isClear = body.minutes === 0 && body.status === ''
  if (failed || (result == null && !isClear)) return
  const mine =
    result && (result.minutes > 0 || result.status) ? result : null
  emits('saved', mine)
  const marked = (mine?.status || body.status || '') as string
  if (
    (KUN_GALGAME_PLAY_STATE_DONE as readonly string[]).includes(marked)
  ) {
    emits('finished', marked as KunGalgamePlayState)
  }
  open.value = false
  useMessage(isClear ? '已清除游玩记录' : '已保存', 'success')
}

const save = () => {
  if (!canSave.value) return
  const body: { minutes?: number; status?: string } = {}
  if (status.value) body.status = status.value
  // 0 means withdraw. Sending it unasked would wipe a duration the user still wants.
  if (minutes.value > 0 && !tooShort.value && !tooLong.value) {
    body.minutes = minutes.value
  }
  if (body.status == null && body.minutes == null) return
  submit(body)
}

const clear = async () => {
  if (pending.value) return
  const ok = await useComponentMessageStore().alert(
    '清除游玩记录',
    '本站这一条记录的时长和游玩状态都会被清零, 不再计入本站中位数。其它应用上报的记录不受影响。'
  )
  if (!ok) return
  submit({ minutes: 0, status: '' })
}
</script>

<template>
  <KunModal
    v-model="open"
    inner-class-name="max-w-md w-full"
    aria-label="记录游玩状态"
  >
    <div class="space-y-4">
      <div>
        <h3 class="text-lg font-bold">记录游玩状态</h3>
        <p class="text-default-500 line-clamp-1 text-sm">
          {{ galgame.name }}
        </p>
      </div>

      <div class="space-y-2">
        <span class="text-default-600 text-sm">游玩状态</span>
        <KunRadioGroup
          v-model="status"
          :options="KUN_GALGAME_PLAY_STATE_OPTIONS"
          variant="pill"
          orientation="horizontal"
          color="primary"
          size="sm"
          class-name="flex-wrap"
        />
      </div>

      <KunNumberInput
        v-model="hours"
        label="通关用时 (小时, 可选)"
        :min="0"
        :max="KUN_GALGAME_PLAYTIME_HOURS_MAX"
        :step="0.5"
        :precision="1"
        placeholder="例如 12.5"
      />

      <KunInfo
        v-if="tooShort"
        color="warning"
        title="太短了"
        description="不足 10 分钟的记录不会计入统计, 如需撤回请使用清除。"
      />
      <KunInfo
        v-else-if="tooLong"
        color="warning"
        title="超出上限"
        :description="`单部作品最多可记录 ${KUN_GALGAME_PLAYTIME_HOURS_MAX} 小时。`"
      />
      <p v-else class="text-default-500 text-sm">
        只有「单线 / 主线 / 全线通关」的记录会计入本站中位数, 且需要至少 3
        位玩家上报。
      </p>

      <div class="flex items-center justify-end gap-2">
        <KunButton
          v-if="mine"
          variant="light"
          color="danger"
          :disabled="pending"
          @click="clear"
        >
          清除记录
        </KunButton>
        <KunButton variant="light" color="default" @click="open = false">
          取消
        </KunButton>
        <KunButton
          color="primary"
          :loading="pending"
          :disabled="!canSave"
          @click="save"
        >
          保存
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
