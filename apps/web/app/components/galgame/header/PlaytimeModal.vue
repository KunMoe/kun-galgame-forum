<script setup lang="ts">
import {
  KUN_GALGAME_PLAYTIME_HOURS_MAX,
  KUN_GALGAME_PLAYTIME_MINUTES_FLOOR,
  KUN_GALGAME_PLAYTIME_STATUS_CONST,
  KUN_GALGAME_PLAYTIME_STATUS_OPTIONS,
  type KunGalgamePlaytimeStatus
} from '~/constants/galgame-playtime'

const props = defineProps<{
  galgame: GalgameDetail
  mine: GalgameMyPlaytime | null
}>()

const emits = defineEmits<{
  saved: [GalgameMyPlaytime | null]
}>()

const open = defineModel<boolean>({ required: true })

const hours = ref(0)
const status = ref<KunGalgamePlaytimeStatus>('doing')

watch(open, (isOpen) => {
  if (!isOpen) return
  hours.value = props.mine ? Math.round((props.mine.minutes / 60) * 10) / 10 : 0
  const current = props.mine?.status ?? ''
  const allowed = KUN_GALGAME_PLAYTIME_STATUS_CONST as readonly string[]
  status.value = allowed.includes(current)
    ? (current as KunGalgamePlaytimeStatus)
    : 'doing'
})

const minutes = computed(() => Math.round((Number(hours.value) || 0) * 60))

const tooShort = computed(
  () => minutes.value > 0 && minutes.value < KUN_GALGAME_PLAYTIME_MINUTES_FLOOR
)
const tooLong = computed(
  () => minutes.value > KUN_GALGAME_PLAYTIME_HOURS_MAX * 60
)

const pending = ref(false)

const submit = async (payload: number) => {
  pending.value = true
  const result = await kunFetch<GalgameMyPlaytime>(
    `/galgame/${props.galgame.id}/playtime`,
    { method: 'PUT', body: { minutes: payload, status: status.value } }
  )
  pending.value = false
  if (!result) return
  emits('saved', payload > 0 ? result : null)
  open.value = false
  useMessage(payload > 0 ? '已记录游玩时长' : '已清除游玩时长', 'success')
}

const save = () => {
  if (pending.value || tooShort.value || tooLong.value || !minutes.value) return
  submit(minutes.value)
}

const clear = async () => {
  if (pending.value) return
  const ok = await useComponentMessageStore().alert(
    '清除游玩时长',
    '本站这一条记录会被清零, 不再计入本站中位数。其它应用上报的记录不受影响。'
  )
  if (!ok) return
  submit(0)
}
</script>

<template>
  <KunModal
    v-model="open"
    inner-class-name="max-w-md w-full"
    aria-label="记录游玩时长"
  >
    <div class="space-y-4">
      <div>
        <h3 class="text-lg font-bold">记录游玩时长</h3>
        <p class="text-default-500 line-clamp-1 text-sm">
          {{ galgame.name }}
        </p>
      </div>

      <KunNumberInput
        v-model="hours"
        label="通关用时 (小时)"
        :min="0"
        :max="KUN_GALGAME_PLAYTIME_HOURS_MAX"
        :step="0.5"
        :precision="1"
        placeholder="例如 12.5"
      />

      <div class="space-y-2">
        <span class="text-default-600 text-sm">游玩状态</span>
        <KunRadioGroup
          v-model="status"
          :options="KUN_GALGAME_PLAYTIME_STATUS_OPTIONS"
          variant="pill"
          orientation="horizontal"
          color="primary"
          size="sm"
        />
      </div>

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
        只有「已通关」的记录会计入本站中位数, 且需要至少 3 位玩家上报。
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
          :disabled="!minutes || tooShort || tooLong"
          @click="save"
        >
          保存
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
