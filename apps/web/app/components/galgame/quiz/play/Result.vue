<script setup lang="ts">
import { rateGalgameQuizQualitySchema } from '~/validations/galgame-quiz'
import { settle } from '#shared/utils/api/problem'
import type {
  Quiz,
  QuizAnswer,
  QuizQuality,
  QuizSolution
} from '#shared/utils/api/schemas'
import { contentPlainText } from '~/utils/contentPlainText'

const props = defineProps<{
  quiz: Quiz
  answer: QuizAnswer | null
  solution: QuizSolution
}>()

const emits = defineEmits<{ rated: [result: QuizQuality] }>()

const api = useApiClient()

const optionRows = computed(() => {
  if (props.quiz.quiz_type !== 'single' && props.quiz.quiz_type !== 'multiple') {
    return []
  }
  const chosen = props.answer?.submission?.choice_indexes ?? []
  const correct = new Set(props.solution.correct_choice_indexes)
  return props.quiz.choices.map((text, i) => ({
    text,
    correct: correct.has(i),
    chosen: chosen.includes(i)
  }))
})

const judge = computed(() => {
  if (props.quiz.quiz_type !== 'judge') return null
  return {
    answer: props.solution.is_statement_true,
    mine: props.answer?.submission?.is_statement_true
  }
})

const hasExplanation = computed(
  () => contentPlainText(props.solution.explanation).trim().length > 0
)

const rowClass = (correct: boolean, chosen: boolean) => {
  if (correct) return 'border-success/60 bg-success/10 text-success'
  if (chosen) return 'border-danger/60 bg-danger/10 text-danger'
  return 'border-default-200 text-default-600'
}

const quality = ref(props.quiz.viewer?.quality_rating ?? 0)
const isRating = ref(false)

watch(
  () => props.quiz.viewer?.quality_rating,
  (v) => {
    quality.value = v ?? 0
  }
)

const submitQuality = async () => {
  if (!requireLogin()) return
  const body = { rating: quality.value }
  const valid = useKunSchemaValidator(rateGalgameQuizQualitySchema, body)
  if (!valid) return

  isRating.value = true
  const res = await settle(
    api.PUT('/quizzes/{quiz_id}/quality-rating', {
      params: { path: { quiz_id: props.quiz.id } },
      body
    })
  )
  isRating.value = false
  if (!res.ok) {
    reportProblem(res.problem)
    return
  }
  useMessage('评分成功', 'success')
  emits('rated', res.data)
}
</script>

<template>
  <div class="space-y-4">
    <KunInfo
      v-if="answer?.is_correct === true"
      color="success"
      icon="lucide:circle-check"
      title="回答正确"
    />
    <KunInfo
      v-else-if="answer?.is_correct === false"
      color="danger"
      icon="lucide:circle-x"
      title="回答错误"
    />
    <KunInfo
      v-else
      color="secondary"
      icon="lucide:book-open"
      title="本题答案"
    />

    <div v-if="optionRows.length" class="space-y-2">
      <div
        v-for="(row, i) in optionRows"
        :key="i"
        class="flex items-center justify-between gap-2 rounded-lg border px-3 py-2 text-sm"
        :class="rowClass(row.correct, row.chosen)"
      >
        <span class="break-words whitespace-pre-wrap">{{ row.text }}</span>
        <span class="flex shrink-0 items-center gap-2">
          <KunChip v-if="row.chosen" size="sm" variant="light">你的选择</KunChip>
          <KunIcon v-if="row.correct" name="lucide:check" />
        </span>
      </div>
    </div>

    <div v-else-if="judge" class="flex flex-wrap items-center gap-3 text-sm">
      <KunChip color="success" variant="flat">
        正确答案: {{ judge.answer ? '正确' : '错误' }}
      </KunChip>
      <KunChip
        v-if="judge.mine !== undefined && judge.mine !== null"
        :color="judge.mine === judge.answer ? 'success' : 'danger'"
        variant="light"
      >
        你的回答: {{ judge.mine ? '正确' : '错误' }}
      </KunChip>
    </div>

    <div
      v-if="hasExplanation"
      class="bg-default-100 rounded-lg px-3 py-2 text-sm"
    >
      <p class="text-default-500 mb-1 flex items-center gap-1">
        <KunIcon name="lucide:lightbulb" />解析
      </p>
      <ContentDocument :document="solution.explanation" compact />
    </div>

    <KunDivider />

    <div v-if="quiz.viewer?.has_answered" class="space-y-2">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <label class="text-sm font-medium">给这道题的质量打分 (1-10)</label>
        <span
          v-if="quiz.quality_count > 0 && quiz.quality_average != null"
          class="text-default-500 text-sm"
        >
          平均 {{ quiz.quality_average }} 分 · {{ quiz.quality_count }} 人评分
        </span>
      </div>
      <div class="flex flex-wrap items-center gap-4">
        <KunRating v-model="quality" :max="10" aria-label="quiz-quality" />
        <span class="text-warning-500 text-2xl font-bold">{{ quality }}</span>
        <KunButton
          :loading="isRating"
          :disabled="quality < 1"
          size="sm"
          @click="submitQuality"
        >
          {{ quiz.viewer?.quality_rating ? '更新评分' : '提交评分' }}
        </KunButton>
      </div>
    </div>
  </div>
</template>
