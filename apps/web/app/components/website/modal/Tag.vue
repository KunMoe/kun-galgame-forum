<script setup lang="ts">
import { websiteTagFormSchema } from '~/validations/website'
import type { WebsiteTagForm } from './types'

const props = defineProps<{
  modelValue: boolean
  initialData?: WebsiteTagForm
  isEditing: boolean
  loading?: boolean
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [data: WebsiteTagForm]
}>()

const { data: groups } = useWebsiteTagGroups()
const groupOptions = computed(() =>
  (groups.value ?? []).map((group) => ({
    value: group.id,
    label: group.label || group.slug
  }))
)

const emptyForm = (): WebsiteTagForm => ({
  slug: '',
  label: '',
  level: 0,
  description: '',
  website_tag_group_id: null
})

const isModalOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const formData = reactive<WebsiteTagForm>(emptyForm())

watch(
  () => isModalOpen.value,
  (isOpen) => {
    if (isOpen) {
      Object.assign(formData, emptyForm(), props.initialData ?? {})
    }
  }
)

const handleSubmit = () => {
  const result = websiteTagFormSchema.safeParse(formData)
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
    :aria-label="isEditing ? '编辑标签' : '创建新标签'"
    inner-class-name="max-w-md"
  >
    <form @submit.prevent="handleSubmit">
      <h2 class="mb-6 text-xl font-bold">
        {{ isEditing ? '编辑标签' : '创建新标签' }}
      </h2>

      <div class="space-y-4">
        <KunInput
          v-model="formData.slug"
          label="标签标识 (URL 用, 小写英文)"
          placeholder="performance0"
          required
        />
        <KunInput
          v-model="formData.label"
          label="标签显示名"
          placeholder="网站访问速度极快"
          required
        />
        <KunSelect
          v-model="formData.website_tag_group_id"
          label="所属分组"
          :options="groupOptions"
        />
        <KunInput
          v-model="formData.level"
          label="标签价值 (-100 ~ 20)"
          type="number"
          required
        />
        <KunTextarea
          v-model="formData.description"
          label="标签描述 (300 字符之内)"
          auto-grow
          show-char-count
          :maxlength="300"
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
