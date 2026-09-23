<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_CONST,
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_TYPE_ICON_MAP,
  KUN_QUIZ_TYPE_DESCRIPTION_MAP,
  KUN_QUIZ_CATEGORY_CONST,
  KUN_QUIZ_CATEGORY_MAP,
  KUN_QUIZ_SPOILER_CONST,
  KUN_QUIZ_SPOILER_MAP,
  KUN_QUIZ_PROMPT_MAX,
  kunQuizDifficultyLabel,
  kunQuizDifficultyColor
} from '~/constants/galgame-quiz'
import { createGalgameQuizSchema } from '~/validations/galgame-quiz'
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import type {
  Quiz,
  QuizCategory,
  QuizCreate,
  QuizPatch,
  QuizSource,
  QuizSpoilerLevel,
  QuizType,
  WorkRef
} from '#shared/utils/api/schemas'
import { coerceQuizType } from '~/store/modules/edit/quiz'
import type { RecentQuizGalgame } from '~/store/modules/edit/quizGalgame'

type QuizEditorValue = {
  choices: string[]
  correct_choice_indexes: number[]
  is_statement_true: boolean | null
}

const props = defineProps<{
  workId?: string
  source?: QuizSource | null
  works?: WorkRef[]
}>()

const emits = defineEmits<{
  published: [quiz: Quiz]
  updated: []
  cancel: []
}>()

const api = useApiClient()
const createKey = useIdempotencyKey()
const workName = useWorkName()

const category = ref<QuizCategory>('trivia')
const type = ref<QuizType>('single')
const difficulty = ref(3)
const spoilerLevel = ref<QuizSpoilerLevel>('none')
const question = ref('')
const description = ref('')
const explanation = ref('')
const pickedWorkIds = ref<string[]>([])
const hideGalgame = ref(false)
const showExplanation = ref(false)
const isSubmitting = ref(false)

const editorRef = ref<{
  getValue: () => QuizEditorValue
  validate: () => string | null
  reset: () => void
  load: (content: QuizEditorValue) => void
} | null>(null)

const initialSelected = ref<RecentQuizGalgame[]>([])
const isEditing = computed(() => !!props.source)
const original = ref<QuizSource | null>(null)

const worksOf = (d: QuizSource): RecentQuizGalgame[] => {
  const byId = new Map((props.works ?? []).map((w) => [w.id, w]))
  return d.work_ids.map((id) => {
    const work = byId.get(id)
    if (work) {
      return {
        id: work.id,
        name: workName(work),
        coverUrl: work.cover?.url,
        thumbhash: work.cover?.thumbhash ?? undefined,
        isNsfw: work.is_nsfw
      }
    }
    return { id, name: `#${id}` }
  })
}

watch(
  () => props.source,
  async (d) => {
    if (!d) return
    original.value = d
    category.value = d.quiz_category
    type.value = d.quiz_type
    difficulty.value = d.difficulty
    spoilerLevel.value = d.spoiler_level
    question.value = d.prompt_text
    description.value = d.description_markdown
    explanation.value = d.explanation_markdown
    showExplanation.value = !!d.explanation_markdown
    hideGalgame.value = d.is_work_hidden
    pickedWorkIds.value = [...d.work_ids]
    initialSelected.value = worksOf(d)
    await nextTick()
    if (!editorRef.value) await nextTick()
    editorRef.value?.load({
      choices: d.choices,
      correct_choice_indexes: d.correct_choice_indexes,
      is_statement_true: d.is_statement_true
    })
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
    if (props.source || isRestoring.value) return
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

const onContentChange = (content: QuizEditorValue) => {
  if (props.source || isRestoring.value) return
  persist.content = content
}

onMounted(async () => {
  if (props.source) return
  isRestoring.value = true
  category.value = persist.category
  type.value = coerceQuizType(persist.type)
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
  if (saved && Array.isArray(saved.choices)) {
    editorRef.value?.load(saved as QuizEditorValue)
  }
  await nextTick()
  isRestoring.value = false
})

const typeGroupOptions = computed(() =>
  KUN_QUIZ_TYPE_CONST.map((t) => ({
    value: t,
    label: KUN_QUIZ_TYPE_MAP[t],
    icon: KUN_QUIZ_TYPE_ICON_MAP[t],
    disabled: isEditing.value
  }))
)
const typeSelection = computed<QuizType[]>({
  get: () => [type.value],
  set: (arr) => {
    if (isEditing.value) return
    const last = arr[arr.length - 1]
    if (last) type.value = last
  }
})

const categoryOptions = KUN_QUIZ_CATEGORY_CONST.map((c) => ({
  value: c,
  label: KUN_QUIZ_CATEGORY_MAP[c]
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
  label: KUN_QUIZ_SPOILER_MAP[s]
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

const workIds = () =>
  props.workId ? [props.workId] : pickedWorkIds.value.map(String)

const createPayload = (): QuizCreate => {
  const value = editorRef.value?.getValue() ?? {
    choices: [],
    correct_choice_indexes: [],
    is_statement_true: null
  }
  const payload: QuizCreate = {
    quiz_type: type.value,
    quiz_category: category.value,
    difficulty: difficulty.value,
    spoiler_level: spoilerLevel.value,
    prompt_text: question.value,
    description_markdown: description.value,
    explanation_markdown: explanation.value,
    is_work_hidden: hideGalgame.value,
    work_ids: workIds()
  }
  if (type.value === 'judge') {
    payload.is_statement_true = value.is_statement_true
    payload.correct_choice_indexes = []
  } else {
    payload.choices = value.choices
    payload.correct_choice_indexes = value.correct_choice_indexes
  }
  return payload
}

const sameArray = <T,>(a: T[], b: T[]) => JSON.stringify(a) === JSON.stringify(b)

const patchFromChanges = (): QuizPatch => {
  const before = original.value
  const body: QuizPatch = {}
  if (!before) return body
  const value = editorRef.value?.getValue() ?? {
    choices: [],
    correct_choice_indexes: [],
    is_statement_true: null
  }
  if (question.value !== before.prompt_text) body.prompt_text = question.value
  if (description.value !== before.description_markdown) {
    body.description_markdown = description.value
  }
  if (explanation.value !== before.explanation_markdown) {
    body.explanation_markdown = explanation.value
  }
  if (category.value !== before.quiz_category) {
    body.quiz_category = category.value
  }
  if (difficulty.value !== before.difficulty) body.difficulty = difficulty.value
  if (spoilerLevel.value !== before.spoiler_level) {
    body.spoiler_level = spoilerLevel.value
  }
  if (hideGalgame.value !== before.is_work_hidden) {
    body.is_work_hidden = hideGalgame.value
  }
  const ids = workIds()
  if (!sameArray(ids, before.work_ids)) body.work_ids = ids
  if (before.quiz_type !== 'judge' && !sameArray(value.choices, before.choices)) {
    body.choices = value.choices
  }
  if (!sameArray(value.correct_choice_indexes, before.correct_choice_indexes)) {
    body.correct_choice_indexes = value.correct_choice_indexes
  }
  if (before.quiz_type === 'judge') {
    if (value.is_statement_true !== before.is_statement_true) {
      body.is_statement_true = value.is_statement_true
    }
  }
  return body
}

const submit = async () => {
  const contentError = editorRef.value?.validate()
  if (contentError) {
    useMessage(contentError, 'warn')
    return
  }

  if (isEditing.value && props.source) {
    const body = patchFromChanges()
    const valid = useKunSchemaValidator(createGalgameQuizSchema, {
      ...createPayload(),
      ...body
    })
    if (!valid) return
    if (Object.keys(body).length === 0) {
      useMessage('已保存修改', 'success')
      emits('updated')
      return
    }
    isSubmitting.value = true
    const ok = await settle(
      api.PATCH('/quizzes/{quiz_id}', {
        params: { path: { quiz_id: props.source.quiz_id } },
        body
      })
    )
    isSubmitting.value = false
    if (!ok.ok) {
      reportProblem(ok.problem)
      return
    }
    useMessage('已保存修改', 'success')
    emits('updated')
    return
  }

  const payload = createPayload()
  const valid = useKunSchemaValidator(createGalgameQuizSchema, payload)
  if (!valid) return

  isSubmitting.value = true
  const res = await settle(
    api.POST('/quizzes', {
      params: {
        header: { 'Idempotency-Key': createKey.take('/quizzes', payload) }
      },
      body: payload
    })
  )
  isSubmitting.value = false
  if (!res.ok) {
    reportProblem(res.problem)
    return
  }
  createKey.clear()
  useMessage('出题成功', 'success')
  resetForm()
  persist.reset()
  emits('published', res.data)
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
      <label class="text-sm font-medium">题型</label>
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
      :maxlength="KUN_QUIZ_PROMPT_MAX"
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
