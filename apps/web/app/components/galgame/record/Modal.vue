<script setup lang="ts">
import type { KunRadioOption } from '@kungal/ui-vue'
import {
  KUN_GALGAME_PLAYTIME_HOURS_MAX,
  KUN_GALGAME_PLAYTIME_MINUTES_FLOOR,
  KUN_GALGAME_PLAY_STATE_OPTIONS,
  KUN_GALGAME_PLAY_STATE_UNSTATED_DONE_OPTION,
  type KunGalgamePlayStateRead
} from '~/constants/galgame-playtime'
import {
  KUN_GALGAME_RATING_RECOMMEND_CONST,
  KUN_GALGAME_RATING_RECOMMEND_MAP,
  KUN_GALGAME_RATING_SPOILER_CONST,
  KUN_GALGAME_RATING_SPOILER_MAP,
  KUN_GALGAME_RATING_GAME_TYPE_CONST,
  KUN_GALGAME_RATING_GAME_TYPE_MAP,
  KUN_GALGAME_RATING_GAME_TYPE_DESCRIPTION_MAP,
  type KunGalgameDim
} from '~/constants/galgame-rating'
import type {
  RecordRating,
  WorkRecord,
  WorkRecordPut
} from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { galgameRatingFormSchema } from '~/validations/galgame-rating'
import { usePersistEditGalgameRatingStore } from '~/store/modules/edit/rating'
import { aspectDims, dimsToAspectScores } from '~/utils/galgame/ratingCard'

type Recommend = (typeof KUN_GALGAME_RATING_RECOMMEND_CONST)[number]
type Spoiler = (typeof KUN_GALGAME_RATING_SPOILER_CONST)[number]
type GameType = (typeof KUN_GALGAME_RATING_GAME_TYPE_CONST)[number]

const props = defineProps<{
  workId: number
  workName?: string
}>()

const emits = defineEmits<{
  saved: [WorkRecord]
}>()

const open = defineModel<boolean>({ required: true })

const { shortSummary: summaryDraft } = storeToRefs(
  usePersistEditGalgameRatingStore()
)

const api = useApiClient()
const { setPlaytime } = useMyGalgameInteractions()

const loading = ref(false)
const pending = ref(false)
const loaded = ref<WorkRecord | null>(null)

const status = ref<KunGalgamePlayStateRead | ''>('')
const hours = ref(0)

const overall = ref(0)
const recommend = ref<Recommend>('neutral')
const spoilerLevel = ref<Spoiler>('none')
const shortSummary = ref('')
const gameTypes = ref<GameType[]>([])
const showAdvanced = ref(false)
const aspectsOn = ref(false)
const emptyDims = (): Record<KunGalgameDim, number> => ({
  art: 0,
  story: 0,
  music: 0,
  character: 0,
  route: 0,
  system: 0,
  voice: 0,
  replay_value: 0
})
const dims = ref(emptyDims())

const isWish = computed(() => status.value === 'wish')
const statusOptions = computed<
  readonly KunRadioOption<KunGalgamePlayStateRead>[]
>(() =>
  loaded.value?.play_state === 'done'
    ? KUN_GALGAME_PLAY_STATE_OPTIONS.flatMap<
        KunRadioOption<KunGalgamePlayStateRead>
      >((o) =>
        o.value === 'done_one_route'
          ? [KUN_GALGAME_PLAY_STATE_UNSTATED_DONE_OPTION, o]
          : [o]
      )
    : KUN_GALGAME_PLAY_STATE_OPTIONS
)
const hadRating = computed(() => !!loaded.value?.rating_summary)

const rated = computed(() => overall.value > 0)
const hasRatingContent = computed(
  () =>
    gameTypes.value.length > 0 ||
    shortSummary.value.trim() !== '' ||
    (aspectsOn.value && Object.values(dims.value).some((v) => v > 0))
)
const deletingRating = computed(
  () => hadRating.value && !rated.value && !isWish.value
)

const minutes = computed(() => Math.round((Number(hours.value) || 0) * 60))
const tooShort = computed(
  () => minutes.value > 0 && minutes.value < KUN_GALGAME_PLAYTIME_MINUTES_FLOOR
)
const tooLong = computed(
  () => minutes.value > KUN_GALGAME_PLAYTIME_HOURS_MAX * 60
)

const recommendOptions = KUN_GALGAME_RATING_RECOMMEND_CONST.map((v) => ({
  value: v,
  label: KUN_GALGAME_RATING_RECOMMEND_MAP[v] || v
}))
const spoilerOptions = KUN_GALGAME_RATING_SPOILER_CONST.map((v) => ({
  value: v,
  label: KUN_GALGAME_RATING_SPOILER_MAP[v] || v
}))

const toggleAdvanced = () => {
  if (!aspectsOn.value) {
    const d = dims.value
    dims.value = Object.fromEntries(
      Object.entries(d).map(([k, v]) => [k, v > 0 ? v : overall.value])
    ) as Record<KunGalgameDim, number>
    aspectsOn.value = true
  }
  showAdvanced.value = !showAdvanced.value
}

const toggleGameType = (checked: boolean, t: GameType) => {
  gameTypes.value = checked
    ? [...new Set([...gameTypes.value, t])]
    : gameTypes.value.filter((x) => x !== t)
}

const clearRating = () => {
  overall.value = 0
  recommend.value = 'neutral'
  spoilerLevel.value = 'none'
  shortSummary.value = ''
  gameTypes.value = []
  dims.value = emptyDims()
  aspectsOn.value = false
  showAdvanced.value = false
}

const fillRating = (r: WorkRecord['rating_summary']) => {
  clearRating()
  if (!r) {
    shortSummary.value = summaryDraft.value
    return
  }
  overall.value = r.overall
  recommend.value = r.recommend
  spoilerLevel.value = r.spoiler_level
  shortSummary.value = r.short_summary
  gameTypes.value = [...r.game_types]
  dims.value = aspectDims(r.aspect_scores)
  aspectsOn.value = Object.values(dims.value).some((v) => v > 0)
  showAdvanced.value = aspectsOn.value
}

const restoreRating = () => fillRating(loaded.value?.rating_summary ?? null)

const fill = (record: WorkRecord) => {
  loaded.value = record
  status.value = record.play_state ?? ''
  hours.value =
    record.minutes > 0 ? Math.round((record.minutes / 60) * 10) / 10 : 0
  fillRating(record.rating_summary)
}

watch(open, async (isOpen) => {
  if (!isOpen) return
  loading.value = true
  const result = await settle(
    api.GET('/works/{work_id}/record', {
      params: { path: { work_id: String(props.workId) } }
    })
  )
  loading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    open.value = false
    return
  }
  fill(result.data)
})

watch(shortSummary, (value) => {
  if (!hadRating.value) {
    summaryDraft.value = value
  }
})

const ratingBody = (): RecordRating | null => {
  const fields = {
    recommend: recommend.value,
    overall: overall.value,
    game_types: [...gameTypes.value],
    spoiler_level: spoilerLevel.value,
    short_summary: shortSummary.value,
    aspect_scores: dimsToAspectScores(
      aspectsOn.value ? dims.value : emptyDims()
    )
  }
  if (!useKunSchemaValidator(galgameRatingFormSchema, fields)) {
    return null
  }
  return fields
}

const save = async () => {
  if (pending.value || loading.value) return
  if (!isWish.value && (tooShort.value || tooLong.value)) {
    useMessage(
      tooShort.value
        ? '不足 10 分钟的时长不会计入统计, 请填 0 或更长的时长'
        : `单部作品最多可记录 ${KUN_GALGAME_PLAYTIME_HOURS_MAX} 小时`,
      'warn'
    )
    return
  }
  if (!isWish.value && !rated.value && hasRatingContent.value) {
    useMessage(
      '请先打总体评分, 只想记录游玩状态和时长的话, 点「清除评分」',
      'warn'
    )
    return
  }

  const body: WorkRecordPut = {}
  if (status.value) body.play_state = status.value
  if (!isWish.value) {
    if (minutes.value > 0) body.minutes = minutes.value
    if (rated.value) {
      const rating = ratingBody()
      if (!rating) return
      body.rating_input = rating
    }
  }

  const losingRating = hadRating.value && !body.rating_input
  const losingPlaytime =
    (loaded.value?.minutes ?? 0) > 0 && body.minutes === undefined
  if (losingRating || losingPlaytime) {
    const ok = await useComponentMessageStore().alert(
      losingRating ? '保存后会删除你的评分' : '保存后会撤回你的游玩时长',
      [
        losingRating
          ? '评分在所有 NextMoe 站点共用, 删除后其它站点上也会一起消失。'
          : '',
        losingPlaytime
          ? '本站上报的游玩时长会撤回, 其它应用上报的不受影响。'
          : ''
      ].join('')
    )
    if (!ok) return
  }

  pending.value = true
  const result = await settle(
    api.PUT('/works/{work_id}/record', {
      params: { path: { work_id: String(props.workId) } },
      body
    })
  )
  pending.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  const saved = result.data
  if (saved.rating_summary && !hadRating.value) {
    summaryDraft.value = ''
  }
  setPlaytime(
    props.workId,
    saved.play_state || saved.minutes > 0
      ? { minutes: saved.minutes, play_state: saved.play_state }
      : null
  )
  emits('saved', saved)
  open.value = false
  useMessage('已保存', 'success')
}
</script>

<template>
  <KunModal
    v-model="open"
    inner-class-name="max-w-[780px] w-[90vw]"
    :is-dismissable="false"
    aria-label="评分"
  >
    <div class="space-y-4">
      <div>
        <h3 class="text-lg font-bold">评分</h3>
        <p v-if="workName" class="text-default-500 line-clamp-1 text-sm">
          {{ workName }}
        </p>
        <p class="text-default-500 text-sm">
          评分、游玩状态与时长在所有 NextMoe 站点共用, 在这里写的,
          在其它站点也是同一份
        </p>
      </div>

      <KunLoading v-if="loading" />

      <template v-else>
        <KunInfo
          v-if="isWish"
          color="warning"
          icon="lucide:info"
          title="想玩的作品还不能评分"
          :description="hadRating ? '保存后会删除你的评分' : ''"
        />
        <KunInfo
          v-else-if="!hadRating"
          color="secondary"
          icon="lucide:lollipop"
          title="萌萌点奖励"
          description="在本站发布评分可获得 3 萌萌点, 短评超过 20 字为 5 萌萌点, 超过 100 字为 10 萌萌点"
        />

        <div class="flex flex-wrap items-end justify-between gap-3">
          <div class="flex items-end gap-6">
            <div class="flex flex-col gap-1">
              <label class="text-sm">总体评分 (1-10)</label>
              <KunRating
                v-model="overall"
                :max="10"
                :disabled="isWish"
                aria-label="overall"
              />
            </div>
            <span
              v-if="rated"
              :class="
                cn(
                  'text-warning-500 text-4xl font-bold',
                  isWish && 'opacity-50'
                )
              "
            >
              {{ overall }}
            </span>
            <span v-else class="text-default-500 text-sm">未打分</span>
          </div>

          <KunButton
            v-if="deletingRating"
            size="sm"
            variant="flat"
            color="primary"
            @click="restoreRating"
          >
            <KunIcon name="lucide:undo-2" />恢复评分
          </KunButton>
          <KunButton
            v-else-if="rated || hasRatingContent"
            size="sm"
            variant="light"
            color="danger"
            :disabled="isWish"
            @click="clearRating"
          >
            <KunIcon name="lucide:trash-2" />
            {{ hadRating ? '删除评分' : '清除评分' }}
          </KunButton>
        </div>
        <p v-if="deletingRating" class="text-warning text-sm">
          保存后会删除你的评分, 游玩状态和时长保留
        </p>

        <div>
          <span class="text-sm">本游戏的类型 (必选, 可多选)</span>
          <div class="mt-2 flex flex-col gap-2">
            <div
              v-for="t in KUN_GALGAME_RATING_GAME_TYPE_CONST"
              :key="t"
              class="flex gap-3"
            >
              <KunCheckBox
                :model-value="gameTypes.includes(t)"
                :label="KUN_GALGAME_RATING_GAME_TYPE_MAP[t]"
                :disabled="isWish"
                @update:model-value="(checked) => toggleGameType(checked, t)"
              />
              <p class="text-default-700 text-sm">
                {{ KUN_GALGAME_RATING_GAME_TYPE_DESCRIPTION_MAP[t] }}
              </p>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <KunSelect
            v-model="recommend"
            :options="recommendOptions"
            :disabled="isWish"
            label="推荐程度"
          />
          <KunSelect
            v-model="spoilerLevel"
            :options="spoilerOptions"
            :disabled="isWish"
            label="剧透程度"
          />
        </div>

        <KunTextarea
          v-model="shortSummary"
          label="短评 (可选, 请勿包含 R18 内容, 新评分的短评会自动保存在本地, 关掉面板不会丢失)"
          :rows="4"
          :maxlength="2000"
          :show-char-count="true"
          :disabled="isWish"
          auto-grow
        />

        <div v-if="showAdvanced">
          <GalgameRatingAdvanced v-model="dims" :readonly="isWish" />
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <KunButton
            variant="flat"
            size="sm"
            :disabled="isWish"
            @click="toggleAdvanced"
          >
            {{ showAdvanced ? '收起高级评分' : '展开高级评分' }}
          </KunButton>

          <KunLink to="/doc/galgame-rating-guide" size="sm">
            鲲 Galgame 八维超评分系统介绍及使用说明
          </KunLink>
        </div>

        <div class="border-default-200 space-y-3 rounded-lg border p-3">
          <div class="flex items-center justify-between gap-2">
            <span class="font-medium">游玩记录</span>
            <KunButton
              v-if="status"
              size="sm"
              variant="light"
              color="default"
              @click="status = ''"
            >
              清除状态
            </KunButton>
          </div>
          <KunRadioGroup
            v-model="status"
            :options="statusOptions"
            variant="pill"
            orientation="horizontal"
            color="primary"
            size="sm"
            class-name="flex-wrap"
          />

          <template v-if="!isWish">
            <KunNumberInput
              v-model="hours"
              label="游玩时长 (小时, 可选)"
              :min="0"
              :max="KUN_GALGAME_PLAYTIME_HOURS_MAX"
              :step="0.5"
              :precision="1"
              placeholder="例如 12.5"
            />
            <p v-if="tooShort" class="text-warning text-sm">
              不足 10 分钟的记录不会计入统计
            </p>
          </template>
        </div>

        <div class="flex items-center justify-end gap-2">
          <KunButton variant="light" color="default" @click="open = false">
            取消
          </KunButton>
          <KunButton color="primary" :loading="pending" @click="save">
            保存
          </KunButton>
        </div>
      </template>
    </div>
  </KunModal>
</template>
