<script setup lang="ts">
import type { KunRadioOption, KunCheckBoxGroupOption } from '@kungal/ui-vue'
import type { QuizSubmission, QuizType } from '#shared/utils/api/schemas'

const props = defineProps<{
  type: QuizType
  choices: string[]
}>()

const singleValue = ref(-1)
const multiValues = ref<number[]>([])
const judgeValue = ref<'true' | 'false' | ''>('')

const choiceOptions = computed<KunRadioOption<number>[]>(() =>
  props.choices.map((o, i) => ({ value: i, label: o }))
)
const checkChoiceOptions = computed<KunCheckBoxGroupOption<number>[]>(
  () => choiceOptions.value
)

const getSubmitted = (): QuizSubmission => {
  switch (props.type) {
    case 'single':
      return { choice_indexes: [singleValue.value] }
    case 'multiple':
      return {
        choice_indexes: [...multiValues.value].sort((a, b) => a - b)
      }
    case 'judge':
      return {
        choice_indexes: [],
        is_statement_true: judgeValue.value === 'true'
      }
  }
}

const validate = (): string | null => {
  switch (props.type) {
    case 'single':
      return singleValue.value >= 0 ? null : '请选择一个答案'
    case 'multiple':
      return multiValues.value.length > 0 ? null : '请至少选择一个答案'
    case 'judge':
      return judgeValue.value ? null : '请选择正确或错误'
  }
}

defineExpose({ getSubmitted, validate })
</script>

<template>
  <div class="space-y-3">
    <KunRadioGroup
      v-if="type === 'single'"
      v-model="singleValue"
      :options="choiceOptions"
      variant="card"
    />

    <KunCheckBoxGroup
      v-else-if="type === 'multiple'"
      v-model="multiValues"
      :options="checkChoiceOptions"
      variant="card"
    />

    <KunRadioGroup
      v-else-if="type === 'judge'"
      v-model="judgeValue"
      :options="[
        { value: 'true', label: '正确' },
        { value: 'false', label: '错误' }
      ]"
      orientation="horizontal"
      variant="card"
    />
  </div>
</template>
