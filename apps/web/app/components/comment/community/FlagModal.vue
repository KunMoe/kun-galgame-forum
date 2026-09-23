<script setup lang="ts">
import type { WallFlagReason } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const { isOpen, targetPostId } = useGalgameCommentFlag()
const api = useApiClient()

const REASON_OPTIONS: { value: WallFlagReason; label: string }[] = [
  { value: 'spam', label: '垃圾信息' },
  { value: 'abuse', label: '辱骂骚扰' },
  { value: 'off_topic', label: '离题' },
  { value: 'other', label: '其他' },
  { value: 'nsfw_mislabel', label: '分级标注错误' }
]

const reason = ref<WallFlagReason | null>(null)
const note = ref('')
const isSubmitting = ref(false)

watch(isOpen, (open) => {
  if (open) {
    reason.value = null
    note.value = ''
  }
})

const submit = async () => {
  const postId = targetPostId.value
  if (postId == null) {
    return
  }
  if (reason.value == null) {
    useMessage('请选择举报理由', 'warn')
    return
  }
  if (note.value.length > 500) {
    useMessage('说明最多 500 个字符', 'warn')
    return
  }

  isSubmitting.value = true
  const result = await settle(
    api.POST('/wall-comments/{wall_comment_id}/flags', {
      params: { path: { wall_comment_id: postId } },
      body: { flag_reason: reason.value, ...(note.value ? { note: note.value } : {}) }
    })
  )
  isSubmitting.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('已举报，达到阈值将自动隐藏并进入人工审核', 'success')
  isOpen.value = false
}
</script>

<template>
  <KunModal v-model="isOpen" inner-class-name="max-w-md w-[92vw]">
    <div class="space-y-4">
      <div>
        <span class="text-xl">举报评论</span>
        <p class="text-default-500 text-sm">
          举报对其他用户匿名；为防止滥用，平台会记录举报人。请选择理由并按需补充说明。
        </p>
      </div>

      <KunSelect
        v-model="reason"
        :options="REASON_OPTIONS"
        label="举报理由"
        placeholder="请选择举报理由"
      />

      <div class="space-y-2">
        <span class="text-default-600 text-sm font-medium">补充说明</span>
        <KunTextarea
          name="community-comment-flag-note"
          placeholder="请尽量清晰、详细地描述问题：违规的具体内容是什么、为什么违规。描述越具体，我们越能快速准确地处理。"
          :rows="5"
          v-model="note"
        />
      </div>

      <div class="flex justify-end gap-2">
        <KunButton variant="light" color="default" @click="isOpen = false">
          取消
        </KunButton>
        <KunButton color="danger" :loading="isSubmitting" @click="submit">
          提交举报
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
