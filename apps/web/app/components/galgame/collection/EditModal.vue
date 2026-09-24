<script setup lang="ts">
import { createCollectionSchema } from '~/validations/collection'
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import type {
  CollectionCreate,
  CollectionPatch,
  CollectionVisibility
} from '#shared/utils/api/schemas'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  collectionId?: string
  isDefault?: boolean
  initial?: {
    title: string
    description: string
    visibility: CollectionVisibility
  }
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [newId?: string]
}>()

const api = useApiClient()
const createKey = useIdempotencyKey()

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const visibilityOptions = [
  { value: 'public', label: '公开 · 所有人可见' },
  { value: 'private', label: '私密 · 仅自己可见' }
] as const

const form = reactive<{
  title: string
  description: string
  visibility: CollectionVisibility
}>({
  title: '',
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
    form.title = props.initial?.title ?? ''
    form.description = props.initial?.description ?? ''
    form.visibility = props.initial?.visibility ?? 'public'
  }
)

const submit = async () => {
  const parsed = createCollectionSchema.safeParse({
    title: form.title.trim(),
    description: form.description,
    visibility: form.visibility
  })
  if (!parsed.success) {
    const issue = JSON.parse(parsed.error.message)[0]
    useMessage(formatKunZodIssue(issue), 'warn')
    return
  }

  submitting.value = true
  if (props.mode === 'create') {
    const payload: CollectionCreate = {
      title: parsed.data.title,
      description: parsed.data.description,
      visibility: parsed.data.visibility
    }
    if (props.isDefault) {
      payload.is_default = true
    }
    const result = await settle(
      api.POST('/collections', {
        params: {
          header: {
            'Idempotency-Key': createKey.take('/collections', payload)
          }
        },
        body: payload
      })
    )
    submitting.value = false
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    createKey.clear()
    useMessage(10566, 'success')
    emits('saved', result.data.id)
  } else {
    if (!props.collectionId) {
      submitting.value = false
      return
    }
    const body: CollectionPatch = {}
    if (parsed.data.title !== (props.initial?.title ?? '')) {
      body.title = parsed.data.title
    }
    if (parsed.data.description !== (props.initial?.description ?? '')) {
      body.description = parsed.data.description
    }
    if (parsed.data.visibility !== (props.initial?.visibility ?? 'public')) {
      body.visibility = parsed.data.visibility
    }
    if (Object.keys(body).length === 0) {
      submitting.value = false
      isOpen.value = false
      return
    }
    const result = await settle(
      api.PATCH('/collections/{collection_id}', {
        params: { path: { collection_id: props.collectionId } },
        body
      })
    )
    submitting.value = false
    if (!result.ok) {
      reportProblem(result.problem)
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

      <KunInput v-model="form.title" label="名称" required />
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
