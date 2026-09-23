<script setup lang="ts">
import { KUN_TODO_PROJECTS, KUN_TODO_PROJECT_LABEL } from '~/constants/update'
import { todoSchema } from '~/validations/todo'
import type { TodoCreate } from '#shared/utils/api/schemas'

const props = defineProps<{
  modelValue: boolean
  initialData?: TodoCreate
  isEditing: boolean
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [data: TodoCreate]
}>()

const isModalOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const projectOptions = KUN_TODO_PROJECTS.map((project) => ({
  value: project,
  label: KUN_TODO_PROJECT_LABEL[project]
}))

const emptyForm = (): TodoCreate => ({ project: 'forum', text: '' })

const formData = reactive<TodoCreate>(emptyForm())

watch(
  () => isModalOpen.value,
  (isOpen) => {
    if (isOpen) {
      Object.assign(formData, props.initialData ?? emptyForm())
    }
  }
)

const handleSubmit = () => {
  const result = todoSchema.safeParse(formData)
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
        {{ isEditing ? '编辑待办' : '创建新待办' }}
      </h2>

      <div class="space-y-4">
        <KunSelect
          v-model="formData.project"
          :options="projectOptions"
          label="待办类型"
          required
        />
        <KunTextarea
          v-model="formData.text"
          label="待办内容 (1000 字符之内)"
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
