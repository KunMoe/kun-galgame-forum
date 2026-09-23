<script setup lang="ts">
import {
  KUN_UPDATE_LOG_CHANGE_TYPES,
  KUN_UPDATE_LOG_CHANGE_TYPE_LABEL
} from '~/constants/update'
import { updateLogSchema } from '~/validations/update-log'
import type { UpdateLogCreate } from '#shared/utils/api/schemas'

const props = defineProps<{
  modelValue: boolean
  initialData?: UpdateLogCreate
  isEditing: boolean
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [data: UpdateLogCreate]
}>()

const isModalOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const changeTypeOptions = KUN_UPDATE_LOG_CHANGE_TYPES.map((type) => ({
  label: KUN_UPDATE_LOG_CHANGE_TYPE_LABEL[type],
  value: type
}))

const emptyForm = (): UpdateLogCreate => ({
  change_type: 'feat',
  release_version: '',
  text: ''
})

const formData = reactive<UpdateLogCreate>(emptyForm())

watch(
  () => isModalOpen.value,
  (isOpen) => {
    if (isOpen) {
      Object.assign(formData, props.initialData ?? emptyForm())
    }
  }
)

const handleSubmit = () => {
  const result = updateLogSchema.safeParse(formData)
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }
  emits('submit', result.data)
}
</script>

<template>
  <KunModal
    :is-dismissable="false"
    v-model="isModalOpen"
    inner-class-name="max-w-xl"
  >
    <form @submit.prevent>
      <h2 class="mb-6 text-xl font-bold">
        {{ isEditing ? '编辑更新日志' : '创建新更新日志' }}
      </h2>

      <div class="space-y-4">
        <KunInput v-model="formData.release_version" label="版本号" required />
        <KunSelect
          v-model="formData.change_type"
          :options="changeTypeOptions"
          label="日志类型"
          required
        />
        <KunTextarea
          v-model="formData.text"
          label="更新内容 (1000 字符之内)"
          :rows="5"
        />
      </div>

      <div class="mt-6 flex justify-end gap-3">
        <KunButton variant="light" color="danger" @click="isModalOpen = false">
          取消
        </KunButton>
        <KunButton @click="handleSubmit" color="primary">
          {{ isEditing ? '保存更改' : '创建' }}
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
