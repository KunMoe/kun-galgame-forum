<script setup lang="ts">
const props = defineProps<{ message: AgentMessage; busy: boolean }>()
const emit = defineEmits<{
  edit: [text: string]
  retry: []
  switch: [id: string]
}>()

const editing = ref(false)
const draft = ref('')

const text = computed(() => messageText(props.message))
const files = computed(() =>
  props.message.parts.filter(
    (p): p is Extract<AgentPart, { type: 'file' }> => p.type === 'file'
  )
)
const canSend = computed(
  () =>
    !props.busy &&
    draft.value.trim().length > 0 &&
    draft.value.trim() !== text.value.trim()
)

const startEdit = () => {
  draft.value = text.value
  editing.value = true
}

const submit = () => {
  if (!canSend.value) return
  emit('edit', draft.value.trim())
  editing.value = false
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') editing.value = false
  if (e.key !== 'Enter' || e.shiftKey || e.isComposing) return
  e.preventDefault()
  submit()
}
</script>

<template>
  <div class="group flex flex-col items-end gap-1">
    <AgentMessageFiles v-if="files.length" :files="files" />

    <div
      v-if="editing"
      class="border-default-200 bg-content1 rounded-kun-lg focus-within:border-primary/50 flex w-full max-w-[85%] flex-col gap-2 border p-2"
    >
      <KunTextarea
        v-model="draft"
        label="编辑消息"
        :rows="1"
        auto-grow
        max-height="320px"
        resize="none"
        autofocus
        class="[&_label]:sr-only [&_textarea]:border-0 [&_textarea]:bg-transparent [&_textarea]:shadow-none [&_textarea]:ring-0 [&_textarea:focus]:border-transparent [&_textarea:focus]:shadow-none [&_textarea:focus]:ring-0"
        @keydown="onKeydown"
      />
      <div class="flex justify-end gap-2">
        <KunButton variant="light" size="sm" @click="editing = false">
          取消
        </KunButton>
        <KunButton
          color="primary"
          size="sm"
          :disabled="!canSend"
          @click="submit"
        >
          发送
        </KunButton>
      </div>
    </div>

    <div
      v-else-if="text"
      class="bg-default-100 rounded-kun-lg max-w-[85%] px-4 py-2.5 break-words whitespace-pre-wrap"
    >
      {{ text }}
    </div>

    <div v-if="!editing" class="flex items-center gap-0.5">
      <div
        class="flex items-center gap-0.5 transition-opacity group-hover:opacity-100 focus-within:opacity-100 md:opacity-0"
      >
        <AgentIconAction
          v-if="text"
          icon="lucide:copy"
          label="复制"
          @click="useKunCopy(text)"
        />
        <AgentIconAction
          icon="lucide:pencil"
          label="编辑"
          :disabled="busy"
          @click="startEdit"
        />
        <AgentIconAction
          icon="lucide:rotate-cw"
          label="重试"
          :disabled="busy"
          @click="emit('retry')"
        />
      </div>
      <AgentVersionSwitcher
        :ids="message.sibling_ids"
        :current="message.id"
        :disabled="busy"
        @switch="emit('switch', $event)"
      />
    </div>
  </div>
</template>
