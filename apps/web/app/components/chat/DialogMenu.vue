<script setup lang="ts">
import type { ChatConversation } from '#shared/types/chat'
import type { ApiResult } from '#shared/utils/api/problem'
import { chatProblemMessage } from '#shared/utils/api/chat'
import { chatMuted } from '~/utils/chatModel'

const props = defineProps<{ conversation: ChatConversation }>()

const act = useChatActions()
const busy = ref(false)
const confirmClear = ref(false)

const muted = computed(() => chatMuted(props.conversation))
const pinned = computed(() => props.conversation.me.pinned_rank !== null)
const archived = computed(() => props.conversation.me.archived)
const accepted = computed(() => props.conversation.me.accepted)

const run = async <T,>(call: () => Promise<ApiResult<T>>) => {
  busy.value = true
  const r = await call()
  busy.value = false
  if (!r.ok) {
    useMessage(chatProblemMessage(r.problem), 'warn')
  }
  return r
}

const setDialog = (patch: Record<string, unknown>) =>
  run(() => act.setDialog(props.conversation.id, patch))

const clear = async () => {
  const r = await run(() => act.clearHistory(props.conversation.id, false))
  if (r.ok) {
    confirmClear.value = false
  }
}

const itemClass =
  'hover:bg-default-100 flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm'
</script>

<template>
  <KunPopover position="bottom-end" inner-class="min-w-40 p-1.5">
    <template #trigger>
      <KunButton
        variant="light"
        size="sm"
        :is-icon-only="true"
        aria-label="对话设置"
        :loading="busy"
      >
        <KunIcon name="lucide:ellipsis-vertical" class="size-5" />
      </KunButton>
    </template>
    <div class="flex flex-col">
      <button
        v-if="accepted"
        type="button"
        :class="itemClass"
        @click="setDialog({ pinned: !pinned })"
      >
        <KunIcon
          :name="pinned ? 'lucide:pin-off' : 'lucide:pin'"
          class="size-4"
        />
        {{ pinned ? '取消置顶' : '置顶对话' }}
      </button>
      <button
        type="button"
        :class="itemClass"
        @click="setDialog({ muted: !muted })"
      >
        <KunIcon
          :name="muted ? 'lucide:bell' : 'lucide:bell-off'"
          class="size-4"
        />
        {{ muted ? '取消静音' : '静音' }}
      </button>
      <button
        v-if="accepted"
        type="button"
        :class="itemClass"
        @click="setDialog({ archived: !archived })"
      >
        <KunIcon
          :name="archived ? 'lucide:archive-restore' : 'lucide:archive'"
          class="size-4"
        />
        {{ archived ? '移出归档' : '归档' }}
      </button>
      <button
        type="button"
        :class="itemClass"
        @click="setDialog({ marked_unread: true })"
      >
        <KunIcon name="lucide:mail" class="size-4" />
        标为未读
      </button>
      <button
        type="button"
        :class="cn(itemClass, 'text-danger')"
        @click="confirmClear = true"
      >
        <KunIcon name="lucide:eraser" class="size-4" />
        清空聊天记录
      </button>
    </div>
  </KunPopover>

  <KunModal
    v-model="confirmClear"
    role="alertdialog"
    title="清空聊天记录"
    description="只会从你这里清空，对方仍能看到这些消息。"
    inner-class-name="w-full max-w-sm"
  >
    <div class="flex justify-end gap-2">
      <KunButton variant="light" @click="confirmClear = false">取消</KunButton>
      <KunButton color="danger" :loading="busy" @click="clear">清空</KunButton>
    </div>
  </KunModal>
</template>
