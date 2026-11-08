<script setup lang="ts">
import { createCollectionSchema } from '~/validations/collection'
import { settle, type ClientProblem } from '#shared/utils/api/problem'
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
    tags: string[]
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

const form = reactive<{
  title: string
  description: string
  visibility: CollectionVisibility
  tags: string[]
}>({
  title: '',
  description: '',
  visibility: 'public',
  tags: []
})
const submitting = ref(false)
const fieldErrors = ref<Record<string, string>>({})

const showProblem = (problem: ClientProblem) => {
  fieldErrors.value = collectionFieldErrors(problem, '/title')
  if (!Object.keys(fieldErrors.value).length) {
    reportProblem(problem)
  }
}

watch(
  () => isOpen.value,
  (open) => {
    if (!open) {
      return
    }
    submitting.value = false
    fieldErrors.value = {}
    form.title = props.initial?.title ?? ''
    form.description = props.initial?.description ?? ''
    form.visibility = props.initial?.visibility ?? 'public'
    form.tags = [...(props.initial?.tags ?? [])]
  }
)

const submit = async () => {
  fieldErrors.value = {}
  const parsed = createCollectionSchema.safeParse({
    title: form.title.trim(),
    description: form.description,
    visibility: form.visibility,
    tags: form.tags.map((t) => t.replace(/^#+/, ''))
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
    if (parsed.data.tags.length) {
      payload.tag_names = parsed.data.tags
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
      showProblem(result.problem)
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
    if (
      parsed.data.tags.join('\n') !== (props.initial?.tags ?? []).join('\n')
    ) {
      body.tag_names = parsed.data.tags
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
      showProblem(result.problem)
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
    :title="mode === 'create' ? '新建收藏夹' : '编辑收藏夹'"
  >
    <form class="space-y-4" @submit.prevent>
      <KunInput
        v-model="form.title"
        label="名称"
        required
        :error="fieldErrors['/title'] ?? ''"
      />
      <KunTextarea
        v-model="form.description"
        label="描述 (可选)"
        :rows="3"
        :maxlength="500"
        show-char-count
        :error="fieldErrors['/description'] ?? ''"
      />
      <KunSelect
        v-model="form.visibility"
        label="谁可以看到"
        :options="[...COLLECTION_VISIBILITY_OPTIONS]"
      />
      <KunTagInput
        v-model="form.tags"
        label="标签 (可选)"
        placeholder="输入后按回车，例如 纯爱、拔作、入坑推荐"
        :max-tags="COLLECTION_TAG_MAX"
        :max-tag-length="COLLECTION_TAG_LENGTH_MAX"
        :error="fieldErrors['/tag_names'] ?? ''"
        helper-text="最多 5 个，每个不超过 16 字；公开收藏夹可以在广场按标签筛选"
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
