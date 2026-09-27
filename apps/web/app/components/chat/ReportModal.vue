<script setup lang="ts">
import type { ChatMessage, ChatReportReason } from '#shared/types/chat'
import { chatProblemMessage } from '#shared/utils/api/chat'

const props = defineProps<{ message: ChatMessage | null }>()
const emit = defineEmits<{ close: [] }>()

const act = useChatActions()
const reasonOptions: { label: string; value: ChatReportReason }[] = [
  { label: '垃圾广告', value: 'spam' },
  { label: '骚扰辱骂', value: 'harassment' },
  { label: '色情内容', value: 'sexual' },
  { label: '暴力内容', value: 'violence' },
  { label: '违法内容', value: 'illegal' },
  { label: '其他', value: 'other' }
]
const reason = ref<ChatReportReason>('spam')
const note = ref('')
const busy = ref(false)

const open = computed({
  get: () => !!props.message,
  set: (v) => {
    if (!v) {
      emit('close')
    }
  }
})

watch(
  () => props.message,
  () => {
    reason.value = 'spam'
    note.value = ''
  }
)

const submit = async () => {
  if (!props.message) {
    return
  }
  busy.value = true
  const r = await act.report(props.message, reason.value, note.value.trim())
  busy.value = false
  if (!r.ok) {
    useMessage(chatProblemMessage(r.problem), 'warn')
    return
  }
  useMessage('已提交举报，感谢反馈', 'success')
  emit('close')
}
</script>

<template>
  <KunModal v-model="open" title="举报消息" inner-class-name="w-full max-w-md">
    <div class="flex flex-col gap-3">
      <p class="text-default-500 text-sm">
        举报会把这条消息和它之前的最多 10 条消息交给审核人员查看。
      </p>
      <KunSelect v-model="reason" :options="reasonOptions" label="举报原因" />
      <KunTextarea v-model="note" placeholder="补充说明（可选）" />
      <div class="flex justify-end gap-2">
        <KunButton variant="light" @click="emit('close')">取消</KunButton>
        <KunButton color="danger" :loading="busy" @click="submit">
          提交举报
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
