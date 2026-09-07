<script setup lang="ts">
import { createCollectionSchema } from '~/validations/collection'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  collectionId?: number
  initial?: {
    name: string
    description: string
    visibility: CollectionVisibility
  }
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [newId?: number]
}>()

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const visibilityOptions = [
  { value: 'public', label: '公开 · 所有人可见' },
  { value: 'private', label: '私密 · 仅自己可见' }
] as const

const form = reactive<{
  name: string
  description: string
  visibility: CollectionVisibility
}>({
  name: '',
  description: '',
  visibility: 'public'
})
const submitting = ref(false)

watch(
  () => isOpen.value,
  (open) => {
    if (!open) {
      return
    }
    submitting.value = false
    form.name = props.initial?.name ?? ''
    form.description = props.initial?.description ?? ''
    form.visibility = props.initial?.visibility ?? 'public'
  }
)

const submit = async () => {
  const payload = {
    name: form.name.trim(),
    description: form.description,
    visibility: form.visibility,
  }

  const result = createCollectionSchema.safeParse(payload)
  if (!result.success) {
    const issue = JSON.parse(result.error.message)[0]
    useMessage(formatKunZodIssue(issue), 'warn')
    return
  }

  submitting.value = true
  if (props.mode === 'create') {
    const id = await kunFetch<number>('/galgame/collection', {
      method: 'POST',
      body: result.data
    })
    submitting.value = false
    if (id === null || id === undefined) {
      return
    }
    useMessage(10566, 'success')
    emits('saved', id)
  } else {
    const ok = await kunFetch<string>(
      `/galgame/collection/${props.collectionId}`,
      { method: 'PATCH', body: result.data }
    )
    submitting.value = false
    if (!ok) {
      return
    }
    useMessage(10567, 'success')
    emits('saved')
  }
  isOpen.value = false
}
</script>

<template>
  <KunModal
    v-model="isOpen"
    inner-class-name="max-w-md"
    :is-dismissable="false"
  >
    <form class="space-y-4" @submit.prevent>
      <h2 class="text-xl font-bold">
        {{ mode === 'create' ? '新建收藏夹' : '编辑收藏夹' }}
      </h2>

      <KunInput v-model="form.name" label="名称" required />
      <KunTextarea v-model="form.description" label="描述 (可选)" :rows="3" />
      <KunSelect
        v-model="form.visibility"
        label="隐私设置"
        :options="visibilityOptions"
      />

      <div class="flex justify-end gap-3">
        <KunButton variant="light" color="danger" @click="isOpen = false">
          取消
        </KunButton>
        <KunButton color="primary" :loading="submitting" @click="submit">
          {{ mode === 'create' ? '创建' : '保存' }}
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
