<script setup lang="ts">
import type { Conversation, DirectMessage } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import { maxDecimalId } from '~/utils/decimalId'

const props = defineProps<{
  userId: string
  conversation?: Conversation
  composerDisabled: boolean
}>()

const historyScroll = useTemplateRef<{
  getViewport: () => HTMLElement | null
}>('historyScroll')
const getHistoryViewport = () => historyScroll.value?.getViewport() ?? null
const messageInput = ref('')
const messages = ref<DirectMessage[]>([])
const nextCursor = ref<string | undefined>(undefined)
const loadingMore = ref(false)
const isSending = ref(false)
const isUploadingImage = ref(false)
const pendingImages = ref<{ name: string; url: string }[]>([])
const messageTextarea = useTemplateRef<{
  insertAtCaret: (text: string) => void
}>('messageTextarea')
const fileInput = ref<HTMLInputElement | null>(null)
const isShowLoader = computed(() => Boolean(nextCursor.value))
const api = useApiClient()
const idempotency = useIdempotencyKey()
const conversationEpoch = useState('message-conversation-epoch', () => 0)
const didMarkRead = ref(false)

const scrollToBottom = () => {
  const viewport = getHistoryViewport()
  if (viewport) {
    viewport.scrollTo({
      top: viewport.scrollHeight,
      behavior: 'smooth'
    })
  }
}

const markLoadedRead = async (loaded: DirectMessage[]) => {
  if (didMarkRead.value || props.composerDisabled) {
    return
  }
  if (!props.conversation || props.conversation.unread_count <= 0) {
    return
  }
  const upTo = maxDecimalId(loaded.map((message) => message.id))
  if (!upTo) {
    return
  }
  didMarkRead.value = true
  const result = await settle(
    api.PUT('/me/conversations/{user_id}/read-marker', {
      params: { path: { user_id: props.userId } },
      body: { up_to_id: upTo }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  conversationEpoch.value += 1
}

const getMessageHistory = async (cursor?: string) => {
  const result = await settle(
    api.GET('/me/conversations/{user_id}/messages', {
      params: {
        path: { user_id: props.userId },
        query: {
          limit: 30,
          ...(cursor ? { cursor } : {})
        }
      }
    })
  )
  return result
}

const loadFirstPage = async () => {
  messages.value = []
  nextCursor.value = undefined
  didMarkRead.value = false
  if (props.composerDisabled) {
    return
  }
  const result = await getMessageHistory()
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  const page = [...result.data.items].reverse()
  messages.value = page
  nextCursor.value = result.data.next_cursor
  await markLoadedRead(page)
  nextTick(() => {
    scrollToBottom()
  })
}

const postMessage = async (content: string): Promise<boolean> => {
  if (isSending.value || props.composerDisabled) {
    return false
  }
  if (content.length > 1000) {
    useMessage(10402, 'warn')
    return false
  }

  isSending.value = true
  const body = { content_markdown: content }
  const result = await settle(
    api.POST('/me/conversations/{user_id}/messages', {
      params: {
        path: { user_id: props.userId },
        header: {
          'Idempotency-Key': idempotency.take(
            `/me/conversations/${props.userId}/messages`,
            body
          )
        }
      },
      body
    })
  )
  isSending.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  idempotency.clear()
  messages.value.push(result.data)
  conversationEpoch.value += 1
  nextTick(() => scrollToBottom())
  return true
}

const sendMessage = async () => {
  if (props.composerDisabled) {
    return
  }
  if (isUploadingImage.value) {
    useMessage('图片正在上传中, 请稍候', 'warn')
    return
  }

  const text = messageInput.value.trim()
  const imageMarkdown = pendingImages.value
    .map((img) => `![${img.name.replace(/[[\]()]/g, '')}](${img.url})`)
    .join(' ')
  const content = [text, imageMarkdown].filter(Boolean).join('\n')

  if (!content) {
    useMessage(10401, 'warn')
    return
  }

  if (await postMessage(content)) {
    messageInput.value = ''
    pendingImages.value = []
  }
}

const uploadImages = async (files: File[]) => {
  const images = files.filter((file) => file.type.startsWith('image/'))
  if (!images.length) {
    return
  }

  isUploadingImage.value = true
  try {
    for (const image of images) {
      const uploaded = await uploadImage(image, 'message', image.name)
      if (uploaded) {
        pendingImages.value.push({
          name: image.name,
          url: imageToken(uploaded.hash)
        })
      }
    }
  } finally {
    isUploadingImage.value = false
  }
}

const removePendingImage = (index: number) => {
  pendingImages.value.splice(index, 1)
}

const handlePaste = (event: ClipboardEvent) => {
  const files = Array.from(event.clipboardData?.files ?? [])
  if (!files.some((file) => file.type.startsWith('image/'))) {
    return
  }
  event.preventDefault()
  uploadImages(files)
}

const handleDrop = (event: DragEvent) => {
  uploadImages(Array.from(event.dataTransfer?.files ?? []))
}

const handleEnter = (event: KeyboardEvent) => {
  if (event.isComposing || event.shiftKey) {
    return
  }
  event.preventDefault()
  sendMessage()
}

const openFilePicker = () => {
  fileInput.value?.click()
}

const onFileChange = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files?.length) {
    uploadImages(Array.from(input.files))
  }
  input.value = ''
}

const onEmoji = (emoji: string) => {
  messageTextarea.value?.insertAtCaret(emoji)
}

const onSticker = (url: string) => {
  pendingImages.value.push({ name: 'sticker', url })
}

const handleRecallContextMenu = async (payload: {
  event: MouseEvent
  message: DirectMessage
}) => {
  const target = payload.message
  if (!target.viewer.is_mine || target.state === 'recalled') {
    return
  }

  const confirmed = await useComponentMessageStore().alert(
    '撤回这条消息?',
    '撤回后对方将看到 “XX 撤回了一条消息”, 内容不可恢复'
  )
  if (!confirmed) {
    return
  }

  const result = await settle(
    api.PATCH('/me/conversations/{user_id}/messages/{message_id}', {
      params: {
        path: { user_id: props.userId, message_id: target.id }
      },
      body: { state: 'recalled' }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }

  const idx = messages.value.findIndex((m) => m.id === target.id)
  if (idx !== -1) {
    messages.value[idx] = result.data
  }
  conversationEpoch.value += 1
  useMessage('撤回成功', 'success')
}

const handleLoadHistoryMessages = async () => {
  const viewport = getHistoryViewport()
  if (!viewport || !nextCursor.value || loadingMore.value) {
    return
  }

  const previousScrollHeight = viewport.scrollHeight
  const previousScrollTop = viewport.scrollTop
  loadingMore.value = true
  const result = await getMessageHistory(nextCursor.value)
  loadingMore.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }

  if (result.data.items.length > 0) {
    const older = [...result.data.items].reverse()
    const seen = new Set(messages.value.map((message) => message.id))
    messages.value = [
      ...older.filter((message) => !seen.has(message.id)),
      ...messages.value
    ]
    nextCursor.value = result.data.next_cursor

    nextTick(() => {
      const next = getHistoryViewport()
      if (next) {
        const newScrollHeight = next.scrollHeight
        next.scrollTo({
          top: previousScrollTop + (newScrollHeight - previousScrollHeight)
        })
      }
    })
  } else {
    nextCursor.value = undefined
  }
}

watch(
  () => [props.userId, props.composerDisabled] as const,
  () => {
    void loadFirstPage()
  }
)

onMounted(() => {
  void loadFirstPage()
})
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <KunOverlayScroll ref="historyScroll" :defer="false" class="min-h-0 flex-1">
      <div class="space-y-3 py-3">
        <div class="flex justify-center">
          <KunButton
            v-if="isShowLoader"
            @click="handleLoadHistoryMessages"
            size="sm"
            variant="light"
            :loading="loadingMore"
          >
            加载更多
          </KunButton>
        </div>

        <MessagePmItem
          v-for="message in messages"
          :key="message.id"
          :message="message"
          @context-menu="handleRecallContextMenu"
        />

        <div
          v-if="!messages.length && !composerDisabled"
          class="text-default-500 py-10 text-center"
        >
          暂无消息，发送一条消息开始聊天吧
        </div>
      </div>
    </KunOverlayScroll>

    <div
      class="shrink-0 border-t px-3 py-3"
      @paste="handlePaste"
      @drop.prevent="handleDrop"
      @dragover.prevent
    >
      <div
        v-if="pendingImages.length || isUploadingImage"
        class="mb-2 flex flex-wrap gap-2"
      >
        <div
          v-for="(img, index) in pendingImages"
          :key="img.url"
          class="border-default-200 relative h-16 w-16 overflow-hidden rounded-lg border"
        >
          <img
            :src="img.url"
            :alt="img.name"
            class="h-full w-full object-cover"
          />
          <button
            type="button"
            @click="removePendingImage(index)"
            class="bg-background/70 text-default-600 hover:text-danger absolute top-0.5 right-0.5 flex h-5 w-5 items-center justify-center rounded-full text-xs leading-none"
            aria-label="移除图片"
          >
            ✕
          </button>
        </div>
        <div
          v-if="isUploadingImage"
          class="border-default-200 text-default-500 flex h-16 w-16 items-center justify-center rounded-lg border border-dashed text-xs"
        >
          上传中...
        </div>
      </div>

      <div class="flex flex-col gap-1.5 sm:flex-row sm:items-end sm:gap-1">
        <div class="flex gap-1">
          <KunPopover position="top-start" :auto-position="true">
            <template #trigger>
              <KunButton
                :is-icon-only="true"
                variant="light"
                size="lg"
                :disabled="composerDisabled"
                aria-label="表情和贴纸"
              >
                <KunIcon name="lucide:smile" />
              </KunButton>
            </template>
            <MessagePmEmojiStickerPicker @emoji="onEmoji" @sticker="onSticker" />
          </KunPopover>

          <KunButton
            :is-icon-only="true"
            variant="light"
            size="lg"
            :disabled="composerDisabled"
            @click="openFilePicker"
            aria-label="上传图片"
          >
            <KunIcon name="lucide:image" />
          </KunButton>
          <input
            ref="fileInput"
            type="file"
            accept="image/*"
            multiple
            class="hidden"
            :disabled="composerDisabled"
            @change="onFileChange"
          />
        </div>

        <div class="flex flex-1 items-end gap-1">
          <KunTextarea
            ref="messageTextarea"
            v-model="messageInput"
            placeholder="输入消息... (可粘贴或拖拽图片, Enter 发送, Shift+Enter 换行)"
            class="flex-1"
            :auto-grow="true"
            :rows="1"
            max-height="160px"
            :disabled="composerDisabled"
            @keydown.enter="handleEnter"
          />
          <KunButton
            @click="sendMessage"
            :loading="isSending"
            :disabled="composerDisabled"
            size="lg"
          >
            发送
          </KunButton>
        </div>
      </div>
    </div>
  </div>
</template>
