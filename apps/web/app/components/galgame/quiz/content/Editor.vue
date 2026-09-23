<script setup lang="ts">
import type { QuizType } from '#shared/utils/api/schemas'
import {
  KUN_QUIZ_CHOICE_LIMIT,
  KUN_QUIZ_CHOICE_MAX
} from '~/constants/galgame-quiz'

type QuizEditorValue = {
  choices: string[]
  correct_choice_indexes: number[]
  is_statement_true: boolean | null
}

const props = defineProps<{ type: QuizType }>()
const emits = defineEmits<{ change: [content: QuizEditorValue] }>()

const options = ref<string[]>(['', ''])
const singleAnswer = ref(0)
const multiAnswers = ref<number[]>([])
const judgeAnswer = ref<'true' | 'false'>('true')

const reset = () => {
  options.value = ['', '']
  singleAnswer.value = 0
  multiAnswers.value = []
  judgeAnswer.value = 'true'
}
watch(() => props.type, reset)

const addOption = () => {
  if (options.value.length >= KUN_QUIZ_CHOICE_LIMIT) return
  options.value.push('')
}
const removeOption = (i: number) => {
  if (options.value.length <= 2) return
  options.value.splice(i, 1)
  multiAnswers.value = multiAnswers.value
    .filter((a) => a !== i)
    .map((a) => (a > i ? a - 1 : a))
  if (singleAnswer.value >= options.value.length) singleAnswer.value = 0
}

const isCorrect = (i: number) =>
  props.type === 'single'
    ? singleAnswer.value === i
    : multiAnswers.value.includes(i)
const setCorrect = (i: number, checked: boolean) => {
  if (props.type === 'single') {
    singleAnswer.value = i
  } else if (checked) {
    if (!multiAnswers.value.includes(i)) multiAnswers.value.push(i)
  } else {
    multiAnswers.value = multiAnswers.value.filter((x) => x !== i)
  }
}
const correctLabel = computed(() =>
  props.type === 'single' ? '正确答案' : '已选'
)
const toggleCorrect = (i: number) =>
  setCorrect(i, props.type === 'single' ? true : !isCorrect(i))

const getValue = (): QuizEditorValue => {
  if (props.type === 'judge') {
    return {
      choices: [],
      correct_choice_indexes: [],
      is_statement_true: judgeAnswer.value === 'true'
    }
  }
  return {
    choices: options.value.map((o) => o.trim()),
    correct_choice_indexes:
      props.type === 'single'
        ? [singleAnswer.value]
        : [...multiAnswers.value].sort((a, b) => a - b),
    is_statement_true: null
  }
}

watch(
  [options, singleAnswer, multiAnswers, judgeAnswer, () => props.type],
  () => emits('change', getValue()),
  { deep: true }
)

const validate = (): string | null => {
  if (props.type === 'judge') return null
  const opts = options.value.map((o) => o.trim())
  if (opts.length < 2) return '至少需要 2 个选项'
  if (opts.length > KUN_QUIZ_CHOICE_LIMIT) return '最多 20 个选项'
  if (opts.some((o) => !o)) return '选项内容不能为空'
  if (opts.some((o) => o.length > KUN_QUIZ_CHOICE_MAX))
    return '选项长度不能超过 200 字'
  if (new Set(opts).size !== opts.length) return '选项不能重复'
  if (props.type === 'single' && singleAnswer.value >= opts.length)
    return '请标记正确答案'
  if (props.type === 'multiple' && multiAnswers.value.length === 0)
    return '请至少标记一个正确答案'
  return null
}

const load = (value: QuizEditorValue) => {
  if (props.type === 'judge') {
    judgeAnswer.value = value.is_statement_true ? 'true' : 'false'
    return
  }
  options.value = value.choices?.length ? [...value.choices] : ['', '']
  if (props.type === 'single') {
    singleAnswer.value =
      typeof value.correct_choice_indexes?.[0] === 'number'
        ? value.correct_choice_indexes[0]
        : 0
    return
  }
  multiAnswers.value = Array.isArray(value.correct_choice_indexes)
    ? [...value.correct_choice_indexes]
    : []
}

defineExpose({ getValue, validate, reset, load })
</script>

<template>
  <div class="space-y-3">
    <template v-if="type === 'single' || type === 'multiple'">
      <div class="flex items-center justify-between">
        <label class="text-sm font-medium">选项</label>
        <span class="text-default-400 text-xs">
          点「设为正确答案」标记（{{
            type === 'single' ? '单选题唯一' : '多选题可多个'
          }}，2–20 个）
        </span>
      </div>

      <div class="space-y-2">
        <div
          v-for="(_, i) in options"
          :key="i"
          :class="
            cn(
              'flex items-center gap-2 rounded-lg px-2 py-1.5 transition-colors',
              isCorrect(i) && 'bg-success/10 ring-success/40 ring-1 ring-inset'
            )
          "
        >
          <KunButton
            :variant="isCorrect(i) ? 'flat' : 'light'"
            :color="isCorrect(i) ? 'success' : 'default'"
            size="sm"
            class-name="w-28 shrink-0 justify-start"
            :aria-pressed="isCorrect(i)"
            @click="toggleCorrect(i)"
          >
            <span class="flex items-center gap-1">
              <KunIcon
                :name="isCorrect(i) ? 'lucide:circle-check' : 'lucide:circle'"
              />
              {{ isCorrect(i) ? correctLabel : '设为正确' }}
            </span>
          </KunButton>

          <KunInput
            v-model="options[i]"
            :placeholder="`选项 ${i + 1}`"
            :maxlength="KUN_QUIZ_CHOICE_MAX"
            class-name="grow"
          />

          <KunButton
            :is-icon-only="true"
            variant="light"
            color="danger"
            size="sm"
            :disabled="options.length <= 2"
            @click="removeOption(i)"
          >
            <KunIcon name="lucide:trash-2" />
          </KunButton>
        </div>
      </div>

      <KunButton
        variant="light"
        size="sm"
        :disabled="options.length >= KUN_QUIZ_CHOICE_LIMIT"
        @click="addOption"
      >
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:plus" />添加选项
        </span>
      </KunButton>
    </template>

    <template v-else-if="type === 'judge'">
      <label class="text-sm font-medium">正确答案</label>
      <div class="grid grid-cols-2 gap-3">
        <button
          type="button"
          class="flex items-center justify-center gap-2 rounded-xl border-2 py-5 text-lg font-medium transition-colors"
          :class="
            judgeAnswer === 'true'
              ? 'border-success bg-success/10 text-success'
              : 'border-default-200 text-default-500 hover:border-default-300'
          "
          @click="judgeAnswer = 'true'"
        >
          <KunIcon name="lucide:circle-check" />正确
        </button>
        <button
          type="button"
          class="flex items-center justify-center gap-2 rounded-xl border-2 py-5 text-lg font-medium transition-colors"
          :class="
            judgeAnswer === 'false'
              ? 'border-danger bg-danger/10 text-danger'
              : 'border-default-200 text-default-500 hover:border-default-300'
          "
          @click="judgeAnswer = 'false'"
        >
          <KunIcon name="lucide:circle-x" />错误
        </button>
      </div>
    </template>
  </div>
</template>
