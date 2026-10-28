<script setup lang="ts">
import { useDocEditorContext } from './context'

const { form, readingMinute } = useDocEditorContext()
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h3 class="text-lg font-semibold">正文内容</h3>
        <p class="text-default-500 text-sm">
          使用 Markdown 撰写文档，系统会自动渲染为 HTML 并生成目录。
        </p>
      </div>
      <div class="text-default-500 text-sm">
        预计阅读时长：
        <span class="text-primary font-semibold">
          {{ readingMinute }} 分钟
        </span>
      </div>
    </div>

    <KunMilkdownDualEditorProvider
      entities
      :value-markdown="form.content_markdown"
      @set-markdown="(value) => (form.content_markdown = value)"
    >
      <template #default>
        <KunLink
          underline="hover"
          to="/doc/topic-publish-rule"
          target="_blank"
          size="sm"
        >
          查看文档撰写规范
        </KunLink>
      </template>
    </KunMilkdownDualEditorProvider>
  </div>
</template>
