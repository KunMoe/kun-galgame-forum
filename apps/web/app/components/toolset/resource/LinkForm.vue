<script setup lang="ts">
import { createToolsetResourceSchema } from '~/validations/toolset'
import { applyResourceLinkBlur } from '~~/shared/utils/resourceLink'
import { settle, type ClientProblem } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import type { ToolsetUploadResult } from '~/composables/useToolsetResumeUploads'
import {
  emptyToolsetResourceDraft,
  usePersistEditToolsetResourceStore
} from '~/store/modules/edit/toolsetResource'
import type {
  ToolsetResourceCreate,
  ToolsetResourceSummary
} from '#shared/utils/api/schemas'

const props = defineProps<{
  toolsetId: string
  type: 'file' | 'link'
  uploadResult: ToolsetUploadResult | null
}>()

const emits = defineEmits<{
  onClose: []
  onSuccess: [ToolsetResourceSummary]
  onUploadStale: []
}>()

const api = useApiClient()
const createKey = useIdempotencyKey()
const resumeUploads = useToolsetResumeUploads(props.toolsetId)
const draft = usePersistEditToolsetResourceStore()

const formData = reactive({
  toolset_resource_type: props.type,
  artifact_id: props.type === 'file' ? (props.uploadResult?.id ?? '') : '',
  ...(draft.toolsetId === props.toolsetId
    ? { ...draft.fields }
    : emptyToolsetResourceDraft())
})
const isLoading = ref(false)

watch(
  () => ({
    link_url: formData.link_url,
    size_label: formData.size_label,
    extraction_code: formData.extraction_code,
    archive_password: formData.archive_password,
    note: formData.note
  }),
  (fields) => {
    draft.keep(props.toolsetId)
    draft.fields = fields
  }
)

const sizeDisplay = computed(() => {
  if (props.type === 'file') {
    const bytes = props.uploadResult?.file_size ?? 0
    return Number.isFinite(bytes) && bytes > 0 ? formatFileSize(bytes) : ''
  }
  return formData.size_label
})

const onSizeInput = (value: string | number) => {
  if (props.type === 'link') {
    formData.size_label = String(value)
  }
}

watch(
  () => props.type,
  () => {
    formData.toolset_resource_type = props.type
    if (props.type === 'file') {
      formData.artifact_id = props.uploadResult?.id ?? ''
      formData.link_url = ''
      formData.size_label = ''
    } else {
      formData.artifact_id = ''
      formData.link_url = ''
      formData.size_label = ''
    }
  }
)

watch(
  () => props.uploadResult,
  () => {
    if (props.type === 'file') {
      formData.artifact_id = props.uploadResult?.id ?? ''
    }
  }
)

const staleUploadMessage = (problem: ClientProblem) => {
  const error = problem.errors.find((e) => e.pointer === '/artifact_id')
  if (error?.reason === 'UNKNOWN_REFERENCE') {
    return '这个已上传的文件已失效, 请重新上传'
  }
  if (error && problem.code === 'ALREADY_EXISTS') {
    return '这个文件已经发布过了, 刷新页面即可看到'
  }
  return null
}

const commitRecognizedLink = () => {
  if (props.type !== 'link') return
  const recognized = applyResourceLinkBlur(
    formData.link_url,
    formData.extraction_code,
    formData.archive_password
  )
  if (!recognized.applied) return
  formData.link_url = recognized.links[0] ?? formData.link_url
  formData.extraction_code = recognized.code
  formData.archive_password = recognized.password
}

const submitLink = async () => {
  commitRecognizedLink()
  const payload: ToolsetResourceCreate =
    props.type === 'file'
      ? {
          toolset_resource_type: 'file',
          artifact_id: formData.artifact_id,
          ...(formData.extraction_code
            ? { extraction_code: formData.extraction_code }
            : {}),
          ...(formData.archive_password
            ? { archive_password: formData.archive_password }
            : {}),
          ...(formData.note ? { note: formData.note } : {})
        }
      : {
          toolset_resource_type: 'link',
          link_url: formData.link_url,
          size_label: formData.size_label,
          ...(formData.extraction_code
            ? { extraction_code: formData.extraction_code }
            : {}),
          ...(formData.archive_password
            ? { archive_password: formData.archive_password }
            : {}),
          ...(formData.note ? { note: formData.note } : {})
        }
  const result = useKunSchemaValidator(createToolsetResourceSchema, payload)
  if (!result) {
    return
  }

  isLoading.value = true
  const created = await settle(
    api.POST('/toolsets/{toolset_id}/resources', {
      params: {
        path: { toolset_id: props.toolsetId },
        header: {
          'Idempotency-Key': createKey.take(
            `/toolsets/${props.toolsetId}/resources`,
            payload
          )
        }
      },
      body: payload
    })
  )
  isLoading.value = false

  if (!created.ok) {
    const artifactId = payload.artifact_id
    const stale = artifactId ? staleUploadMessage(created.problem) : null
    if (!artifactId || !stale) {
      reportProblem(created.problem)
      return
    }
    resumeUploads.remove(artifactId)
    useMessage(stale, 'warn')
    emits('onUploadStale')
    return
  }
  createKey.clear()
  if (payload.artifact_id) {
    resumeUploads.remove(payload.artifact_id)
  }
  draft.reset()
  useMessage('资源发布成功', 'success')
  emits('onSuccess', created.data)
  emits('onClose')
}
</script>

<template>
  <div class="space-y-3">
    <KunInput
      :placeholder="
        props.type === 'link'
          ? '大小 (如 520KB, 1007MB, 0721GB)'
          : '确认上传完成后, 自动生成文件大小'
      "
      :disabled="props.type === 'file'"
      :model-value="sizeDisplay"
      @update:model-value="onSizeInput"
    />
    <KunInput
      v-if="props.type === 'link'"
      placeholder="提取码 (可选)"
      v-model="formData.extraction_code"
    />
    <KunInput
      placeholder="解压密码 (可选)"
      v-model="formData.archive_password"
    />
    <KunTextarea
      placeholder="备注 (建议写明您提供的资源的使用注意事项等)"
      v-model="formData.note"
    />
    <ResourceLinkInput
      v-if="props.type === 'link'"
      v-model="formData.link_url"
      v-model:code="formData.extraction_code"
      v-model:password="formData.archive_password"
      placeholder="资源链接 (可直接粘贴分享文本)"
    />
    <div class="flex justify-end gap-2">
      <KunButton variant="light" color="danger" @click="emits('onClose')">
        取消
      </KunButton>
      <KunButton :loading="isLoading" :disabled="isLoading" @click="submitLink">
        提交链接
      </KunButton>
    </div>
  </div>
</template>
