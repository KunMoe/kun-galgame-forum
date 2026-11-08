<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import type {
  CollectionReportCategory,
  CollectionReportCreate
} from '#shared/utils/api/schemas'

const props = defineProps<{
  modelValue: boolean
  collectionId: string
  workId?: string
  workName?: string
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const api = useApiClient()
const reportKey = useIdempotencyKey()

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const category = ref<CollectionReportCategory>('spam')
const note = ref('')
const submitting = ref(false)

watch(isOpen, (open) => {
  if (open) {
    category.value = 'spam'
    note.value = ''
  }
})

const submit = async () => {
  submitting.value = true
  const body: CollectionReportCreate = {
    collection_id: props.collectionId,
    report_category: category.value
  }
  if (note.value.trim()) {
    body.note = note.value.trim()
  }
  if (props.workId) {
    body.work_id = props.workId
  }
  const result = await settle(
    api.POST('/collection-reports', {
      params: {
        header: {
          'Idempotency-Key': reportKey.take('/collection-reports', body)
        }
      },
      body
    })
  )
  submitting.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  reportKey.clear()
  useMessage('举报已提交，感谢你的反馈', 'success')
  isOpen.value = false
}
</script>

<template>
  <KunModal
    v-model="isOpen"
    inner-class-name="max-w-md"
    :title="workId ? '举报这条备注' : '举报收藏夹'"
    :description="workName ? `「${workName}」上的备注` : undefined"
  >
    <form class="space-y-4" @submit.prevent>
      <KunSelect
        v-model="category"
        label="举报原因"
        :options="[...COLLECTION_REPORT_CATEGORIES]"
      />
      <KunTextarea
        v-model="note"
        label="补充说明 (可选)"
        :rows="3"
        :maxlength="500"
        show-char-count
      />

      <div class="flex justify-end gap-3">
        <KunButton variant="light" @click="isOpen = false">取消</KunButton>
        <KunButton color="danger" :loading="submitting" @click="submit">
          提交举报
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
