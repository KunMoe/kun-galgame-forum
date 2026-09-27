<script setup lang="ts">
import type { ChatMessage } from '#shared/types/chat'
import { chatProblemMessage } from '#shared/utils/api/chat'

const props = defineProps<{
  message: ChatMessage | null
  conversationId: string
}>()
const emit = defineEmits<{ close: [] }>()

const chat = useChatStore()
const act = useChatActions()
const forEveryone = ref(true)
const busy = ref(false)

const own = computed(() => props.message?.sender_id === chat.model.me)
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
  () => (forEveryone.value = true)
)

const confirm = async () => {
  if (!props.message) {
    return
  }
  busy.value = true
  const r = await act.remove(
    props.conversationId,
    [props.message.seq],
    own.value && forEveryone.value
  )
  busy.value = false
  if (!r.ok) {
    useMessage(chatProblemMessage(r.problem), 'warn')
  }
  emit('close')
}
</script>

<template>
  <KunModal
    v-model="open"
    role="alertdialog"
    title="删除消息"
    inner-class-name="w-full max-w-sm"
  >
    <div class="flex flex-col gap-4">
      <p class="text-default-600 text-sm">
        {{ own ? '确定删除这条消息吗？' : '这条消息只会从你这里删除。' }}
      </p>
      <KunCheckBox v-if="own" v-model="forEveryone" label="同时为对方删除" />
      <div class="flex justify-end gap-2">
        <KunButton variant="light" @click="emit('close')">取消</KunButton>
        <KunButton color="danger" :loading="busy" @click="confirm">
          删除
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
