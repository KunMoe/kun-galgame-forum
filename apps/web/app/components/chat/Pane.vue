<script setup lang="ts">
import {
  formatKunChatMarkdown,
  parseKunChatMarkdown,
  type KunChatAttachment,
  type KunChatFormattedText,
  type KunChatMedia,
  type KunChatMessageAction,
  type KunChatReplyQuote
} from '@kungal/ui-vue'
import type { ChatMessage, ChatPhoto } from '#shared/types/chat'
import type { ApiResult } from '#shared/utils/api/problem'
import { chatProblemMessage } from '#shared/utils/api/chat'

const props = defineProps<{ id: string }>()

const EDIT_WINDOW = 48 * 3600 * 1000

type Attachment = KunChatAttachment & {
  photo?: Pick<ChatPhoto, 'image_hash' | 'width' | 'height' | 'thumbhash'>
}

const chat = useChatStore()
const act = useChatActions()

const conv = computed(() => chat.model.conversations[props.id] ?? null)
const win = computed(() => chat.model.windows[props.id] ?? null)
const peer = computed(() =>
  conv.value?.peer_id ? (chat.model.users[conv.value.peer_id] ?? null) : null
)
const typing = computed(() => chat.typing[props.id] ?? [])
const pinned = computed(
  () =>
    conv.value?.pinned_messages ??
    win.value?.items.filter((m) => m.pinned_at && m.seq > 0) ??
    []
)
const incomingRequest = computed(
  () =>
    !!conv.value &&
    !conv.value.me.accepted &&
    conv.value.last_message?.sender_id !== chat.model.me
)

const openedReadSeq = ref<number | null>(null)
const draft = ref('')
const replyTo = ref<ChatMessage | null>(null)
const quote = ref<KunChatReplyQuote | null>(null)
const editing = ref<ChatMessage | null>(null)
const attachments = ref<Attachment[]>([])
const list = ref<{
  scrollToSeq: (seq: number) => boolean
  scrollToBottom: () => void
} | null>(null)
const loadingOlder = ref(false)
const loadingNewer = ref(false)
const requestBusy = ref<'accept' | 'delete' | null>(null)
const deleting = ref<ChatMessage | null>(null)
const reporting = ref<ChatMessage | null>(null)

let savedDraft = ''

const persistDraft = async (id: string) => {
  if (draft.value === savedDraft) {
    return
  }
  savedDraft = draft.value
  await act.saveDraft(
    id,
    parseKunChatMarkdown(draft.value),
    replyTo.value?.seq ?? null
  )
}

watch(
  () => props.id,
  async (id, previous) => {
    if (previous) {
      void persistDraft(previous)
    }
    openedReadSeq.value = null
    replyTo.value = null
    quote.value = null
    editing.value = null
    attachments.value = []
    await chat.open(id)
    const c = chat.model.conversations[id]
    openedReadSeq.value = c?.me.last_read_seq ?? 0
    draft.value = c?.me.draft
      ? formatKunChatMarkdown(c.me.draft.text, c.me.draft.entities)
      : ''
    savedDraft = draft.value
  },
  { immediate: true }
)
onBeforeUnmount(() => void persistDraft(props.id))
onMounted(() => void chat.loadReactions())

const resolveMedia = (media: KunChatMedia) => imageTokenUrl(media.image_hash)

const notify = <T,>(r: ApiResult<T> | undefined) => {
  if (r && !r.ok) {
    useMessage(chatProblemMessage(r.problem), 'warn')
  }
  return r
}

const actions = (m: ChatMessage, own: boolean): KunChatMessageAction[] => {
  if (m.status === 'failed') {
    return ['retry', 'delete']
  }
  const out: KunChatMessageAction[] = ['reply', 'quote', 'copy']
  if (own && m.text && Date.now() - Date.parse(m.created_at) < EDIT_WINDOW) {
    out.push('edit')
  }
  if (conv.value?.me.accepted) {
    out.push(m.pinned_at ? 'unpin' : 'pin')
  }
  out.push('delete')
  if (!own) {
    out.push('report')
  }
  return out
}

const onAction = async (
  action: string,
  m: ChatMessage,
  detail: { quote?: KunChatReplyQuote }
) => {
  switch (action) {
    case 'reply':
    case 'quote':
      editing.value = null
      replyTo.value = m
      quote.value = detail.quote ?? null
      break
    case 'edit':
      editing.value = m
      break
    case 'pin':
    case 'unpin':
      notify(await act.pin(props.id, m.seq, action === 'pin'))
      break
    case 'retry':
      notify(await act.retry(m))
      break
    case 'delete':
      if (m.status === 'failed') {
        act.discard(m)
      } else {
        deleting.value = m
      }
      break
    case 'report':
      reporting.value = m
      break
  }
}

const send = async (body: KunChatFormattedText) => {
  const photos = attachments.value.filter((a) => a.photo)
  const reply = replyTo.value
  const q = quote.value
  attachments.value = []
  savedDraft = ''
  if (!photos.length) {
    notify(await act.send(props.id, body, { replyTo: reply, quote: q }))
    return
  }
  for (const [i, a] of photos.entries()) {
    const last = i === photos.length - 1
    notify(
      await act.send(props.id, last ? body : { text: '', entities: [] }, {
        replyTo: i === 0 ? reply : null,
        quote: i === 0 ? q : null,
        photo: a.photo
      })
    )
  }
}

const edit = async (body: KunChatFormattedText, target: ChatMessage) =>
  notify(await act.edit(target, body))

const editLast = () => {
  const mine = [...(win.value?.items ?? [])]
    .reverse()
    .find((m) => m.sender_id === chat.model.me && m.text && m.seq > 0)
  if (mine) {
    editing.value = mine
  }
}

const upload = async (key: string, file: File) => {
  const r = notify(await act.uploadPhoto(file))
  const i = attachments.value.findIndex((a) => a.key === key)
  if (i < 0 || !r) {
    return
  }
  attachments.value[i] = r.ok
    ? {
        ...attachments.value[i]!,
        error: false,
        photo: {
          image_hash: r.data.image_hash,
          width: r.data.width,
          height: r.data.height,
          thumbhash: r.data.thumbhash
        }
      }
    : { ...attachments.value[i]!, error: true }
}

const files = new Map<string, File>()

const attach = async (picked: File[]) => {
  for (const file of picked) {
    const key = crypto.randomUUID()
    files.set(key, file)
    attachments.value.push({
      key,
      url: URL.createObjectURL(file),
      name: file.name
    })
    await upload(key, file)
  }
}

const retryAttachment = (key: string) => {
  const file = files.get(key)
  if (file) {
    void upload(key, file)
  }
}

const removeAttachment = (key: string) => {
  const a = attachments.value.find((x) => x.key === key)
  if (a?.url) {
    URL.revokeObjectURL(a.url)
  }
  files.delete(key)
  attachments.value = attachments.value.filter((x) => x.key !== key)
}

const older = async () => {
  loadingOlder.value = true
  await chat.loadOlder(props.id)
  loadingOlder.value = false
}
const newer = async () => {
  loadingNewer.value = true
  await chat.loadNewer(props.id)
  loadingNewer.value = false
}
const jump = async (target: number) => {
  await chat.jump(props.id, target)
  await nextTick()
  list.value?.scrollToSeq(target)
}
const latest = async () => {
  await chat.latest(props.id)
  await nextTick()
  list.value?.scrollToBottom()
}

const accept = async () => {
  requestBusy.value = 'accept'
  notify(await act.accept(props.id))
  requestBusy.value = null
}
const declineRequest = async () => {
  requestBusy.value = 'delete'
  const r = notify(await act.clearHistory(props.id, true))
  requestBusy.value = null
  if (r?.ok) {
    await navigateTo('/messages')
  }
}

const openProfile = (uid: string) => navigateTo(`/user/${uid}`)
</script>

<template>
  <div v-if="conv" class="flex h-full min-h-0 flex-col">
    <KunChatHeader
      :user="peer"
      :kind="conv.kind"
      :title="conv.title ?? undefined"
      :typing="typing"
      :users="chat.users"
      @back="navigateTo('/messages')"
      @title-click="peer && !peer.deleted && openProfile(peer.id)"
    >
      <template #actions>
        <ChatDialogMenu :conversation="conv" />
      </template>
    </KunChatHeader>

    <KunChatPinnedBar
      v-if="pinned.length"
      :messages="pinned"
      :resolve-media-url="resolveMedia"
      :unpinnable="conv.me.accepted"
      @jump="(seq: number) => list?.scrollToSeq(seq) || jump(seq)"
      @unpin="(seq: number) => act.pin(conv!.id, seq, false).then(notify)"
    />

    <KunChatRequestBar
      v-if="incomingRequest"
      :user="peer"
      :actions="['accept', 'delete']"
      :loading="requestBusy"
      @accept="accept"
      @delete="declineRequest"
    />

    <KunChatMessageList
      v-if="win && openedReadSeq !== null"
      ref="list"
      :key="conv.id"
      class="bg-default-100 min-h-0 flex-1"
      :messages="win.items"
      :users="chat.users"
      :current-user-id="chat.model.me"
      :kind="conv.kind"
      :last-read-seq="openedReadSeq"
      :peer-read-seq="conv.peer_read_seq"
      :reaction-options="chat.reactions"
      :resolve-media-url="resolveMedia"
      :actions="actions"
      :has-older="win.hasOlder"
      :has-newer="win.hasNewer"
      :loading-older="loadingOlder"
      :loading-newer="loadingNewer"
      @action="onAction"
      @react="
        (m: ChatMessage, r: string | null) => act.react(m, r).then(notify)
      "
      @retry="(m: ChatMessage) => act.retry(m)?.then(notify)"
      @read="(seq: number) => act.read(conv!.id, seq)"
      @load-older="older"
      @load-newer="newer"
      @jump="jump"
      @latest="latest"
      @user-click="openProfile"
    >
      <template #footer>
        <KunChatTyping
          v-if="typing.length"
          :events="typing"
          :users="chat.users"
          :kind="conv.kind"
        />
      </template>
    </KunChatMessageList>
    <div v-else class="bg-default-100 flex flex-1 items-center justify-center">
      <KunLoading />
    </div>

    <KunChatComposer
      v-model="draft"
      v-model:reply-to="replyTo"
      v-model:quote="quote"
      v-model:editing="editing"
      :users="chat.users"
      :attachments="attachments"
      accept="image/*"
      :disabled="!!peer?.deleted"
      disabled-text="对方已注销，无法再发送消息"
      @send="send"
      @edit="edit"
      @edit-last="editLast"
      @attach="attach"
      @remove-attachment="removeAttachment"
      @retry-attachment="retryAttachment"
      @typing="act.sendTyping(conv.id)"
    />

    <ChatDeleteModal
      :message="deleting"
      :conversation-id="conv.id"
      @close="deleting = null"
    />
    <ChatReportModal :message="reporting" @close="reporting = null" />
  </div>
  <div v-else class="flex h-full items-center justify-center">
    <KunLoading />
  </div>
</template>
