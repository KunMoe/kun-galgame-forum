<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const { isOpen, target } = useReportModal()
const { reasons, load } = useReportReasons()
const api = useApiClient()

const reasonKey = ref('')
const note = ref('')
const isSubmitting = ref(false)

const reasonOptions = computed(() =>
  reasons.value.map((r) => ({ value: r.key, label: r.display_name }))
)

watch(isOpen, (open) => {
  if (open) {
    load()
    reasonKey.value = ''
    note.value = ''
  }
})

const submit = async () => {
  if (!target.value) {
    return
  }
  if (!reasonKey.value) {
    useMessage('请选择举报理由', 'warn')
    return
  }
  isSubmitting.value = true
  const result = await settle(
    api.POST('/reports', {
      body: {
        subject_kind: target.value.subjectKind,
        subject_id: String(target.value.subjectId),
        reason_key: reasonKey.value,
        note: note.value.trim() || null,
        snapshot: (target.value.snapshot ?? '').slice(0, 1000) || null,
        subject_url: target.value.subjectUrl || null
      }
    })
  )
  isSubmitting.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('举报已提交，感谢你的反馈', 'success')
  isOpen.value = false
}
</script>

<template>
  <KunModal v-model="isOpen" inner-class-name="max-w-md w-[92vw]">
    <div class="space-y-4">
      <div>
        <span class="text-xl">举报内容</span>
        <p class="text-default-500 text-sm">
          举报对其他用户匿名；为防止滥用，平台会记录举报人。请选择理由并按需补充说明。
        </p>
      </div>

      <KunSelect
        v-model="reasonKey"
        :options="reasonOptions"
        label="举报理由"
        placeholder="请选择举报理由"
      />

      <div class="space-y-2">
        <span class="text-default-600 text-sm font-medium">补充说明</span>
        <KunTextarea
          name="report-note"
          placeholder="请尽量清晰、详细地描述问题：违规的具体内容是什么、出现在页面的哪个位置、为什么违规。描述越具体，我们越能快速准确地处理；一句「违规」「不当」等模糊反馈往往无法受理。"
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
