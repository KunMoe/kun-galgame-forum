<script setup lang="ts">
import type { KunSelectOption, KunTagInputInvalidReason } from '@kungal/ui-vue'
import {
  kunGalgameToolsetTypeOptions,
  kunGalgameToolsetLanguageOptions,
  kunGalgameToolsetPlatformOptions,
  kunGalgameToolsetVersionOptions
} from '~/constants/toolset'
import { toolsetUpdateForm } from '~/components/toolset/rewriteStore'
import { updateToolsetSchema } from '~/validations/toolset'
import { settle } from '#shared/utils/api/problem'
import { problemMessage } from '#shared/utils/api/message'
import type {
  Toolset,
  ToolsetPatch,
  ToolsetSource
} from '#shared/utils/api/schemas'

const isSubmitting = ref(false)
const api = useApiClient()
const toolsetId = computed(() => toolsetUpdateForm.toolset_id)

const { data: toolset, problem: toolsetProblem } = await useApi<Toolset>(
  () => `toolset-rewrite:${toolsetId.value || 'none'}`,
  (apiClient, { signal }) => {
    if (!toolsetId.value) {
      return Promise.resolve({
        data: undefined,
        response: new Response(null, { status: 200 })
      })
    }
    return apiClient.GET('/toolsets/{toolset_id}', {
      params: { path: { toolset_id: toolsetId.value } },
      signal
    })
  }
)

const { data: source, problem: sourceProblem } = await useApi<ToolsetSource>(
  () => `toolset-source:${toolsetId.value || 'none'}`,
  (apiClient, { signal }) => {
    if (!toolsetId.value) {
      return Promise.resolve({
        data: undefined,
        response: new Response(null, { status: 200 })
      })
    }
    return apiClient.GET('/toolsets/{toolset_id}/source', {
      params: { path: { toolset_id: toolsetId.value } },
      signal
    })
  }
)

type Snapshot = {
  title: string
  content_markdown: string
  interface_language: Toolset['interface_language']
  platform: Toolset['platform']
  toolset_type: Toolset['toolset_type']
  release_channel: Toolset['release_channel']
  homepage_urls: string[]
  aliases: string[]
}

const original = ref<Snapshot | null>(null)

const applyLoaded = () => {
  if (!toolset.value || !source.value) {
    return
  }
  toolsetUpdateForm.title = toolset.value.title
  toolsetUpdateForm.content_markdown = source.value.content_markdown
  toolsetUpdateForm.interface_language = toolset.value.interface_language
  toolsetUpdateForm.platform = toolset.value.platform
  toolsetUpdateForm.toolset_type = toolset.value.toolset_type
  toolsetUpdateForm.release_channel = toolset.value.release_channel
  toolsetUpdateForm.homepage_urls = [...toolset.value.homepage_urls]
  toolsetUpdateForm.aliases = [...toolset.value.aliases]
  original.value = {
    title: toolset.value.title,
    content_markdown: source.value.content_markdown,
    interface_language: toolset.value.interface_language,
    platform: toolset.value.platform,
    toolset_type: toolset.value.toolset_type,
    release_channel: toolset.value.release_channel,
    homepage_urls: [...toolset.value.homepage_urls],
    aliases: [...toolset.value.aliases]
  }
}

watch([toolset, source], applyLoaded, { immediate: true })

const loadProblem = computed(
  () => toolsetProblem.value ?? sourceProblem.value ?? null
)

const onAliasInvalid = (reason: KunTagInputInvalidReason) => {
  if (reason === 'duplicate') useMessage(10505, 'warn')
  else if (reason === 'max-reached') useMessage(10508, 'warn')
}

const patchFromChanges = (): ToolsetPatch => {
  const before = original.value
  const body: ToolsetPatch = {}
  if (!before) {
    return body
  }
  if (toolsetUpdateForm.title !== before.title) {
    body.title = toolsetUpdateForm.title
  }
  if (toolsetUpdateForm.content_markdown !== before.content_markdown) {
    body.content_markdown = toolsetUpdateForm.content_markdown
  }
  if (toolsetUpdateForm.interface_language !== before.interface_language) {
    body.interface_language = toolsetUpdateForm.interface_language
  }
  if (toolsetUpdateForm.platform !== before.platform) {
    body.platform = toolsetUpdateForm.platform
  }
  if (toolsetUpdateForm.toolset_type !== before.toolset_type) {
    body.toolset_type = toolsetUpdateForm.toolset_type
  }
  if (toolsetUpdateForm.release_channel !== before.release_channel) {
    body.release_channel = toolsetUpdateForm.release_channel
  }
  if (
    JSON.stringify(toolsetUpdateForm.homepage_urls) !==
    JSON.stringify(before.homepage_urls)
  ) {
    body.homepage_urls = toolsetUpdateForm.homepage_urls
  }
  if (
    JSON.stringify(toolsetUpdateForm.aliases) !== JSON.stringify(before.aliases)
  ) {
    body.aliases = toolsetUpdateForm.aliases
  }
  return body
}

const handleSubmit = async () => {
  const result = updateToolsetSchema.safeParse(toolsetUpdateForm)
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }
  if (isSubmitting.value || !toolsetId.value) {
    return
  }

  const body = patchFromChanges()
  if (Object.keys(body).length === 0) {
    navigateTo(`/toolset/${toolsetId.value}`)
    return
  }

  isSubmitting.value = true
  const res = await settle(
    api.PATCH('/toolsets/{toolset_id}', {
      params: { path: { toolset_id: toolsetId.value } },
      body
    })
  )
  isSubmitting.value = false

  if (!res.ok) {
    reportProblem(res.problem)
    return
  }

  useMessage('更新工具信息成功', 'success')
  navigateTo(`/toolset/${toolsetId.value}`)
}

const handleUpdatePageLink = (value: string | number) => {
  toolsetUpdateForm.homepage_urls = value
    .toString()
    .split(',')
    .map((l) => l.trim())
    .filter(Boolean)
}
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="编辑工具信息"
      description="更新你发布的 Galgame 工具信息"
    />

    <KunNull
      v-if="!toolsetId"
      description="未找到要编辑的工具，请从工具详情页进入"
    />

    <KunNull
      v-else-if="loadProblem"
      :description="problemMessage(loadProblem)"
    />

    <template v-else>
      <div class="space-y-2">
        <label class="text-sm font-medium">名称</label>
        <KunInput v-model="toolsetUpdateForm.title" placeholder="工具名称" />
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <KunSelect
          v-model="toolsetUpdateForm.toolset_type"
          label="工具类型"
          :options="
            kunGalgameToolsetTypeOptions.filter(
              (o) => o.value !== 'all'
            ) as KunSelectOption<
              Exclude<
                (typeof kunGalgameToolsetTypeOptions)[number]['value'],
                'all'
              >
            >[]
          "
        />
        <KunSelect
          v-model="toolsetUpdateForm.release_channel"
          label="版本"
          :options="
            kunGalgameToolsetVersionOptions.filter(
              (o) => o.value !== 'all'
            ) as KunSelectOption<
              Exclude<
                (typeof kunGalgameToolsetVersionOptions)[number]['value'],
                'all'
              >
            >[]
          "
        />
      </div>

      <div class="space-y-2">
        <div class="text-xl font-medium">简介</div>
        <p class="text-default-500 text-sm">
          请在此处具体说明工具是什么, 以及如何使用该工具, 越详细越好
        </p>
        <KunMilkdownDualEditorProvider
          :key="source?.toolset_id ?? 'empty'"
          :value-markdown="toolsetUpdateForm.content_markdown"
          @set-markdown="(value) => (toolsetUpdateForm.content_markdown = value)"
          language="zh-cn"
        >
          <KunLink target="_blank" to="/doc/create-galgame-toolset">
            发布 Galgame 工具规定
          </KunLink>
        </KunMilkdownDualEditorProvider>
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <KunSelect
          v-model="toolsetUpdateForm.platform"
          label="平台"
          :options="
            kunGalgameToolsetPlatformOptions.filter(
              (o) => o.value !== 'all'
            ) as KunSelectOption<
              Exclude<
                (typeof kunGalgameToolsetPlatformOptions)[number]['value'],
                'all'
              >
            >[]
          "
        />
        <KunSelect
          v-model="toolsetUpdateForm.interface_language"
          label="语言"
          :options="
            kunGalgameToolsetLanguageOptions.filter(
              (o) => o.value !== 'all'
            ) as KunSelectOption<
              Exclude<
                (typeof kunGalgameToolsetLanguageOptions)[number]['value'],
                'all'
              >
            >[]
          "
        />
      </div>

      <div class="space-y-2">
        <div class="text-sm font-medium">主页 / 下载链接</div>
        <KunTextarea
          :model-value="toolsetUpdateForm.homepage_urls.toString()"
          @update:model-value="handleUpdatePageLink"
          placeholder="如果有多个页面链接, 需要用英语逗号分隔每个链接"
        />
      </div>

      <div class="space-y-2">
        <div class="text-sm font-medium">别名</div>
        <KunTagInput
          v-model="toolsetUpdateForm.aliases"
          :max-tags="17"
          :max-tag-length="500"
          placeholder="输入别名后回车"
          description="按 Enter 添加，最多 17 个"
          color="primary"
          @invalid="onAliasInvalid"
        />
      </div>

      <div class="flex justify-end">
        <KunButton :loading="isSubmitting" @click="handleSubmit">
          保存
        </KunButton>
      </div>
    </template>
  </div>
</template>
