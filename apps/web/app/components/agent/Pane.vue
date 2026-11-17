<script setup lang="ts">
import type { AgentFilePart } from '~/utils/agentUpload'

const props = defineProps<{ id: string | null }>()

const agent = useAgentStore()
const userStore = usePersistUserStore()
const route = useRoute()
const history = useAgentHistoryDrawer()
const missing = ref(false)
const loading = ref(false)
const greeting = ref<AgentGreeting | null>(null)

watch(
  () => props.id,
  async (id) => {
    missing.value = false
    if (!id) return
    loading.value = !agent.threads.has(id)
    const r = await agent.open(id)
    loading.value = false
    if (r && !r.ok) missing.value = true
  },
  { immediate: true }
)

onMounted(() => {
  greeting.value = pickAgentGreeting({
    greetings: agent.persona?.greetings ?? [],
    name: userStore.name,
    hour: new Date().getHours(),
    recent: agent.conversations
  })
  const ask = route.query.ask
  if (!props.id && typeof ask === 'string' && ask.trim()) {
    void navigateTo(
      { query: { ...route.query, ask: undefined } },
      { replace: true }
    )
    void submit(`帮我找找：${ask.trim().slice(0, AGENT_ASK_MAX)}`, [])
  }
})

const messages = computed(
  () => (props.id ? agent.threads.get(props.id) : []) ?? []
)
const busy = computed(() => !!props.id && agent.running.has(props.id))
const conversation = computed(() =>
  props.id ? agent.findConversation(props.id) : undefined
)
const title = computed(
  () => conversation.value?.title || (props.id ? '对话' : '新对话')
)
const suggestions = computed(() =>
  (agent.persona?.suggestions ?? []).filter((g) => g.prompts?.length)
)
const personaName = computed(() => agent.persona?.name || 'AI 助手')
const avatar = computed(() => agent.persona?.avatar || '/favicon.webp')

const scroller = useTemplateRef<HTMLElement>('scroller')
const stick = ref(true)
const onScroll = () => {
  const el = scroller.value
  if (el) stick.value = el.scrollHeight - el.scrollTop - el.clientHeight < 80
}
const toBottom = () =>
  nextTick(() => {
    const el = scroller.value
    if (el) el.scrollTop = el.scrollHeight
  })
watch(
  () => {
    const last = messages.value.at(-1)
    return [
      messages.value.length,
      last?.parts.length,
      last?.parts.reduce((n, p) => n + ('text' in p ? p.text.length : 0), 0)
    ]
  },
  () => {
    if (stick.value) void toBottom()
  }
)
watch(
  () => props.id,
  () => {
    stick.value = true
    void toBottom()
  }
)

const submit = async (
  text: string,
  files: AgentFilePart[],
  webSearch?: boolean
) => {
  stick.value = true
  if (props.id) return agent.send(props.id, text, files, { webSearch })
  const id = await agent.create()
  if (!id) return false
  const ok = await agent.send(id, text, files, { webSearch })
  if (ok) await navigateTo(`/chat/${id}`)
  return ok
}

const fileParts = (m: AgentMessage) =>
  m.parts.filter(
    (p): p is Extract<AgentPart, { type: 'file' }> => p.type === 'file'
  )

const edit = (m: AgentMessage, text: string) => {
  if (!props.id) return
  stick.value = true
  void agent.send(props.id, text, fileParts(m), {
    parentId: m.parent_id ?? '0'
  })
}

const regenerate = (userMessageId: string | null) => {
  if (!props.id || !userMessageId) return
  stick.value = true
  void agent.regenerate(props.id, userMessageId)
}

const renaming = ref<AgentConversation | null>(null)
const deleting = ref<AgentConversation | null>(null)
const act = (c: AgentConversation, key: string) => {
  if (key === 'star') void agent.update(c.id, { starred: !c.starred })
  else if (key === 'rename') renaming.value = c
  else if (key === 'delete') deleting.value = c
}

const switchTo = (messageId: string) => {
  if (props.id) void agent.switchTo(props.id, messageId)
}
</script>

<template>
  <div class="bg-background relative flex h-full min-h-0 flex-col">
    <div
      :class="
        cn(
          'absolute inset-x-0 top-0 z-10 flex h-12 items-center gap-1 px-2 @3xl/kun-chat:hidden',
          props.id && 'backdrop-blur-md'
        )
      "
    >
      <KunButton
        variant="light"
        is-icon-only
        aria-label="对话历史"
        class-name="shrink-0"
        @click="history = true"
      >
        <KunIcon name="lucide:panel-left" class="size-5" />
      </KunButton>
      <template v-if="props.id">
        <h1 class="min-w-0 flex-1 truncate text-sm font-semibold">
          {{ title }}
        </h1>
        <AgentConversationMenu
          v-if="conversation"
          :starred="conversation.starred"
          @action="act(conversation, $event)"
        />
      </template>
    </div>
    <div v-if="props.id" class="hidden @3xl/kun-chat:block">
      <KunChatHeader :title="title" :subtitle="personaName" :avatar="avatar">
        <template v-if="conversation" #actions>
          <AgentIconAction
            icon="lucide:star"
            :label="conversation.starred ? '取消收藏' : '收藏'"
            :active="conversation.starred"
            @click="
              agent.update(conversation.id, { starred: !conversation.starred })
            "
          />
        </template>
      </KunChatHeader>
    </div>
    <AgentRenameModal :conversation="renaming" @close="renaming = null" />
    <AgentDeleteModal
      :conversation="deleting"
      :open-id="props.id"
      @close="deleting = null"
    />

    <div
      v-if="missing"
      class="text-default-500 flex flex-1 items-center justify-center text-sm"
    >
      对话不存在或已被删除
    </div>

    <div v-else-if="loading" class="flex flex-1 items-center justify-center">
      <KunLoading />
    </div>

    <template v-else-if="!messages.length">
      <div
        class="mx-auto flex w-full max-w-3xl flex-1 flex-col justify-center gap-6 px-4 pb-8"
      >
        <div class="flex flex-col items-center gap-3 text-center">
          <img
            :src="avatar"
            :alt="personaName"
            class="rounded-kun-md size-14"
          />
          <h2 class="text-2xl font-semibold">{{ personaName }}</h2>
          <AgentGreeting v-if="greeting" :greeting="greeting" />
        </div>
        <AgentComposer :busy="busy" :submit="submit" />
        <AgentSuggestions
          v-if="suggestions.length"
          :groups="suggestions"
          :busy="busy"
          @pick="(p, web) => submit(p, [], web)"
        />
      </div>
    </template>

    <template v-else>
      <div
        ref="scroller"
        class="min-h-0 flex-1 overflow-y-auto"
        @scroll.passive="onScroll"
      >
        <div
          class="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 pt-14 pb-6 @3xl/kun-chat:pt-6"
        >
          <AgentMessage
            v-for="m in messages"
            :key="m.id"
            :message="m"
            :busy="busy"
            @edit="edit(m, $event)"
            @retry="regenerate(m.id)"
            @regenerate="regenerate(m.parent_id)"
            @switch="switchTo"
            @feedback="agent.feedback(m, $event)"
          />
        </div>
      </div>
      <div
        class="mx-auto w-full max-w-3xl px-3 pb-1.5 @3xl/kun-chat:px-4 @3xl/kun-chat:pb-4"
      >
        <AgentComposer
          :busy="busy"
          :submit="submit"
          @stop="props.id && agent.cancel(props.id)"
        />
        <p
          class="text-default-400 mt-1 text-center text-[11px] leading-tight @3xl/kun-chat:mt-1.5 @3xl/kun-chat:text-xs"
        >
          AI 回答可能出错，请以作品页信息为准 · 对话消耗 NextMoe 萌萌点
        </p>
      </div>
    </template>
  </div>
</template>
