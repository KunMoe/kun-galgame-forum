<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  modelValue: boolean
  collectionId: string
  workId: string
  workName: string
  note: string
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [workId: string, note: string | null]
}>()

const api = useApiClient()

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const draft = ref('')
const error = ref('')
const submitting = ref(false)

watch(isOpen, (open) => {
  if (open) {
    draft.value = props.note
    error.value = ''
  }
})

const submit = async () => {
  error.value = ''
  submitting.value = true
  const result = await settle(
    api.PATCH('/collections/{collection_id}/works/{work_id}', {
      params: {
        path: { collection_id: props.collectionId, work_id: props.workId }
      },
      body: { note: draft.value }
    })
  )
  submitting.value = false
  if (!result.ok) {
    error.value = collectionFieldErrors(result.problem, '/note')['/note'] ?? ''
    if (!error.value) {
      reportProblem(result.problem)
    }
    return
  }
  emits('saved', props.workId, result.data.note ?? null)
  isOpen.value = false
}
</script>

<template>
  <KunModal
    v-model="isOpen"
    inner-class-name="max-w-lg"
    title="编辑备注"
    :description="workName"
  >
    <form class="space-y-4" @submit.prevent>
      <KunTextarea
        v-model="draft"
        label="备注"
        placeholder="为什么推荐它、适合什么样的玩家…留空即删除备注"
        :rows="4"
        auto-grow
        :maxlength="COLLECTION_NOTE_MAX"
        show-char-count
        :error="error"
      />

      <div class="flex justify-end gap-3">
        <KunButton variant="light" @click="isOpen = false">取消</KunButton>
        <KunButton color="primary" :loading="submitting" @click="submit">
          保存
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
