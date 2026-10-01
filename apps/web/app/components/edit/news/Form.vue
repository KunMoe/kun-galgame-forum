<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import { KUN_NEWS_PREVIEW_MAX, newsSubmissionSchema } from '~/validations/news'

const props = defineProps<{ initial?: KunNewsSubmission }>()

const form = reactive({
  title: props.initial?.title ?? '',
  preview: props.initial?.preview ?? '',
  content_markdown: props.initial?.content_markdown ?? '',
  source_url: props.initial?.source_url ?? ''
})

const api = useApiClient()
const createKey = useIdempotencyKey()
const isSubmitting = ref(false)

type NewsForm = typeof form

const savedBannerHash = props.initial?.banner?.hash ?? ''
const bannerHash = ref(savedBannerHash)
const bannerPreview = ref(props.initial?.banner?.url ?? '')
const pendingBanner = ref<Blob | null>(null)
const bannerPickerKey = ref(0)

const handlePickBanner = (image: Blob) => {
  pendingBanner.value = image
}

const handleRemoveBanner = () => {
  pendingBanner.value = null
  bannerHash.value = ''
  bannerPreview.value = ''
  bannerPickerKey.value += 1
}

// The picked image goes up only when the form is submitted, so cropping again
// or leaving the page spends none of NextMoe's 30 banner uploads a day.
const uploadPendingBanner = async () => {
  if (!pendingBanner.value) {
    return true
  }
  const image = await uploadNewsSubmissionImage(
    pendingBanner.value,
    'banner.webp'
  )
  if (!image) {
    return false
  }
  bannerHash.value = image.hash
  bannerPreview.value = image.url
  pendingBanner.value = null
  return true
}

const create = async (data: NewsForm) => {
  const payload: KunNewsSubmissionCreate = {
    title: data.title,
    preview: data.preview,
    ...(data.content_markdown.trim()
      ? { content_markdown: data.content_markdown }
      : {}),
    ...(data.source_url ? { source_url: data.source_url } : {}),
    ...(bannerHash.value ? { banner_image_hash: bannerHash.value } : {})
  }
  const result = await settle(
    api.POST('/me/news-submissions', {
      params: {
        header: {
          'Idempotency-Key': createKey.take('/me/news-submissions', payload)
        }
      },
      body: payload
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  createKey.clear()
  useMessage('投稿成功，审核通过后会出现在 Galgame 情报中', 'success', 5000)
  return true
}

const update = async (initial: KunNewsSubmission, data: NewsForm) => {
  const body: KunNewsSubmissionPatch = {}
  if (data.title !== initial.title) {
    body.title = data.title
  }
  if (data.preview !== initial.preview) {
    body.preview = data.preview
  }
  if (data.content_markdown !== initial.content_markdown) {
    body.content_markdown = data.content_markdown
  }
  if (data.source_url !== initial.source_url) {
    body.source_url = data.source_url
  }
  if (bannerHash.value !== savedBannerHash) {
    body.banner_image_hash = bannerHash.value
  }
  if (!Object.keys(body).length) {
    return true
  }
  const result = await settle(
    api.PATCH('/me/news-submissions/{news_submission_id}', {
      params: { path: { news_submission_id: initial.id } },
      body
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  useMessage(
    initial.state === 'published' ? '已保存，情报将重新进入审核' : '已保存',
    'success'
  )
  return true
}

const handleSubmit = async () => {
  if (isSubmitting.value) {
    return
  }
  const parsed = newsSubmissionSchema.safeParse(form)
  if (!parsed.success) {
    useMessage(parsed.error.issues[0]?.message ?? '请检查填写内容', 'warn')
    return
  }
  isSubmitting.value = true
  const saved =
    (await uploadPendingBanner()) &&
    (props.initial
      ? await update(props.initial, parsed.data)
      : await create(parsed.data))
  isSubmitting.value = false
  if (saved) {
    await navigateTo('/news/mine')
  }
}
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      :name="initial ? '编辑 Gal 情报' : '发布 Gal 情报'"
      description="分享 Galgame 新作情报、发售消息、业界动态或你自己写的专栏。投稿由 NextMoe 审核员人工审核，通过后才会出现在 Galgame 情报中。"
    >
      <template #endContent>
        <KunButton variant="light" size="sm" href="/news/mine">
          我的投稿
        </KunButton>
      </template>
    </KunHeader>

    <KunInfo
      v-if="initial?.state === 'published'"
      color="warning"
      icon="lucide:info"
      title="这条情报已经发布"
      description="保存修改后它会暂时下线，重新审核通过后再次公开。"
    />

    <div class="space-y-2">
      <div class="text-xl font-medium">标题</div>
      <KunInput v-model="form.title" placeholder="一句话说清这条情报" />
    </div>

    <div class="space-y-2">
      <div class="text-xl font-medium">导语</div>
      <p class="text-default-500 text-sm">
        显示在情报列表里的简短介绍，最多 {{ KUN_NEWS_PREVIEW_MAX }} 字
      </p>
      <KunTextarea
        v-model="form.preview"
        :rows="3"
        :maxlength="KUN_NEWS_PREVIEW_MAX"
        show-char-count
        placeholder="例如: 某某社新作《……》公开，预计 2027 年春季发售"
      />
    </div>

    <div class="space-y-2">
      <div class="text-xl font-medium">封面（可选）</div>
      <p class="text-default-500 text-sm">
        显示在情报卡片和详情页顶部，会裁切为 16:9 的横图
      </p>
      <KunUpload
        :key="bannerPickerKey"
        :initial-image="bannerPreview"
        :size="1280"
        :aspect="16 / 9"
        description="封面不可包含 R18 等敏感内容"
        class-name="w-64"
        @set-image="handlePickBanner"
      />
      <KunButton
        v-if="bannerPreview || pendingBanner"
        variant="light"
        color="danger"
        size="sm"
        @click="handleRemoveBanner"
      >
        移除封面
      </KunButton>
    </div>

    <div class="space-y-2">
      <div class="text-xl font-medium">正文</div>
      <p class="text-default-500 text-sm">
        原创内容写在这里。转载请只写摘要并附上原文链接。正文与原文链接至少填写一项
      </p>
      <KunMilkdownDualEditorProvider
        :value-markdown="form.content_markdown"
        :disable-image="true"
        language="zh-cn"
        placeholder="支持 Markdown，不支持 HTML"
        @set-markdown="(value: string) => (form.content_markdown = value)"
      />
    </div>

    <div class="space-y-2">
      <div class="text-xl font-medium">原文链接</div>
      <KunInput
        v-model="form.source_url"
        type="url"
        placeholder="https://..."
      />
    </div>

    <div class="flex justify-end">
      <KunButton :loading="isSubmitting" @click="handleSubmit">
        {{ initial ? '保存修改' : '提交投稿' }}
      </KunButton>
    </div>
  </div>
</template>
