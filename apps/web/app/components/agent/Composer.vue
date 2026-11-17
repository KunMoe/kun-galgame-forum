<script setup lang="ts">
import type { AgentFilePart } from '~/utils/agentUpload'

const props = defineProps<{
  busy: boolean
  submit: (text: string, files: AgentFilePart[]) => Promise<boolean>
}>()
const emit = defineEmits<{ stop: [] }>()

const attachments = useAgentAttachments()
const text = ref('')
const sending = ref(false)

const canSend = computed(
  () =>
    !props.busy &&
    !sending.value &&
    !attachments.uploading.value &&
    (text.value.trim().length > 0 || attachments.files.value.length > 0)
)

const send = async () => {
  if (!canSend.value) return
  sending.value = true
  const ok = await props.submit(text.value.trim(), attachments.files.value)
  sending.value = false
  if (ok) {
    text.value = ''
    attachments.clear()
  }
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key !== 'Enter' || e.shiftKey || e.isComposing) return
  e.preventDefault()
  void send()
}
</script>

<template>
  <div
    class="border-default-200 bg-content1 rounded-kun-lg focus-within:border-primary/50 flex flex-col gap-2 border p-2 transition-colors"
  >
    <AgentAttachments
      v-if="attachments.items.value.length"
      :items="attachments.items.value"
      @remove="attachments.remove"
    />
    <KunTextarea
      v-model="text"
      label="给 AI 助手发消息"
      placeholder="想找什么样的作品？描述剧情、角色或者氛围就好"
      :rows="1"
      auto-grow
      max-height="224px"
      resize="none"
      class="[&_label]:sr-only [&_textarea]:border-0 [&_textarea]:bg-transparent [&_textarea]:shadow-none [&_textarea]:ring-0 [&_textarea:focus]:border-transparent [&_textarea:focus]:shadow-none [&_textarea:focus]:ring-0"
      @keydown="onKeydown"
    />
    <div class="flex items-center gap-1">
      <AgentUploadMenu
        v-if="attachments.options.value.length"
        :options="attachments.options.value"
        :full="attachments.items.value.length >= attachments.maxFiles.value"
        @pick="(o, files) => attachments.add(files, o)"
      />
      <AgentSettings />
      <span class="text-default-400 ml-auto hidden text-xs sm:inline">
        Enter 发送，Shift+Enter 换行
      </span>
      <KunButton
        v-if="busy"
        color="danger"
        variant="flat"
        size="sm"
        is-icon-only
        aria-label="停止生成"
        class="ml-auto sm:ml-2"
        @click="emit('stop')"
      >
        <KunIcon name="lucide:square" class="icon-solid size-4" />
      </KunButton>
      <KunButton
        v-else
        color="primary"
        size="sm"
        is-icon-only
        aria-label="发送"
        class="ml-auto sm:ml-2"
        :disabled="!canSend"
        :loading="sending"
        @click="send"
      >
        <KunIcon name="lucide:arrow-up" class="size-5" />
      </KunButton>
    </div>
  </div>
</template>
