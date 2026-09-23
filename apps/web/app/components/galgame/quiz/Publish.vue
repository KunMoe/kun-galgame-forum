<script setup lang="ts">
import type { Quiz, QuizSource, WorkRef } from '#shared/utils/api/schemas'

const props = defineProps<{
  modelValue: boolean
  workId?: string
  source?: QuizSource | null
  works?: WorkRef[]
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  onPublished: [quiz: Quiz]
  onUpdated: []
}>()

const close = () => emits('update:modelValue', false)
</script>

<template>
  <KunModal
    :model-value="modelValue"
    :aria-label="source ? '编辑题目' : '出题'"
    inner-class-name="max-w-[720px] w-[90vw]"
    :is-dismissable="false"
    @update:model-value="(v) => emits('update:modelValue', v)"
  >
    <GalgameQuizForm
      :work-id="props.workId"
      :source="props.source"
      :works="props.works"
      @published="
        (q) => {
          emits('onPublished', q)
          close()
        }
      "
      @updated="
        () => {
          emits('onUpdated')
          close()
        }
      "
      @cancel="close"
    />
  </KunModal>
</template>
