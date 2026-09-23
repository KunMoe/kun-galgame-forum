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
  workId: number
  resource?: GalgameResourceDetailLink | null
  refresh: () => void
}>()

const open = defineModel<boolean>({ required: true })

const nuxtApp = useNuxtApp()

const isEditing = computed(() => !!props.resource)

const modalTitle = computed(() =>
  isEditing.value ? '重新编辑资源信息' : '发布 Galgame 资源'
)
const modalSubtitle = computed(() =>
  isEditing.value
    ? '修改链接 / 提取码 / 备注等信息, 保存后立即生效。'
    : '为这部 Galgame 提交一份新的资源链接, 提交后立即对所有用户可见。'
)
const submitLabel = computed(() => (isEditing.value ? '保存修改' : '发布资源'))

interface FormShape {
  type: string
  title: string
  version_label: string
  link: string[]
  language: string
  platform: string
  languages: string[]
  platforms: string[]
  runtimes: string[]
  size: string
  code: string
  password: string
  note: string
}

const defaultForm = (): FormShape => ({
  type: 'game',
  title: '',
  version_label: '',
  link: [],
  language: 'zh-cn',
  platform: 'windows',
  languages: ['zh-cn'],
  platforms: ['win'],
  runtimes: ['native-win'],
  size: '',
  code: '',
  password: '',
  note: ''
})

const snapshotFromResource = (): FormShape => {
  const r = props.resource
  if (!r) return defaultForm()
  const languages = r.languages?.length
    ? r.languages
    : r.language === 'others'
      ? ['other']
      : [r.language]
  const platforms = r.platforms?.length
    ? r.platforms
    : r.platform === 'windows'
      ? ['win']
      : r.platform === 'app'
        ? ['and']
        : r.platform === 'linux'
          ? ['lin']
          : r.platform === 'mac'
            ? ['mac']
            : r.platform === 'others'
              ? ['oth']
              : []
  const runtimes = r.runtimes?.length
    ? r.runtimes
    : r.platform === 'windows'
      ? ['native-win']
      : r.platform === 'app'
        ? ['native-and']
        : []
  return {
    type: r.type === 'others' || r.type === 'ai' ? 'other' : r.type,
    title: r.title ?? '',
    version_label: r.version_label ?? '',
    link: [...r.link],
    language: r.language,
    platform: r.platform,
    languages,
    platforms,
    runtimes,
    size: r.size,
    code: r.code,
    password: r.password,
    note: r.note
  }
}

const form = ref<FormShape>(snapshotFromResource())
const linkText = ref(form.value.link.join(', '))
const size = reactive(splitResourceSize(form.value.size))
const sizeInput = ref<{ inputRef?: HTMLInputElement | null } | null>(null)

const paintSizeAmount = () => {
  const el = sizeInput.value?.inputRef
  if (el && el.value !== size.amount) el.value = size.amount
}

watch(open, (isOpen) => {
  if (!isOpen) return
  form.value = snapshotFromResource()
  linkText.value = form.value.link.join(', ')
  Object.assign(size, splitResourceSize(form.value.size))
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

const handleSubmit = async () => {
  if (isSubmitting.value) return
  form.value.size = joinResourceSize(size)
  const recognized = applyResourceLinkBlur(
    linkText.value,
    form.value.code,
    form.value.password
  )
  form.value.link = recognized.links.length
    ? recognized.links
    : splitResourceLinkText(linkText.value)
  form.value.code = recognized.code
  form.value.password = recognized.password
  linkText.value = form.value.link.join(', ')
  if (!checkGalgameResourcePublish(form.value)) return

  const method = isEditing.value ? 'PUT' : 'POST'
  const body = isEditing.value
    ? {
        ...form.value,
        galgame_id: props.workId,
        galgame_resource_id: props.resource!.id
      }
    : { ...form.value, galgame_id: props.workId }

  isSubmitting.value = true
  const result = await nuxtApp.runWithContext(() =>
    kunFetch(`/galgame/${props.workId}/resource`, { method, body })
  )
  isSubmitting.value = false

  if (result) {
    nuxtApp.runWithContext(() => {
      useMessage(isEditing.value ? 10550 : 10549, 'success')
      props.refresh()
      open.value = false
    })
  }
}

const handleCancel = () => {
  open.value = false
}

const showRuntime = computed(() => hasRuntimeAxis(form.value.type))
watch(
  () => form.value.type,
  (typ) => {
    if (!hasRuntimeAxis(typ)) form.value.runtimes = []
  }
)
const typeOptions = computed(() => {
  const opts = [...RESOURCE_TYPE_OPTIONS]
  const current = form.value.type
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
  >
    <div class="space-y-5">
      <div class="space-y-1">
        <h2 class="text-lg font-semibold">{{ modalTitle }}</h2>
        <p class="text-default-500 text-sm">{{ modalSubtitle }}</p>
      </div>

      <GalgameResourceHelp />

      <ResourceLinkInput
        v-model="linkText"
        v-model:code="form.code"
        v-model:password="form.password"
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
        <KunInput v-model="form.code" label="提取码（可选）" />
        <KunInput v-model="form.password" label="解压码（可选）" />
      </div>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <KunSelect v-model="form.type" label="资源类型" :options="typeOptions">
          <span>{{ resourceTypeLabel(form.type) }}</span>
        </KunSelect>
        <KunSelect
          v-model="form.version_label"
          label="适配版本（可选）"
          :options="VERSION_LABEL_OPTIONS"
          clearable
        />
        <KunSelect
          v-model="form.languages"
          label="语言（可多选）"
          :options="LANGUAGE_OPTIONS"
          multiple
        />
        <KunSelect
          v-model="form.platforms"
          label="平台（可多选）"
          :options="PLATFORM_OPTIONS"
          multiple
          searchable
          search-placeholder="搜索平台"
        />
        <div v-if="showRuntime" class="sm:col-span-2">
          <KunSelect
            v-model="form.runtimes"
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
          :value-markdown="form.note"
          @set-markdown="(v) => (form.note = v)"
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
    </div>
  </KunModal>
</template>
