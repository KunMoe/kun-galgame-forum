<script setup lang="ts">
import { checkGalgameResourcePublish } from '../utils/checkGalgameResourcePublish'
import {
  FILE_SIZE_UNITS,
  applyResourceSizeInput,
  joinResourceSize,
  splitResourceSize
} from '~~/shared/utils/resourceSize'
import {
  applyResourceLinkBlur,
  splitResourceLinkText
} from '~~/shared/utils/resourceLink'
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import type {
  GalgameResourceCreate,
  GalgameResourcePatch,
  GalgameResourceSource
} from '#shared/utils/api/schemas'
import {
  LANGUAGE_OPTIONS,
  PLATFORM_OPTIONS,
  RESOURCE_TYPE_OPTIONS,
  RUNTIME_OPTIONS,
  VERSION_LABEL_OPTIONS,
  hasRuntimeAxis,
  resourceTypeLabel
} from '~~/shared/utils/galgameResourceVocab'

const props = defineProps<{
  workId: string
  resourceId?: string | null
  refresh: () => void
}>()

const open = defineModel<boolean>({ required: true })

const api = useApiClient()
const createKey = useIdempotencyKey()

const isEditing = computed(() => !!props.resourceId)

const modalTitle = computed(() =>
  isEditing.value ? '重新编辑资源信息' : '发布 Galgame 资源'
)
const modalSubtitle = computed(() =>
  isEditing.value
    ? '修改链接 / 提取码 / 备注等信息, 保存后立即生效。'
    : '为这部 Galgame 提交一份新的资源链接, 提交后立即对所有用户可见。'
)
const submitLabel = computed(() => (isEditing.value ? '保存修改' : '发布资源'))

type ResourceType = GalgameResourceCreate['resource_type']
type ResourceLanguage = GalgameResourceCreate['resource_languages'][number]
type ResourcePlatform = NonNullable<
  GalgameResourceCreate['resource_platforms']
>[number]
type ResourceRuntime = NonNullable<
  GalgameResourceCreate['resource_runtimes']
>[number]
type VersionLabel = NonNullable<
  Exclude<GalgameResourceCreate['version_label'], null>
>

interface FormShape {
  resource_type: ResourceType
  title: string
  version_label: string
  download_urls: string[]
  resource_languages: ResourceLanguage[]
  resource_platforms: ResourcePlatform[]
  resource_runtimes: ResourceRuntime[]
  size: string
  extraction_code: string
  archive_password: string
  content_markdown: string
}

const defaultForm = (): FormShape => ({
  resource_type: 'game',
  title: '',
  version_label: '',
  download_urls: [],
  resource_languages: ['zh-cn'],
  resource_platforms: ['win'],
  resource_runtimes: ['native-win'],
  size: '',
  extraction_code: '',
  archive_password: '',
  content_markdown: ''
})

const snapshotFromSource = (source: GalgameResourceSource): FormShape => ({
  resource_type: source.resource_type,
  title: source.title,
  version_label: source.version_label ?? '',
  download_urls: [...source.download_urls],
  resource_languages: [...source.resource_languages],
  resource_platforms: [...source.resource_platforms],
  resource_runtimes: [...source.resource_runtimes],
  size: source.size,
  extraction_code: source.extraction_code,
  archive_password: source.archive_password,
  content_markdown: source.content_markdown
})

const form = ref<FormShape>(defaultForm())
const source = ref<GalgameResourceSource | null>(null)
const linkText = ref('')
const size = reactive(splitResourceSize(form.value.size))
const sizeInput = ref<{ inputRef?: HTMLInputElement | null } | null>(null)
const isLoadingSource = ref(false)
const editorKey = ref(0)

const paintSizeAmount = () => {
  const el = sizeInput.value?.inputRef
  if (el && el.value !== size.amount) el.value = size.amount
}

const applyForm = (next: FormShape) => {
  form.value = next
  linkText.value = next.download_urls.join(', ')
  Object.assign(size, splitResourceSize(next.size))
  editorKey.value += 1
  nextTick(paintSizeAmount)
}

watch(open, async (isOpen) => {
  if (!isOpen) return
  source.value = null
  if (!props.resourceId) {
    applyForm(defaultForm())
    return
  }
  isLoadingSource.value = true
  const result = await settle(
    api.GET('/galgame-resources/{resource_id}/source', {
      params: { path: { resource_id: props.resourceId } }
    })
  )
  isLoadingSource.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    open.value = false
    return
  }
  source.value = result.data
  applyForm(snapshotFromSource(result.data))
})
watch(size, () => {
  form.value.size = joinResourceSize(size)
})
watch(
  () => size.unit,
  () => {
    nextTick(paintSizeAmount)
  }
)

const onSizeAmount = (raw: string | number) => {
  Object.assign(size, applyResourceSizeInput(String(raw), size.unit))
  nextTick(paintSizeAmount)
}

const isSubmitting = ref(false)

const sameList = (a: readonly string[], b: readonly string[]) =>
  a.length === b.length && a.every((item, i) => item === b[i])

const versionToken = (value: string): VersionLabel | null =>
  value ? (value as VersionLabel) : null

const handleSubmit = async () => {
  if (isSubmitting.value) return
  form.value.size = joinResourceSize(size)
  const recognized = applyResourceLinkBlur(
    linkText.value,
    form.value.extraction_code,
    form.value.archive_password
  )
  form.value.download_urls = recognized.links.length
    ? recognized.links
    : splitResourceLinkText(linkText.value)
  form.value.extraction_code = recognized.code
  form.value.archive_password = recognized.password
  linkText.value = form.value.download_urls.join(', ')
  if (!checkGalgameResourcePublish(form.value)) return

  isSubmitting.value = true
  if (isEditing.value && props.resourceId) {
    const current = source.value
    if (!current) {
      isSubmitting.value = false
      return
    }
    const body: GalgameResourcePatch = {}
    if (form.value.resource_type !== current.resource_type) {
      body.resource_type = form.value.resource_type
    }
    if (form.value.title !== current.title) {
      body.title = form.value.title
    }
    const nextVersion = versionToken(form.value.version_label)
    if (nextVersion !== current.version_label) {
      body.version_label = nextVersion
    }
    if (!sameList(form.value.resource_languages, current.resource_languages)) {
      body.resource_languages = form.value.resource_languages
    }
    if (!sameList(form.value.resource_platforms, current.resource_platforms)) {
      body.resource_platforms = form.value.resource_platforms
    }
    if (!sameList(form.value.resource_runtimes, current.resource_runtimes)) {
      body.resource_runtimes = form.value.resource_runtimes
    }
    if (form.value.size !== current.size) {
      body.size = form.value.size
    }
    if (!sameList(form.value.download_urls, current.download_urls)) {
      body.download_urls = form.value.download_urls
    }
    if (form.value.extraction_code !== current.extraction_code) {
      body.extraction_code = form.value.extraction_code
    }
    if (form.value.archive_password !== current.archive_password) {
      body.archive_password = form.value.archive_password
    }
    if (form.value.content_markdown !== current.content_markdown) {
      body.content_markdown = form.value.content_markdown
    }
    if (Object.keys(body).length === 0) {
      isSubmitting.value = false
      open.value = false
      return
    }
    const result = await settle(
      api.PATCH('/galgame-resources/{resource_id}', {
        params: { path: { resource_id: props.resourceId } },
        body
      })
    )
    isSubmitting.value = false
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    useMessage(10550, 'success')
    props.refresh()
    open.value = false
    return
  }

  const nextVersion = versionToken(form.value.version_label)
  const payload: GalgameResourceCreate = {
    resource_type: form.value.resource_type,
    resource_languages: form.value.resource_languages,
    resource_platforms: form.value.resource_platforms,
    title: form.value.title,
    size: form.value.size,
    download_urls: form.value.download_urls,
    extraction_code: form.value.extraction_code,
    archive_password: form.value.archive_password,
    content_markdown: form.value.content_markdown,
    ...(hasRuntimeAxis(form.value.resource_type)
      ? { resource_runtimes: form.value.resource_runtimes }
      : {}),
    ...(nextVersion ? { version_label: nextVersion } : {})
  }
  const result = await settle(
    api.POST('/works/{work_id}/resources', {
      params: {
        path: { work_id: props.workId },
        header: {
          'Idempotency-Key': createKey.take(
            `/works/${props.workId}/resources`,
            payload
          )
        }
      },
      body: payload
    })
  )
  isSubmitting.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  createKey.clear()
  useMessage(10549, 'success')
  props.refresh()
  open.value = false
}

const handleCancel = () => {
  open.value = false
}

const showRuntime = computed(() => hasRuntimeAxis(form.value.resource_type))
watch(
  () => form.value.resource_type,
  (typ) => {
    if (!hasRuntimeAxis(typ)) form.value.resource_runtimes = []
  }
)
const typeOptions = computed(() => {
  const opts = [...RESOURCE_TYPE_OPTIONS]
  const current = form.value.resource_type
  if (current && !opts.some((o) => o.value === current)) {
    opts.push({ value: current, label: resourceTypeLabel(current) })
  }
  return opts
})
</script>

<template>
  <KunModal
    v-model="open"
    inner-class-name="max-w-3xl w-[92vw]"
    :is-dismissable="false"
    :aria-label="modalTitle"
  >
    <div class="space-y-5">
      <div class="space-y-1">
        <h2 class="text-lg font-semibold">{{ modalTitle }}</h2>
        <p class="text-default-500 text-sm">{{ modalSubtitle }}</p>
      </div>

      <KunLoading v-if="isLoadingSource" />

      <template v-else>
        <GalgameResourceHelp />

        <ResourceLinkInput
          v-model="linkText"
          v-model:code="form.extraction_code"
          v-model:password="form.archive_password"
          label="资源链接"
          required
          description="网盘 / 磁链 / 网址。可直接粘贴分享文本，多链接用逗号分隔"
          placeholder="https://..."
        />

        <KunInput
          v-model="form.title"
          label="资源标题（可选）"
          description="显示在资源卡片上，用来区分同一类型的多份资源"
          placeholder="例如 全年龄补丁 / PC+安卓直装"
        />

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div class="space-y-1 sm:col-span-2">
            <div class="flex items-end gap-2">
              <div class="min-w-0 flex-1">
                <KunInput
                  ref="sizeInput"
                  :model-value="size.amount"
                  label="资源体积"
                  required
                  placeholder="3.8 或 500MB"
                  inputmode="decimal"
                  @update:model-value="onSizeAmount"
                  @blur="paintSizeAmount"
                />
              </div>
              <KunRadioGroup
                v-model="size.unit"
                :options="FILE_SIZE_UNITS"
                variant="pill"
                orientation="horizontal"
                color="primary"
                size="md"
                aria-label="资源体积单位"
                class-name="mb-px w-auto shrink-0"
              />
            </div>
            <p class="text-default-500 text-xs">
              可直接输入 500MB / 3.8GB，单位会跟着变
            </p>
          </div>
          <KunInput v-model="form.extraction_code" label="提取码（可选）" />
          <KunInput v-model="form.archive_password" label="解压码（可选）" />
        </div>

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <KunSelect
            v-model="form.resource_type"
            label="资源类型"
            :options="typeOptions"
          >
            <span>{{ resourceTypeLabel(form.resource_type) }}</span>
          </KunSelect>
          <KunSelect
            v-model="form.version_label"
            label="适配版本（可选）"
            :options="VERSION_LABEL_OPTIONS"
            clearable
          />
          <KunSelect
            v-model="form.resource_languages"
            label="语言（可多选）"
            :options="LANGUAGE_OPTIONS"
            multiple
          />
          <KunSelect
            v-model="form.resource_platforms"
            label="平台（可多选）"
            :options="PLATFORM_OPTIONS"
            multiple
            searchable
            search-placeholder="搜索平台"
          />
          <div v-if="showRuntime" class="sm:col-span-2">
            <KunSelect
              v-model="form.resource_runtimes"
              label="运行环境（可多选）"
              :options="RUNTIME_OPTIONS"
              multiple
            />
          </div>
        </div>

        <div class="space-y-1">
          <p class="text-default-700 text-sm font-medium">资源备注（可选）</p>
          <p class="text-default-500 text-xs">
            注意事项 / 介绍 / 作者信息，支持 Markdown 与图片
          </p>
          <KunMilkdownDualEditorProvider
            :key="editorKey"
            :value-markdown="form.content_markdown"
            @set-markdown="(v) => (form.content_markdown = v)"
          />
        </div>

        <div class="flex justify-end gap-2">
          <KunButton variant="light" color="default" @click="handleCancel">
            取消
          </KunButton>
          <KunButton
            variant="solid"
            color="primary"
            :loading="isSubmitting"
            @click="handleSubmit"
          >
            {{ submitLabel }}
          </KunButton>
        </div>
      </template>
    </div>
  </KunModal>
</template>
