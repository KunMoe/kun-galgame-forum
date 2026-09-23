<script setup lang="ts">
import { updateToolsetResourceSchema } from '~/validations/toolset'
import { KUN_GALGAME_TOOLSET_STORAGE_MAP } from '~/constants/toolset'
import { applyResourceLinkBlur } from '~~/shared/utils/resourceLink'
import { settle } from '#shared/utils/api/problem'
import type {
  ToolsetDownload,
  ToolsetResourceSource,
  ToolsetResourcePatch,
  ToolsetResourceSummary
} from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  toolsetId: string
  resource: ToolsetResourceSummary
}>()

const emits = defineEmits<{
  deleted: [string]
  updated: [ToolsetResourceSummary]
}>()

const api = useApiClient()

const base = ref<ToolsetResourceSummary>(props.resource)
watch(
  () => props.resource,
  (v) => {
    base.value = v
  }
)

const download = ref<ToolsetDownload | null>(null)
const editSource = ref<ToolsetResourceSource | null>(null)
const showing = ref(false)
const fetching = ref(false)

const isEditing = ref(false)
const isDeleting = ref(false)
const isSaving = ref(false)

const canEditResource = computed(() => base.value.viewer?.can_edit ?? false)
const canDeleteResource = computed(() => base.value.viewer?.can_delete ?? false)
const isFile = computed(() => base.value.toolset_resource_type === 'file')
const poster = computed(() => toKunUser(base.value.poster))

const displaySize = computed(() => {
  if (isFile.value) {
    return formatFileSize(base.value.archive?.file_size ?? 0)
  }
  return base.value.link?.size_label ?? ''
})

const formData = reactive({
  size_label: base.value.link?.size_label ?? '',
  extraction_code: '',
  archive_password: '',
  note: base.value.note ?? '',
  link_url: ''
})

const fetchDownload = async () => {
  if (download.value) {
    return true
  }

  fetching.value = true
  const result = await settle(
    api.POST('/toolsets/{toolset_id}/resources/{resource_id}/downloads', {
      params: {
        path: {
          toolset_id: props.toolsetId,
          resource_id: base.value.id
        }
      }
    })
  )
  fetching.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  download.value = result.data
  const next = {
    ...base.value,
    download_count: base.value.download_count + 1
  }
  base.value = next
  emits('updated', next)
  return true
}

const toggleShow = async () => {
  if (!showing.value) {
    const ok = await fetchDownload()
    if (!ok) {
      return
    }
  }
  showing.value = !showing.value
}

const startEdit = async () => {
  fetching.value = true
  const result = await settle(
    api.GET('/toolsets/{toolset_id}/resources/{resource_id}/source', {
      params: {
        path: {
          toolset_id: props.toolsetId,
          resource_id: base.value.id
        }
      }
    })
  )
  fetching.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  editSource.value = result.data
  formData.size_label = result.data.size_label ?? ''
  formData.note = result.data.note ?? ''
  formData.extraction_code = result.data.extraction_code
  formData.archive_password = result.data.archive_password
  formData.link_url = result.data.link_url ?? ''
  isEditing.value = true
}

const handleDelete = async () => {
  if (!canDeleteResource.value) {
    useMessage('您没有权限删除该工具资源', 'warn')
    return
  }
  const okConfirm = await useComponentMessageStore().alert(
    '确定删除该工具资源吗？',
    `删除资源将会消耗您 3 萌萌点, 资源将会永久删除, 不可恢复`
  )
  if (!okConfirm) {
    return
  }

  isDeleting.value = true
  const result = await settle(
    api.DELETE('/toolsets/{toolset_id}/resources/{resource_id}', {
      params: {
        path: {
          toolset_id: props.toolsetId,
          resource_id: base.value.id
        }
      }
    })
  )
  isDeleting.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('删除成功', 'success')
  emits('deleted', base.value.id)
}

const handleSave = async () => {
  if (!isFile.value) {
    const recognized = applyResourceLinkBlur(
      formData.link_url,
      formData.extraction_code,
      formData.archive_password
    )
    if (recognized.applied) {
      formData.link_url = recognized.links[0] ?? formData.link_url
      formData.extraction_code = recognized.code
      formData.archive_password = recognized.password
    }
  }

  const body: ToolsetResourcePatch = {}
  if (formData.archive_password !== (editSource.value?.archive_password ?? '')) {
    body.archive_password = formData.archive_password
  }
  if (formData.note !== (base.value.note ?? '')) {
    body.note = formData.note
  }
  if (!isFile.value) {
    if (formData.extraction_code !== (editSource.value?.extraction_code ?? '')) {
      body.extraction_code = formData.extraction_code
    }
    if (formData.size_label !== (base.value.link?.size_label ?? '')) {
      body.size_label = formData.size_label
    }
    if (formData.link_url !== (editSource.value?.link_url ?? '')) {
      body.link_url = formData.link_url
    }
  }

  if (Object.keys(body).length === 0) {
    isEditing.value = false
    return
  }

  const valid = useKunSchemaValidator(updateToolsetResourceSchema, body)
  if (!valid) {
    return
  }

  isSaving.value = true
  const result = await settle(
    api.PATCH('/toolsets/{toolset_id}/resources/{resource_id}', {
      params: {
        path: {
          toolset_id: props.toolsetId,
          resource_id: base.value.id
        }
      },
      body
    })
  )
  isSaving.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  base.value = result.data
  emits('updated', result.data)
  if (download.value) {
    if (body.archive_password !== undefined) {
      download.value.archive_password = body.archive_password
    }
    if (body.extraction_code !== undefined) {
      download.value.extraction_code = body.extraction_code
    }
    if (body.link_url !== undefined) {
      download.value.download_url = body.link_url
    }
  }
  useMessage('更新成功', 'success')
  isEditing.value = false
}
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-2">
        <KunChip size="sm" color="secondary">
          {{ KUN_GALGAME_TOOLSET_STORAGE_MAP[base.toolset_resource_type] }}
        </KunChip>
        <KunChip size="sm" color="warning">
          <KunIcon name="lucide:database" />
          {{ displaySize }}
        </KunChip>
        <KunChip size="sm" color="primary">
          <KunIcon name="lucide:download" />
          <span>{{ `${base.download_count} 人下载` }}</span>
        </KunChip>
      </div>

      <div class="ml-auto flex items-center gap-1">
        <KunButton
          size="sm"
          variant="flat"
          :loading="fetching"
          @click="toggleShow"
        >
          {{ showing ? '隐藏链接' : '获取链接' }}
        </KunButton>

        <KunButton
          v-if="canEditResource"
          :is-icon-only="true"
          variant="light"
          @click="startEdit"
        >
          <KunIcon name="lucide:pencil" />
        </KunButton>
        <KunButton
          v-if="canDeleteResource"
          :is-icon-only="true"
          color="danger"
          variant="light"
          :loading="isDeleting"
          @click="handleDelete"
        >
          <KunIcon name="lucide:trash-2" />
        </KunButton>
      </div>
    </div>

    <div v-if="showing && download" class="space-y-2">
      <div class="flex items-center gap-2">
        <KunAvatar :user="poster" />
        <span>{{ poster.name }}</span>
        <span class="text-default-500 text-sm">
          <KunTime :time="base.created_at" />
        </span>
      </div>

      <div class="flex items-center gap-2">
        <KunCopy
          v-if="download.extraction_code"
          variant="flat"
          :name="`提取码 ${download.extraction_code}`"
          :text="download.extraction_code"
        />
        <KunCopy
          v-if="download.archive_password"
          variant="flat"
          :name="`解压码 ${download.archive_password}`"
          :text="download.archive_password"
        />
      </div>

      <KunInfo v-if="base.note" color="info" title="下载备注信息">
        <pre class="font-sans break-all whitespace-pre-line">
          {{ base.note }}
        </pre>
      </KunInfo>

      <p v-if="!download.download_url" class="text-default-500 text-sm">
        这个资源没有登记下载链接，链接可能写在提取码或备注里
      </p>
      <div v-else class="space-y-2 space-x-2">
        <p class="text-default-500 text-sm">点击下面的链接以下载</p>
        <KunLink
          :to="download.download_url"
          target="_blank"
          rel="noopener noreferrer"
          :is-show-anchor-icon="true"
        >
          {{ download.download_url }}
        </KunLink>
        <p v-if="download.expires_at" class="text-default-500 text-sm">
          链接有效至
          <KunTime :time="download.expires_at" />
        </p>
      </div>
    </div>

    <KunCard
      :is-hoverable="false"
      :is-transparent="true"
      v-if="isEditing"
      content-class="space-y-3 rounded-lg"
    >
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <KunInput
          v-if="!isFile"
          v-model="formData.size_label"
          placeholder="资源大小 (如 1007MB, 0721GB)"
        />
        <KunInput
          v-if="!isFile"
          v-model="formData.extraction_code"
          placeholder="资源提取码 (可选)"
        />
        <KunInput
          v-model="formData.archive_password"
          placeholder="资源解压码 (可选)"
        />
      </div>
      <KunTextarea
        v-model="formData.note"
        placeholder="资源备注 (可选, 建议您写明资源的使用方法和注意事项)"
      />
      <ResourceLinkInput
        v-if="!isFile"
        v-model="formData.link_url"
        v-model:code="formData.extraction_code"
        v-model:password="formData.archive_password"
        placeholder="资源链接 (可直接粘贴分享文本)"
      />

      <div class="flex justify-end gap-2">
        <KunButton variant="light" color="danger" @click="isEditing = false">
          取消
        </KunButton>
        <KunButton :loading="isSaving" :disabled="isSaving" @click="handleSave">
          保存
        </KunButton>
      </div>
    </KunCard>

    <KunDivider margin="0 0 14px 0" />
  </div>
</template>
