<script setup lang="ts">
import {
  MAX_SMALL_FILE_SIZE,
  MAX_LARGE_FILE_SIZE,
  USER_DAILY_UPLOAD_LIMIT,
  MOEMOEPOINT_SINGLE_MB_DIVISOR
} from '~/config/upload'
import {
  initToolsetUploadSchema,
  completeToolsetUploadSchema
} from '~/validations/toolset'
import {
  KUN_GALGAME_TOOLSET_UPLOAD_STATUS_MAP,
  type KUN_GALGAME_TOOLSET_UPLOAD_STATUS_CONST
} from '~/constants/toolset'
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import type { ToolsetPendingUpload } from '~/composables/useToolsetResumeUploads'
import type {
  ToolsetUpload,
  ToolsetUploadCreate,
  ToolsetUploadPatch
} from '#shared/utils/api/schemas'

const props = defineProps<{
  toolsetId: string
}>()

const emits = defineEmits<{
  onUploadSuccess: [{ id: string; file_size: number }]
  onClose: []
}>()

const MB = 1024 * 1024
const LARGE_CHUNK_SIZE = 5 * MB
const UPLOAD_TRANSFER_FAILED = 'UPLOAD_TRANSFER_FAILED'
const DEFAULT_BINARY_CONTENT_TYPE = 'application/octet-stream'
type ToolsetUploadStatus =
  (typeof KUN_GALGAME_TOOLSET_UPLOAD_STATUS_CONST)[number]
type ToolsetUploadPart = {
  part_number: number
  etag: string
}

const resolveContentType = (file: File): string =>
  file.type && file.type.length > 0 ? file.type : DEFAULT_BINARY_CONTENT_TYPE

const { moemoepoint, dailyToolsetUploadBytes } = storeToRefs(
  usePersistUserStore()
)
const canUploadBypass = useCan('toolset.upload_bypass')
const api = useApiClient()
const uploadKey = useIdempotencyKey()
const fileInput = ref<HTMLInputElement>()
const selectedFile = ref<File | null>(null)

const progress = ref(0)
const isDragging = ref(false)
const uploadStatus = ref<ToolsetUploadStatus>('idle')
const resumeId = ref<string | null>(null)

const isLarge = computed(() => {
  const f = selectedFile.value
  return !!f && f.size > MAX_SMALL_FILE_SIZE
})
const dailyUploadLimit = computed(() => {
  if (canUploadBypass.value) {
    return MAX_LARGE_FILE_SIZE
  }

  return Math.max(
    0,
    USER_DAILY_UPLOAD_LIMIT +
      moemoepoint.value * MB -
      dailyToolsetUploadBytes.value
  )
})
const maxSingleFileLimit = computed(() => {
  if (canUploadBypass.value) {
    return MAX_LARGE_FILE_SIZE
  }

  const moemoepointMaxSingleFile =
    Math.floor(moemoepoint.value / MOEMOEPOINT_SINGLE_MB_DIVISOR) * MB

  return Math.min(
    Math.max(USER_DAILY_UPLOAD_LIMIT, moemoepointMaxSingleFile),
    MAX_LARGE_FILE_SIZE
  )
})

const statusMessage = computed(() => {
  if (uploadStatus.value === 'largeUploading') {
    return `正在上传大文件【进度 ${progress.value}%】`
  } else {
    return KUN_GALGAME_TOOLSET_UPLOAD_STATUS_MAP[uploadStatus.value]
  }
})

const resetUploadState = () => {
  progress.value = 0
  uploadStatus.value = 'idle'
}

const setSelectedUploadFile = (file: File) => {
  selectedFile.value = file
  resetUploadState()
  const match = resumeStore
    .list()
    .find((p) => p.size === file.size && p.last_modified === file.lastModified)
  resumeId.value = match ? match.id : null
}

const throwIfUploadFailed = (response: Response) => {
  if (!response.ok) {
    throw new Error(UPLOAD_TRANSFER_FAILED)
  }
}

const isUploadTransferFailedError = (error: unknown) => {
  return error instanceof Error && error.message === UPLOAD_TRANSFER_FAILED
}

const notifyUploadTransferError = (error: unknown) => {
  if (isUploadTransferFailedError(error)) {
    useMessage('文件传输失败，请重试', 'error')
  }
}

const abortUpload = async (uploadId: string) => {
  const result = await settle(
    api.DELETE('/toolsets/{toolset_id}/uploads/{upload_id}', {
      params: {
        path: {
          toolset_id: props.toolsetId,
          upload_id: uploadId
        }
      }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  uploadKey.clear()
}

const checkFileValid = (file: File | null) => {
  if (!file) {
    return false
  }
  if (!isValidArchive(file.name || '')) {
    useMessage('我们仅支持 .7z, .zip, .rar 压缩格式上传', 'warn')
    return false
  }
  if (file.size > MAX_LARGE_FILE_SIZE) {
    useMessage(
      `文件大小超过最大文件限制 ${MAX_LARGE_FILE_SIZE / MB} MB`,
      'warn'
    )
    return false
  }
  if (file.size > dailyUploadLimit.value) {
    useMessage(
      `超出当日可用上传额度, 剩余 ${(dailyUploadLimit.value / MB).toFixed(2)} MB`,
      'warn'
    )
    return false
  }
  if (file.size > maxSingleFileLimit.value) {
    useMessage(
      `单文件大小超过限制, 最大 ${(maxSingleFileLimit.value / MB).toFixed(2)} MB`,
      'warn'
    )
    return false
  }
  return true
}

const pick = () => fileInput.value?.click()
const onChange = (e: Event) => {
  const t = e.target as HTMLInputElement
  const targetFile = t.files && t.files[0] ? t.files[0] : null
  const res = checkFileValid(targetFile)
  if (!res) {
    return
  }
  if (!targetFile) {
    return
  }
  setSelectedUploadFile(targetFile)
}
const onDrop = (e: DragEvent) => {
  e.preventDefault()
  e.stopPropagation()
  isDragging.value = false
  const dt = e.dataTransfer
  if (dt?.files && dt.files.length > 0) {
    const res = checkFileValid(dt.files[0]!)
    if (!res) {
      return
    }
    setSelectedUploadFile(dt.files[0]!)
  }
}
const onDragOver = (e: DragEvent) => {
  e.preventDefault()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'copy'
  }
}
const onDragEnter = () => {
  isDragging.value = true
}
const onDragLeave = () => {
  isDragging.value = false
}
const clearSelected = () => {
  selectedFile.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
  resetUploadState()
  resumeId.value = null
}

const resumeStore = useToolsetResumeUploads(props.toolsetId)
const pending = ref<ToolsetPendingUpload[]>([])

const refreshPending = () => {
  pending.value = resumeStore.list()
}

const rememberPending = (f: File, uploadId: string, prog: number) => {
  resumeStore.upsert({
    id: uploadId,
    name: f.name,
    size: f.size,
    last_modified: f.lastModified,
    progress: prog,
    updated_at: Date.now()
  })
}

const forgetPending = (uploadId: string) => {
  resumeStore.remove(uploadId)
  if (resumeId.value === uploadId) {
    resumeId.value = null
  }
  refreshPending()
}

const putParts = async (
  uploadId: string,
  f: File,
  partList: { part_number: number; url: string }[],
  partSize: number,
  contentType: string,
  totalParts: number,
  doneCount: number
): Promise<ToolsetUploadPart[]> => {
  const out: ToolsetUploadPart[] = []
  for (let i = 0; i < partList.length; i++) {
    const cur = partList[i]
    if (!cur) {
      throw new Error('Missing upload part')
    }
    const start = (cur.part_number - 1) * partSize
    const end = Math.min(start + partSize, f.size)
    const resp = await fetch(cur.url, {
      headers: { 'Content-Type': contentType },
      method: 'PUT',
      body: f.slice(start, end)
    })
    throwIfUploadFailed(resp)
    const etag = resp.headers.get('ETag') || resp.headers.get('etag')
    if (!etag) {
      throw new Error('Missing ETag')
    }
    out.push({ part_number: cur.part_number, etag })
    progress.value = Math.round(((doneCount + i + 1) / totalParts) * 100)
    resumeStore.setProgress(uploadId, progress.value)
  }
  return out
}

const completeUpload = async (
  uploadId: string,
  parts: ToolsetUploadPart[] | undefined
): Promise<boolean> => {
  const body: ToolsetUploadPatch =
    parts && parts.length
      ? { state: 'completed', parts }
      : { state: 'completed' }
  if (!useKunSchemaValidator(completeToolsetUploadSchema, body)) {
    return false
  }
  const done = await settle(
    api.PATCH('/toolsets/{toolset_id}/uploads/{upload_id}', {
      params: {
        path: {
          toolset_id: props.toolsetId,
          upload_id: uploadId
        }
      },
      body
    })
  )
  if (!done.ok) {
    reportProblem(done.problem)
    return false
  }
  useMessage('上传成功', 'success')
  emits('onUploadSuccess', {
    id: done.data.id,
    file_size: done.data.file_size
  })
  progress.value = 100
  uploadStatus.value = 'complete'
  forgetPending(uploadId)
  uploadKey.clear()
  return true
}

const registerResumable = (uploadId: string, f: File) => {
  rememberPending(f, uploadId, progress.value)
  resumeId.value = uploadId
  uploadStatus.value = 'idle'
  refreshPending()
}

const applySession = async (session: ToolsetUpload, f: File) => {
  const contentType = resolveContentType(f)
  if (session.is_multipart) {
    rememberPending(f, session.id, 0)
    refreshPending()
    const partList = session.part_urls ?? []
    const partSize = session.part_size || LARGE_CHUNK_SIZE
    try {
      uploadStatus.value = 'largeUploading'
      const parts = await putParts(
        session.id,
        f,
        partList,
        partSize,
        contentType,
        partList.length,
        0
      )
      uploadStatus.value = 'largeComplete'
      if (!(await completeUpload(session.id, parts))) {
        registerResumable(session.id, f)
      }
    } catch (error) {
      registerResumable(session.id, f)
      notifyUploadTransferError(error)
    }
    return
  }

  try {
    uploadStatus.value = 'smallUploading'
    if (!session.upload_url) {
      throw new Error('Missing upload URL')
    }
    const resp = await fetch(session.upload_url, {
      headers: { 'Content-Type': contentType },
      method: 'PUT',
      body: f
    })
    throwIfUploadFailed(resp)
    uploadStatus.value = 'smallComplete'
    progress.value = 100
    if (!(await completeUpload(session.id, undefined))) {
      resetUploadState()
    }
  } catch (error) {
    await abortUpload(session.id)
    notifyUploadTransferError(error)
    resetUploadState()
  }
}

const uploadToArtifact = async (f: File) => {
  const contentType = resolveContentType(f)
  const initData: ToolsetUploadCreate = {
    filename: f.name,
    file_size: f.size,
    content_type: contentType
  }
  if (!useKunSchemaValidator(initToolsetUploadSchema, initData)) {
    return
  }

  progress.value = 0
  uploadStatus.value = isLarge.value ? 'largeInit' : 'smallInit'
  const init = await settle(
    api.POST('/toolsets/{toolset_id}/uploads', {
      params: {
        path: { toolset_id: props.toolsetId },
        header: {
          'Idempotency-Key': uploadKey.take(
            `/toolsets/${props.toolsetId}/uploads`,
            initData
          )
        }
      },
      body: initData
    })
  )
  if (!init.ok) {
    reportProblem(init.problem)
    uploadStatus.value = 'idle'
    return
  }

  await applySession(init.data, f)
}

const resumeUploadToArtifact = async (f: File, uploadId: string) => {
  const contentType = resolveContentType(f)

  progress.value = 0
  uploadStatus.value = 'largeInit'
  const resume = await settle(
    api.GET('/toolsets/{toolset_id}/uploads/{upload_id}', {
      params: {
        path: {
          toolset_id: props.toolsetId,
          upload_id: uploadId
        }
      }
    })
  )
  if (!resume.ok) {
    if (resume.problem.status !== 404) {
      reportProblem(resume.problem)
    }
    forgetPending(uploadId)
    await uploadToArtifact(f)
    return
  }

  if (!resume.data.is_multipart) {
    try {
      uploadStatus.value = 'smallUploading'
      if (!resume.data.upload_url) {
        throw new Error('Missing upload URL')
      }
      const resp = await fetch(resume.data.upload_url, {
        headers: { 'Content-Type': contentType },
        method: 'PUT',
        body: f
      })
      throwIfUploadFailed(resp)
      progress.value = 100
      if (!(await completeUpload(uploadId, undefined))) {
        registerResumable(uploadId, f)
      }
    } catch (error) {
      registerResumable(uploadId, f)
      notifyUploadTransferError(error)
    }
    return
  }

  const partSize = resume.data.part_size || LARGE_CHUNK_SIZE
  const uploaded = resume.data.uploaded_parts ?? []
  const missing = resume.data.part_urls ?? []
  const totalParts = uploaded.length + missing.length
  try {
    uploadStatus.value = 'largeUploading'
    progress.value =
      totalParts > 0 ? Math.round((uploaded.length / totalParts) * 100) : 0
    const fresh = await putParts(
      uploadId,
      f,
      missing,
      partSize,
      contentType,
      totalParts,
      uploaded.length
    )
    const parts: ToolsetUploadPart[] = [
      ...uploaded.map((p) => ({ part_number: p.part_number, etag: p.etag })),
      ...fresh
    ].sort((a, b) => a.part_number - b.part_number)
    uploadStatus.value = 'largeComplete'
    if (!(await completeUpload(uploadId, parts))) {
      registerResumable(uploadId, f)
    }
  } catch (error) {
    registerResumable(uploadId, f)
    notifyUploadTransferError(error)
  }
}

const submit = async () => {
  const f = selectedFile.value
  if (!f) {
    useMessage('请选择文件', 'warn')
    return
  }
  if (resumeId.value) {
    await resumeUploadToArtifact(f, resumeId.value)
  } else {
    await uploadToArtifact(f)
  }
}

const handleContinuePending = (record: ToolsetPendingUpload, file: File) => {
  setSelectedUploadFile(file)
  resumeUploadToArtifact(file, record.id)
}

const handleDeletePending = async (uploadId: string) => {
  await abortUpload(uploadId)
  forgetPending(uploadId)
}

onMounted(refreshPending)
</script>

<template>
  <div class="space-y-4">
    <input ref="fileInput" type="file" hidden @change="onChange" />

    <div v-if="pending.length && uploadStatus === 'idle'" class="space-y-2">
      <div
        class="text-warning border-warning/30 bg-warning/10 flex items-start gap-2 rounded-lg border p-3 text-sm"
      >
        <KunIcon name="lucide:history" class="mt-0.5 shrink-0" />
        <span>
          您有 {{ pending.length }}
          个未完成的上传，您可以点击「继续上传」后选择未传完的文件继续上传
        </span>
      </div>

      <ToolsetResourceResumeList
        :pending="pending"
        @continue="handleContinuePending"
        @delete="handleDeletePending"
      />
    </div>

    <KunCard :is-hoverable="false" :is-transparent="true">
      <div
        class="cursor-pointer rounded-lg border-2 border-dashed p-6 text-center transition-colors"
        :class="
          cn(
            isDragging
              ? 'border-primary-500 bg-primary-50/50'
              : 'border-default-300 hover:border-default-500'
          )
        "
        @click="pick"
        @drop="onDrop"
        @dragover="onDragOver"
        @dragenter="onDragEnter"
        @dragleave="onDragLeave"
      >
        <div v-if="!selectedFile" class="flex flex-col items-center gap-2">
          <KunIcon
            name="lucide:upload-cloud"
            class="text-default-500 text-3xl"
          />
          <div class="text-default-600">点击或拖拽文件到此处</div>
        </div>

        <div v-else class="flex flex-col justify-center gap-2">
          <div class="flex items-center gap-3">
            <KunIcon
              name="lucide:file-check"
              class="text-success-600 text-xl"
            />
            <div class="text-default-700 font-medium">
              {{ selectedFile?.name }}
            </div>
          </div>

          <div class="flex items-center gap-3">
            <div class="text-default-500 text-xs">
              {{ formatFileSize(selectedFile!.size) }}
            </div>
            <span
              class="border-default-200 bg-default-100 text-default-600 rounded-full border px-2 py-0.5 text-xs"
            >
              {{
                isLarge
                  ? `文件大于 ${MAX_SMALL_FILE_SIZE / MB}MB, 分片上传`
                  : `文件小于 ${MAX_SMALL_FILE_SIZE / MB}MB, 直接上传`
              }}
            </span>
          </div>

          <KunProgress :value="progress" />

          <div
            v-if="resumeId && uploadStatus === 'idle'"
            class="text-warning flex items-center justify-center gap-1.5 text-center text-xs"
          >
            <KunIcon name="lucide:history" />
            检测到未完成的上传，将从断点继续
          </div>

          <div
            class="text-default-500 flex items-center justify-center gap-2 text-sm"
          >
            <span>{{ statusMessage }}</span>
            <KunIcon
              class="text-sm"
              v-if="uploadStatus !== 'idle' && uploadStatus !== 'complete'"
              name="svg-spinners:90-ring-with-bg"
            />
            <KunIcon
              class="text-success-600 text-sm"
              v-if="uploadStatus === 'complete'"
              name="lucide:circle-check-big"
            />
          </div>
        </div>
      </div>
    </KunCard>

    <div class="flex items-center justify-end gap-2">
      <KunButton
        v-if="selectedFile"
        variant="light"
        color="danger"
        @click.stop="clearSelected"
      >
        移除文件
      </KunButton>
      <KunButton
        :loading="uploadStatus !== 'idle' && uploadStatus !== 'complete'"
        :disabled="!selectedFile || uploadStatus === 'complete'"
        @click="submit"
      >
        {{ resumeId ? '继续上传' : '确认上传' }}
      </KunButton>
    </div>
  </div>
</template>
