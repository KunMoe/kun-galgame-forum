<script setup lang="ts">
import type { KunSelectOption, KunTagInputInvalidReason } from '@kungal/ui-vue'
import {
  kunGalgameToolsetTypeOptions,
  kunGalgameToolsetLanguageOptions,
  kunGalgameToolsetPlatformOptions,
  kunGalgameToolsetVersionOptions
} from '~/constants/toolset'
import { createToolsetSchema } from '~/validations/toolset'
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import type { ToolsetCreate } from '#shared/utils/api/schemas'

const form = reactive<Required<ToolsetCreate>>({
  title: '',
  content_markdown: '',
  interface_language: 'zh-cn',
  platform: 'windows',
  toolset_type: 'emulator',
  release_channel: 'stable',
  homepage_urls: [] as string[],
  aliases: [] as string[]
})

const onAliasInvalid = (reason: KunTagInputInvalidReason) => {
  if (reason === 'duplicate') useMessage(10505, 'warn')
  else if (reason === 'max-reached') useMessage(10508, 'warn')
}

const isSubmitting = ref(false)
const api = useApiClient()
const createKey = useIdempotencyKey()

const handleSubmit = async () => {
  const result = createToolsetSchema.safeParse(form)
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }
  if (isSubmitting.value) {
    return
  }

  const payload = result.data
  isSubmitting.value = true
  const created = await settle(
    api.POST('/toolsets', {
      params: {
        header: {
          'Idempotency-Key': createKey.take('/toolsets', payload)
        }
      },
      body: payload
    })
  )
  isSubmitting.value = false

  if (!created.ok) {
    reportProblem(created.problem)
    return
  }
  createKey.clear()
  useMessage('创建工具成功', 'success')
  navigateTo(`/toolset/${created.data.id}`)
}

const handleUpdatePageLink = (value: string | number) => {
  form.homepage_urls = value
    .toString()
    .split(',')
    .map((l) => l.trim())
    .filter(Boolean)
}
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="发布 Galgame 工具"
      description="提交与 Galgame 相关的工具，帮助其他用户下载与使用"
    />

    <div class="space-y-2">
      <div class="text-xl font-medium">名称</div>
      <KunInput v-model="form.title" placeholder="工具名称" />
    </div>

    <div class="grid grid-cols-2 gap-3">
      <KunSelect
        v-model="form.toolset_type"
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
        v-model="form.release_channel"
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
      <KunSelect
        v-model="form.platform"
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
        v-model="form.interface_language"
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
      <div class="text-xl font-medium">简介</div>
      <p class="text-default-500 text-sm">
        请在此处具体说明工具是什么, 以及如何使用该工具, 越详细越好
      </p>
      <KunMilkdownDualEditorProvider
        :value-markdown="form.content_markdown"
        @set-markdown="(value) => (form.content_markdown = value)"
        language="zh-cn"
      >
        <KunLink target="_blank" to="/doc/create-galgame-toolset">
          发布 Galgame 工具规定
        </KunLink>
      </KunMilkdownDualEditorProvider>
    </div>

    <div class="space-y-2">
      <div class="text-xl font-medium">主页</div>
      <KunTextarea
        :model-value="form.homepage_urls.toString()"
        @update:model-value="handleUpdatePageLink"
        placeholder="工具的官网, GitHub 仓库等等, 如果有多个链接, 使用英语逗号分隔每个下载链接"
      />
    </div>

    <div class="space-y-2">
      <div class="text-xl font-medium">别名</div>
      <KunTagInput
        v-model="form.aliases"
        :max-tags="17"
        :max-tag-length="500"
        placeholder="输入别名后按下回车添加"
        description="按 Enter 添加，最多 17 个"
        color="primary"
        @invalid="onAliasInvalid"
      />
    </div>

    <div class="flex justify-end">
      <KunButton :loading="isSubmitting" @click="handleSubmit">
        发布
      </KunButton>
    </div>
  </div>
</template>
