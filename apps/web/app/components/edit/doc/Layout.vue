<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { AdminDoc } from '#shared/utils/api/schemas'
import { provideDocEditorContext } from './context'
import type { DocEditorMode, DocEditorForm } from './type'
import { computeReadingMinute } from '~/utils/doc'

const props = withDefaults(
  defineProps<{
    mode: DocEditorMode
    initialDoc?: AdminDoc | null
    redirectOnSuccess?: boolean
  }>(),
  {
    initialDoc: null,
    redirectOnSuccess: true
  }
)

const emit = defineEmits<{
  saved: [doc: AdminDoc]
}>()

const api = useApiClient()
const isRewriteMode = computed(() => props.mode === 'rewrite')

const createDefaultForm = (): DocEditorForm => ({
  doc_id: null,
  title: '',
  slug: '',
  description: '',
  banner_image_hash: '',
  is_pinned: false,
  content_markdown: '',
  doc_category: ''
})

const form = reactive<DocEditorForm>(createDefaultForm())
const isSubmitting = ref(false)
const readingMinute = computed(() =>
  form.content_markdown.trim() ? computeReadingMinute(form.content_markdown) : 0
)

const applyDocToForm = (doc: AdminDoc) => {
  form.doc_id = doc.id
  form.title = doc.title
  form.slug = doc.slug
  form.description = doc.description
  form.banner_image_hash = doc.banner?.hash ?? ''
  form.is_pinned = doc.is_pinned
  form.content_markdown = doc.content_markdown
  form.doc_category = doc.doc_category
}

const resetForm = () => {
  if (isRewriteMode.value && props.initialDoc) {
    applyDocToForm(props.initialDoc)
    return
  }
  Object.assign(form, createDefaultForm())
}

if (isRewriteMode.value && props.initialDoc) {
  applyDocToForm(props.initialDoc)
}

watch(
  () => props.initialDoc,
  (doc) => {
    if (isRewriteMode.value && doc) {
      applyDocToForm(doc)
    }
  }
)

const validateForm = () => {
  if (!form.title.trim()) {
    return '请输入标题'
  }
  if (!form.slug.trim()) {
    return '请输入 slug'
  }
  if (!form.description.trim()) {
    return '请输入简介'
  }
  if (!form.content_markdown.trim()) {
    return '请输入正文内容'
  }
  if (!form.doc_category) {
    return '请选择文档分类'
  }
  return true
}

const handleSubmit = async () => {
  if (isSubmitting.value) {
    return
  }

  const validation = validateForm()
  if (validation !== true) {
    useMessage(validation, 'warn')
    return
  }
  if (!form.doc_category) {
    return
  }

  const body = {
    title: form.title.trim(),
    slug: form.slug.trim(),
    description: form.description.trim(),
    banner_image_hash: form.banner_image_hash,
    is_pinned: form.is_pinned,
    content_markdown: form.content_markdown,
    doc_category: form.doc_category
  }

  isSubmitting.value = true
  try {
    const result = await settle(
      isRewriteMode.value && form.doc_id
        ? api.PATCH('/admin/docs/{doc_id}', {
            params: { path: { doc_id: form.doc_id } },
            body
          })
        : api.POST('/admin/docs', { body })
    )
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    useMessage(isRewriteMode.value ? '更新文档成功' : '创建文档成功', 'success')
    applyDocToForm(result.data)
    if (props.redirectOnSuccess) {
      await navigateTo(`/doc/${result.data.slug}`)
    } else {
      emit('saved', result.data)
    }
  } finally {
    isSubmitting.value = false
  }
}

provideDocEditorContext({
  form,
  mode: props.mode,
  isSubmitting,
  handleSubmit,
  resetForm,
  readingMinute,
  initialBannerUrl: props.initialDoc?.banner?.url ?? ''
})
</script>

<template>
  <div class="contents">
    <ClientOnly>
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div class="order-2 min-w-0 space-y-6 sm:order-1 lg:col-span-1">
          <EditDocMetadataForm />
          <EditDocSubmitActions />
        </div>

        <div class="order-1 min-w-0 space-y-4 sm:order-2 lg:col-span-2">
          <EditDocTitle />
          <EditDocContentEditor />
        </div>
      </div>
    </ClientOnly>
  </div>
</template>
