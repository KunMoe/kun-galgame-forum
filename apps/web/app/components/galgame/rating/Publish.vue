<script setup lang="ts">
import {
  KUN_GALGAME_PLAY_STATE_CONST,
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayState
} from '~/constants/galgame-playtime'
import {
  KUN_GALGAME_RATING_RECOMMEND_CONST,
  KUN_GALGAME_RATING_RECOMMEND_MAP,
  KUN_GALGAME_RATING_SPOILER_CONST,
  KUN_GALGAME_RATING_SPOILER_MAP,
  KUN_GALGAME_RATING_GAME_TYPE_CONST,
  KUN_GALGAME_RATING_GAME_TYPE_MAP,
  KUN_GALGAME_RATING_GAME_TYPE_DESCRIPTION_MAP
} from '~/constants/galgame-rating'
import {
  createGalgameRatingSchema,
  updateGalgameRatingSchema
} from '~/validations/galgame-rating'
import { usePersistEditGalgameRatingStore } from '~/store/modules/edit/rating'

type RatingInitialData = {
  galgameRatingId: number
  recommend: (typeof KUN_GALGAME_RATING_RECOMMEND_CONST)[number]
  overall: number
  play_status: KunGalgamePlayState
  spoiler_level: (typeof KUN_GALGAME_RATING_SPOILER_CONST)[number]
  short_summary: string
  art: number
  story: number
  music: number
  character: number
  route: number
  system: number
  voice: number
  replay_value: number
  galgameType?: string[]
}

const props = defineProps<{
  galgameId: number
  modelValue: boolean
  initialData?: RatingInitialData
  presetPlayState?: KunGalgamePlayState
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  onUpdated: []
  onPublished: [GalgameRatingCardOnGalgamePage]
}>()

const { shortSummary: shortSummaryStore } = storeToRefs(
  usePersistEditGalgameRatingStore()
)

const recommend =
  ref<(typeof KUN_GALGAME_RATING_RECOMMEND_CONST)[number]>('neutral')
const playStatus = ref<KunGalgamePlayState | ''>('')
const spoilerLevel =
  ref<(typeof KUN_GALGAME_RATING_SPOILER_CONST)[number]>('none')
const overall = ref(1)
const shortSummary = ref('')
const selectedTypes = ref<string[]>([])

const toggleGameType = (checked: boolean, t: string) => {
  if (checked) {
    if (!selectedTypes.value.includes(t)) selectedTypes.value.push(t)
  } else {
    selectedTypes.value = selectedTypes.value.filter((x) => x !== t)
  }
}

const showAdvanced = ref(false)
const dims = ref({
  art: 1,
  story: 1,
  music: 1,
  character: 1,
  route: 1,
  system: 1,
  voice: 1,
  replay_value: 1
})

watch(
  () => overall.value,
  (val) => {
    if (val <= 0 || props.initialData) {
      return
    }
    dims.value = {
      art: val,
      story: val,
      music: val,
      character: val,
      route: val,
      system: val,
      voice: val,
      replay_value: val
    }
  }
)

watch(
  () => shortSummary.value,
  () => {
    if (!props.initialData) {
      shortSummaryStore.value = shortSummary.value
    }
  }
)

const isSubmitting = ref(false)

const recommendOptions = computed(() =>
  KUN_GALGAME_RATING_RECOMMEND_CONST.map((v) => ({
    value: v,
    label: KUN_GALGAME_RATING_RECOMMEND_MAP[v] || v
  }))
)
const playStatusOptions = computed(() =>
  KUN_GALGAME_PLAY_STATE_CONST.map((v) => ({
    value: v,
    label: KUN_GALGAME_PLAY_STATE_MAP[v] || v
  }))
)
const spoilerOptions = computed(() =>
  KUN_GALGAME_RATING_SPOILER_CONST.map((v) => ({
    value: v,
    label: KUN_GALGAME_RATING_SPOILER_MAP[v] || v
  }))
)

const close = () => emits('update:modelValue', false)
const isEditing = computed(() => !!props.initialData?.galgameRatingId)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    if (props.initialData) {
      recommend.value = props.initialData.recommend
      playStatus.value = props.initialData.play_status
      spoilerLevel.value = props.initialData.spoiler_level
      overall.value = props.initialData.overall
      shortSummary.value = props.initialData.short_summary || ''
      dims.value = {
        art: props.initialData.art,
        story: props.initialData.story,
        music: props.initialData.music,
        character: props.initialData.character,
        route: props.initialData.route,
        system: props.initialData.system,
        voice: props.initialData.voice,
        replay_value: props.initialData.replay_value
      }
      selectedTypes.value = [...(props.initialData.galgameType ?? [])]
      showAdvanced.value = true
    } else {
      shortSummary.value = shortSummaryStore.value
      playStatus.value = props.presetPlayState ?? ''
    }
  }
)

const resetForm = () => {
  recommend.value = 'neutral'
  playStatus.value = ''
  spoilerLevel.value = 'none'
  overall.value = 1
  shortSummary.value = ''
  showAdvanced.value = false
  dims.value = {
    art: 1,
    story: 1,
    music: 1,
    character: 1,
    route: 1,
    system: 1,
    voice: 1,
    replay_value: 1
  }
  selectedTypes.value = []
}

const submit = async () => {
  if (isEditing.value) {
    const body = {
      galgame_rating_id: props.initialData!.galgameRatingId,
      recommend: recommend.value,
      overall: overall.value,
      play_status: playStatus.value,
      spoiler_level: spoilerLevel.value,
      short_summary: shortSummary.value,
      art: dims.value.art,
      story: dims.value.story,
      music: dims.value.music,
      character: dims.value.character,
      route: dims.value.route,
      system: dims.value.system,
      voice: dims.value.voice,
      replay_value: dims.value.replay_value,
      galgame_type: selectedTypes.value
    }
    const valid = useKunSchemaValidator(updateGalgameRatingSchema, body)
    if (!valid) {
      return
    }

    isSubmitting.value = true
    const res = await kunFetch(
      `/galgame-rating/${props.initialData!.galgameRatingId}`,
      {
        method: 'PUT',
        body
      }
    )
    isSubmitting.value = false
    if (res) {
      useMessage('更新成功', 'success')
      emits('onUpdated')
      close()
    }
  } else {
    const body = {
      galgame_id: props.galgameId,
      recommend: recommend.value,
      overall: overall.value,
      play_status: playStatus.value,
      spoiler_level: spoilerLevel.value,
      short_summary: shortSummary.value,
      art: dims.value.art,
      story: dims.value.story,
      music: dims.value.music,
      character: dims.value.character,
      route: dims.value.route,
      system: dims.value.system,
      voice: dims.value.voice,
      replay_value: dims.value.replay_value,
      galgame_type: selectedTypes.value
    }
    const valid = useKunSchemaValidator(createGalgameRatingSchema, body)
    if (!valid) return

    isSubmitting.value = true
    const res = await kunFetch<GalgameRatingCardOnGalgamePage>(
      '/galgame-rating',
      {
        method: 'POST',
        body
      }
    )
    isSubmitting.value = false
    if (res) {
      useMessage('发布成功', 'success')
      resetForm()
      close()
      emits('onPublished', res)
      shortSummaryStore.value = ''
    }
  }
}
</script>

<template>
  <KunModal
    :model-value="modelValue"
    @update:model-value="(v) => emits('update:modelValue', v)"
    inner-class-name="max-w-[780px] w-[90vw]"
    :is-dismissable="false"
    :aria-label="isEditing ? '编辑评分' : '发布评分'"
  >
    <div class="space-y-3">
      <KunHeader
        :name="isEditing ? '编辑评分' : '发布评分'"
        description="您可以选择性的使用高级评分, 每条评分将会平均有 1w+ 的曝光数, 您的评分会左右本网站 Galgame 的状态"
        scale="h3"
      />

      <KunLink to="/doc/galgame-rating-guide">
        鲲 Galgame 八维超评分系统介绍及使用说明
      </KunLink>

      <KunInfo
        title="萌萌点奖励"
        color="secondary"
        icon="lucide:lollipop"
        description="每发布一个评分, 您会获得 3 萌萌点, 短评超过 20 字为 5 萌萌点, 超过 100 字为 10 萌萌点"
      />

      <div class="flex items-end gap-6">
        <div class="flex flex-col gap-1">
          <label class="text-sm">总体评分 (1-10)</label>
          <KunRating v-model="overall" :max="10" aria-label="overall" />
        </div>

        <span class="text-warning-500 text-4xl font-bold">{{ overall }}</span>
      </div>

      <div>
        <span class="text-sm">本游戏的类型 (必选, 可多选)</span>
        <div class="mt-2 flex flex-col gap-2">
          <div
            class="flex gap-3"
            v-for="t in KUN_GALGAME_RATING_GAME_TYPE_CONST"
            :key="t"
          >
            <KunCheckBox
              :model-value="selectedTypes.includes(t)"
              @update:model-value="(checked) => toggleGameType(checked, t)"
              :label="KUN_GALGAME_RATING_GAME_TYPE_MAP[t]"
            />
            <p class="text-default-700 text-sm">
              {{ KUN_GALGAME_RATING_GAME_TYPE_DESCRIPTION_MAP[t] }}
            </p>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <KunSelect
          :options="recommendOptions"
          v-model="recommend"
          label="推荐程度"
        />
        <KunSelect
          :options="playStatusOptions"
          v-model="playStatus"
          label="游玩状态"
          placeholder="请选择游玩状态"
        />
        <KunSelect
          :options="spoilerOptions"
          v-model="spoilerLevel"
          label="剧透程度"
        />
      </div>

      <KunTextarea
        v-model="shortSummary"
        label="短评 (可选, 请勿包含 R18 内容, 您的短评在编写的同时会被自动保存在本地, 关掉面板不会丢失)"
        :rows="4"
        placeholder="「 枯れない世界と終わる花 」莲真的是我遇见最好的女孩子, 想和她结婚天长地久恩恩爱爱百年好合, 想和她翻云覆雨、干湿分离、水乳交融、干柴烈火, 日日夜夜每时每刻每个开心难过七滋八味苦辣麻香咸的日子里都在想念、挚爱、铭记着莲, 你们懂这种心情吗！！！这已经不是爱了, 这种东西叫本能, 我只是本能的爱着这个孩子而已, 我无法控制自己的本能, 我已经病入膏肓了, 病名为莲。"
        :maxlength="1314"
        :show-char-count="true"
        auto-grow
      />

      <div v-if="showAdvanced" class="pt-2">
        <GalgameRatingAdvanced v-model="dims" />
      </div>

      <div class="flex items-center justify-between">
        <KunButton variant="flat" @click="showAdvanced = !showAdvanced">
          {{ showAdvanced ? '收起高级评分' : '展开高级评分' }}
        </KunButton>
        <div class="flex items-center gap-2">
          <KunButton variant="light" color="default" @click="close">
            取消
          </KunButton>
          <KunButton :loading="isSubmitting" @click="submit">
            {{ isEditing ? '更新' : '提交' }}
          </KunButton>
        </div>
      </div>

      <p class="text-default-500 text-sm">
        如需长评, 建议前往话题区发布话题后将链接放在短评中。评分可较为随意填写,
        我们的算法会帮您摆平一切!
      </p>
    </div>
  </KunModal>
</template>
