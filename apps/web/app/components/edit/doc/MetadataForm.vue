<script setup lang="ts">
import { KUN_DOC_CATEGORIES, KUN_DOC_CATEGORY_MAP } from '~/constants/doc'
import { useDocEditorContext } from './context'
import { normalizeDocSlug } from '~/utils/doc'

const { form, readingMinute, initialBannerUrl } = useDocEditorContext()

const categoryOptions = KUN_DOC_CATEGORIES.map((category) => ({
  label: KUN_DOC_CATEGORY_MAP[category],
  value: category
}))

const normalizeSlug = () => {
  form.slug = normalizeDocSlug(form.slug)
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h3 class="text-lg font-semibold">基础信息</h3>
      <p class="text-default-500 text-sm">
        slug 会自动拼接为 /doc/[slug] 作为访问路径，无需手动填写。
      </p>
    </div>

    <div class="space-y-4">
      <KunInput
        v-model="form.slug"
        label="Slug"
        placeholder="请输入唯一的文档 slug"
        maxlength="128"
        required
        @blur="normalizeSlug"
      />

      <KunCoverUpload
        v-model="form.banner_image_hash"
        :preview-url="initialBannerUrl"
        label="封面"
      />

      <KunTextarea
        v-model="form.description"
        label="文档简介"
        placeholder="请输入用于展示的简介"
        :rows="4"
        auto-grow
        :maxlength="777"
        required
        show-char-count
      />
    </div>

    <div class="space-y-4">
      <KunSelect
        v-model="form.doc_category"
        :options="categoryOptions"
        label="文档分类"
        placeholder="请选择分类"
      />

      <div
        class="border-default-200 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-2"
      >
        <div>
          <p class="text-sm font-medium">预计阅读时长</p>
          <p class="text-default-500 text-xs">根据当前正文实时计算</p>
        </div>
        <span class="text-primary font-semibold">
          {{ readingMinute }} 分钟
        </span>
      </div>

      <div
        class="border-default-200 flex items-center justify-between rounded-lg border px-4 py-2"
      >
        <div>
          <p class="text-sm font-medium">首页置顶</p>
          <p class="text-default-500 text-xs">开启后会在首页轮播中固定展示</p>
        </div>
        <KunSwitch v-model="form.is_pinned" color="primary" />
      </div>
    </div>
  </div>
</template>
