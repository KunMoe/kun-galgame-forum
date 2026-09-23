<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
  workId?: number
  editData?: QuizEditData | null
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  onPublished: [quiz: GalgameQuizCard]
  onUpdated: []
}>()

const close = () => emits('update:modelValue', false)
</script>

<template>
  <KunModal
    :model-value="modelValue"
    inner-class-name="max-w-[720px] w-[90vw]"
    :is-dismissable="false"
    @update:model-value="(v) => emits('update:modelValue', v)"
  >
    <GalgameQuizForm
      :work-id="props.workId"
      :edit-data="props.editData"
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
