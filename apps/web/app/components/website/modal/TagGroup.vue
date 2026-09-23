<script setup lang="ts">
import { websiteTagGroupFormSchema } from '~/validations/website'
import type { WebsiteTagGroupForm } from './types'

const props = defineProps<{
  modelValue: boolean
  initialData?: WebsiteTagGroupForm
  isEditing: boolean
  loading?: boolean
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [data: WebsiteTagGroupForm]
}>()

const emptyForm = (): WebsiteTagGroupForm => ({
  slug: '',
  label: '',
  description: '',
  sort_order: 0,
  is_multi_select: false
})

const isModalOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const formData = reactive<WebsiteTagGroupForm>(emptyForm())

watch(
  () => isModalOpen.value,
  (isOpen) => {
    if (isOpen) {
      Object.assign(formData, emptyForm(), props.initialData ?? {})
    }
  }
)

const handleSubmit = () => {
  const result = websiteTagGroupFormSchema.safeParse(formData)
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
    :aria-label="isEditing ? '编辑标签分组' : '创建标签分组'"
    inner-class-name="max-w-md"
  >
    <form @submit.prevent="handleSubmit">
      <h2 class="mb-6 text-xl font-bold">
        {{ isEditing ? '编辑标签分组' : '创建标签分组' }}
      </h2>

      <div class="space-y-4">
        <KunInput
          v-model="formData.slug"
          label="分组标识 (小写英文)"
          placeholder="performance"
          required
        />
        <KunInput
          v-model="formData.label"
          label="分组显示名"
          placeholder="网站性能"
          required
        />
        <KunInput
          v-model="formData.sort_order"
          label="排序 (数字越小越靠前)"
          type="number"
        />
        <KunTextarea
          v-model="formData.description"
          label="分组描述 (可选)"
          auto-grow
          show-char-count
          :maxlength="300"
        />
        <KunSwitch
          v-model="formData.is_multi_select"
          label="该分组可多选 (关闭则组内标签互斥, 只能选一个)"
        />
      </div>

      <div class="mt-6 flex justify-end gap-3">
        <KunButton
          variant="light"
          color="danger"
          :disabled="loading"
          @click="isModalOpen = false"
        >
          取消
        </KunButton>
        <KunButton type="submit" color="primary" :loading="loading">
          {{ isEditing ? '保存更改' : '创建' }}
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
