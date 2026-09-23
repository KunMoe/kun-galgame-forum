<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_CONST,
  KUN_QUIZ_ENABLED_TYPE_CONST,
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_TYPE_ICON_MAP,
  KUN_QUIZ_TYPE_DESCRIPTION_MAP,
  KUN_QUIZ_CATEGORY_CONST,
  KUN_QUIZ_CATEGORY_MAP,
  KUN_QUIZ_SPOILER_CONST,
  KUN_QUIZ_SPOILER_MAP,
  kunQuizDifficultyLabel,
  kunQuizDifficultyColor
} from '~/constants/galgame-quiz'
import { createGalgameQuizSchema } from '~/validations/galgame-quiz'

const props = defineProps<{
  workId?: number
  editData?: QuizEditData | null
}>()

const emits = defineEmits<{
  published: [quiz: GalgameQuizCard]
  updated: []
  cancel: []
}>()

const category = ref<QuizCategory>('trivia')
const type = ref<QuizType>('single')
const difficulty = ref(3)
const spoilerLevel = ref<QuizSpoilerLevel>('none')
const question = ref('')
const description = ref('')
const explanation = ref('')
const pickedWorkIds = ref<number[]>([])
const hideGalgame = ref(false)
const showExplanation = ref(false)
const isSubmitting = ref(false)

const editorRef = ref<{
  getContent: () => Record<string, unknown>
  validate: () => string | null
  reset: () => void
  load: (content: Record<string, unknown>) => void
} | null>(null)

const initialSelected = ref<{ id: number; name: string }[]>([])
const isEditing = computed(() => !!props.editData)

watch(
  () => props.editData,
  async (d) => {
    if (!d) return
    category.value = d.category
    type.value = d.type
    difficulty.value = d.difficulty
    spoilerLevel.value = d.spoiler_level
    question.value = d.question
    description.value = d.description
    explanation.value = d.explanation
    showExplanation.value = !!d.explanation
    hideGalgame.value = d.hide_galgame
    pickedWorkIds.value = [...d.galgame_ids]
    initialSelected.value = d.galgames.map((g) => ({
      id: g.id,
      name: g.name || `#${g.id}`
    }))
    await nextTick()
    if (!editorRef.value) await nextTick()
    editorRef.value?.load(d.content as unknown as Record<string, unknown>)
  },
  { immediate: true }
)

const persist = usePersistEditQuizStore()
const isRestoring = ref(false)
const editorKey = ref(0)

watch(
  [
    category,
    type,
    difficulty,
    spoilerLevel,
    question,
    description,
    explanation,
    showExplanation,
    hideGalgame
  ],
  () => {
    if (props.editData || isRestoring.value) return
    persist.category = category.value
    persist.type = type.value
    persist.difficulty = difficulty.value
    persist.spoilerLevel = spoilerLevel.value
    persist.question = question.value
    persist.description = description.value
    persist.explanation = explanation.value
    persist.showExplanation = showExplanation.value
    persist.hideGalgame = hideGalgame.value
  }
)

const onContentChange = (content: Record<string, unknown>) => {
  if (props.editData || isRestoring.value) return
  persist.content = content
}

onMounted(async () => {
  if (props.editData) return
  isRestoring.value = true
  category.value = persist.category
  type.value = persist.type
  difficulty.value = persist.difficulty
  spoilerLevel.value = persist.spoilerLevel
  question.value = persist.question
  description.value = persist.description
  explanation.value = persist.explanation
  showExplanation.value = persist.showExplanation
  hideGalgame.value = persist.hideGalgame
  if (persist.description) editorKey.value++
  const saved = persist.content
  await nextTick()
  if (!editorRef.value) await nextTick()
  if (saved && Object.keys(saved).length) {
    editorRef.value?.load(saved)
  }
  await nextTick()
  isRestoring.value = false
})

const typeGroupOptions = KUN_QUIZ_TYPE_CONST.map((t) => {
  const enabled = (KUN_QUIZ_ENABLED_TYPE_CONST as readonly string[]).includes(t)
  return {
    value: t,
    label: KUN_QUIZ_TYPE_MAP[t] ?? t,
    icon: KUN_QUIZ_TYPE_ICON_MAP[t] ?? 'lucide:circle',
    disabled: !enabled
  }
})
const typeSelection = computed<QuizType[]>({
  get: () => [type.value],
  set: (arr) => {
    const last = arr[arr.length - 1]
    if (last) type.value = last
  }
})

const categoryOptions = KUN_QUIZ_CATEGORY_CONST.map((c) => ({
  value: c,
  label: KUN_QUIZ_CATEGORY_MAP[c] ?? c
}))
const categorySelection = computed<QuizCategory[]>({
  get: () => [category.value],
  set: (arr) => {
    const last = arr[arr.length - 1]
    if (last) category.value = last
  }
})

const spoilerOptions = KUN_QUIZ_SPOILER_CONST.map((s) => ({
  value: s,
  label: KUN_QUIZ_SPOILER_MAP[s] ?? s
}))

const resetForm = () => {
  category.value = 'trivia'
  type.value = 'single'
  difficulty.value = 3
  spoilerLevel.value = 'none'
  question.value = ''
  description.value = ''
  explanation.value = ''
  pickedWorkIds.value = []
  hideGalgame.value = false
  initialSelected.value = []
  showExplanation.value = false
  editorRef.value?.reset()
}

const submit = async () => {
  const contentError = editorRef.value?.validate()
  if (contentError) {
    useMessage(contentError, 'warn')
    return
  }
  const content = editorRef.value?.getContent() ?? {}

  const body: Record<string, unknown> = {
    category: category.value,
    type: type.value,
    difficulty: difficulty.value,
    spoiler_level: spoilerLevel.value,
    question: question.value,
    description: description.value,
    content,
    explanation: explanation.value,
    hide_galgame: hideGalgame.value,
    galgame_ids: props.workId ? [props.workId] : pickedWorkIds.value
  }

  const valid = useKunSchemaValidator(createGalgameQuizSchema, body)
  if (!valid) {
    return
  }

  if (isEditing.value && props.editData) {
    body.quiz_id = props.editData.id
    isSubmitting.value = true
    const ok = await kunFetch<{ regraded: number }>(
      `/galgame-quiz/${props.editData.id}`,
      { method: 'PUT', body }
    )
    isSubmitting.value = false
    if (ok) {
      const regraded = ok.regraded ?? 0
      useMessage(
        regraded > 0
          ? `已保存修改，${regraded} 条作答已更正为正确并补发萌萌点`
          : '已保存修改',
        'success'
      )
      emits('updated')
    }
    return
  }

  isSubmitting.value = true
  const res = await kunFetch<GalgameQuizCard>('/galgame-quiz', {
    method: 'POST',
    body
  })
  isSubmitting.value = false
  if (res) {
    useMessage('出题成功', 'success')
    resetForm()
    persist.reset()
    emits('published', res)
  }
}
</script>

<template>
  <div class="space-y-5">
    <div class="space-y-1">
      <KunHeader :name="isEditing ? '编辑题目' : '出题'" scale="h3" />
      <p
        v-if="!isEditing"
        class="text-default-400 flex items-center gap-1 text-xs"
      >
        <KunIcon name="lucide:lollipop" />出题即得 2 萌萌点 · 题目被删除时扣除
      </p>
    </div>

    <KunInfo
      v-if="workId"
      color="primary"
      icon="lucide:link"
      title="已关联当前 Galgame"
      description="本题将关联到你当前所在的 Galgame"
    />
    <GalgameQuizGalgamePicker
      v-else
      v-model="pickedWorkIds"
      :initial-selected="initialSelected"
    />

    <div
      v-if="workId || pickedWorkIds.length"
      class="flex items-center justify-between gap-3"
    >
      <div>
        <label class="text-sm font-medium">隐藏关联作品</label>
        <p class="text-default-400 text-xs">
          开启后关联的 Galgame 会在用户作答后才揭晓, 适合「看图猜游戏」
        </p>
      </div>
      <KunSwitch v-model="hideGalgame" />
    </div>

    <div class="space-y-2">
      <div class="flex items-center gap-2">
        <label class="text-sm font-medium">题型</label>
        <span class="text-default-400 text-xs">填空、问答即将实装</span>
      </div>
      <KunCheckBoxGroup
        v-model="typeSelection"
        :options="typeGroupOptions"
        variant="pill"
        color="primary"
        size="sm"
        orientation="horizontal"
      />
      <p class="text-default-500 text-sm">
        {{ KUN_QUIZ_TYPE_DESCRIPTION_MAP[type] }}
      </p>
    </div>

    <KunTextarea
      v-model="question"
      label="题目"
      :rows="2"
      placeholder="例如: 《永不枯萎的世界与终结之花》中莲什么时候来过月经"
      :maxlength="2000"
      :show-char-count="true"
      auto-grow
    />
    <p class="text-default-400 text-xs">
      题干可用
      <code class="bg-default-100 rounded px-1">||剧透内容||</code>
      标记剧透, 标记内容会被打码, 点击后显示
    </p>

    <div class="space-y-2">
      <div>
        <label class="text-sm font-medium">题目描述（可选）</label>
        <p class="text-default-400 text-xs">
          支持 Markdown, 可上传图片作为线索（例如让玩家根据 CG 猜游戏）
        </p>
      </div>
      <KunMilkdownDualEditorProvider
        :key="editorKey"
        :value-markdown="description"
        placeholder="补充题目背景、线索图片等（可选）"
        @set-markdown="(val) => (description = val)"
      />
    </div>

    <GalgameQuizContentEditor
      ref="editorRef"
      :type="type"
      @change="onContentChange"
    />

    <KunDivider />

    <div class="space-y-4">
      <div class="space-y-2">
        <label class="text-sm font-medium">分类</label>
        <KunCheckBoxGroup
          v-model="categorySelection"
          :options="categoryOptions"
          variant="pill"
          color="primary"
          size="sm"
          orientation="horizontal"
        />
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="space-y-1">
          <div class="flex items-center justify-between">
            <label class="text-sm">难度</label>
            <KunChip
              :color="kunQuizDifficultyColor(difficulty)"
              variant="flat"
              size="sm"
            >
              {{ kunQuizDifficultyLabel(difficulty) }} · {{ difficulty }}
            </KunChip>
          </div>
          <KunSlider
            v-model="difficulty"
            :min="1"
            :max="10"
            :step="1"
            :color="kunQuizDifficultyColor(difficulty)"
          />
        </div>

        <KunSelect
          v-model="spoilerLevel"
          :options="spoilerOptions"
          label="剧透等级"
        />
      </div>
    </div>

    <KunTextarea
      v-if="showExplanation || explanation"
      v-model="explanation"
      label="解析（可选, 作答后展示）"
      :rows="2"
      placeholder="可以补充答案的解析、出处或冷知识"
      :maxlength="2000"
      :show-char-count="true"
      auto-grow
    />
    <KunButton v-else variant="light" size="sm" @click="showExplanation = true">
      <span class="flex items-center gap-1">
        <KunIcon name="lucide:plus" />添加解析（可选）
      </span>
    </KunButton>

    <div class="flex items-center justify-end gap-2 pt-1">
      <KunButton variant="light" color="default" @click="emits('cancel')">
        取消
      </KunButton>
      <KunButton :loading="isSubmitting" @click="submit">
        {{ isEditing ? '保存修改' : '发布题目' }}
      </KunButton>
    </div>
  </div>
</template>
