<script setup lang="ts">
const props = defineProps<{ message: AgentMessage; busy: boolean }>()
const emit = defineEmits<{
  edit: [text: string]
  retry: []
  regenerate: []
  switch: [id: string]
  feedback: [rating: AgentFeedback]
}>()

const layout = computed(() => layoutReply(props.message.parts))
const streaming = computed(() => props.message.status === 'streaming')
const answered = computed(() =>
  layout.value.answer.some((n) => n.type === 'text' && n.text)
)
const answer = computed(() =>
  layout.value.answer.map((n) =>
    n.type === 'text'
      ? { ...n, html: renderAgentMarkdown(n.text) }
      : { ...n, html: '' }
  )
)
const copyText = computed(
  () =>
    layout.value.answer
      .flatMap((n) => (n.type === 'text' && n.text ? [n.text] : []))
      .join('\n\n') || messageText(props.message)
)
const waiting = computed(
  () => streaming.value && !answered.value && !layout.value.steps.length
)

const hostOf = (url: string) => {
  try {
    return new URL(url).hostname
  } catch {
    return url
  }
}
</script>

<template>
  <AgentUserMessage
    v-if="message.role === 'user'"
    :message="message"
    :busy="busy"
    @edit="emit('edit', $event)"
    @retry="emit('retry')"
    @switch="emit('switch', $event)"
  />

  <div v-else class="flex flex-col gap-3">
    <AgentSteps
      v-if="layout.steps.length"
      :steps="layout.steps"
      :cards="layout.cards"
      :running="streaming && !answered"
    />

    <template v-for="n in answer" :key="n.id">
      <KunContent
        v-if="n.type === 'text' && n.html"
        :content="n.html"
        compact
      />
      <KunAccordion
        v-else-if="n.type === 'reasoning' && n.text.trim()"
        variant="light"
      >
        <KunAccordionItem
          :value="n.id"
          :title="n.done ? '思考过程' : '思考中…'"
          icon="lucide:brain"
        >
          <p
            class="text-default-500 text-sm leading-relaxed whitespace-pre-wrap"
          >
            {{ n.text.trim() }}
          </p>
        </KunAccordionItem>
      </KunAccordion>
    </template>

    <AgentEntityCards v-if="layout.cards.length" :cards="layout.cards" />
    <AgentBrowseLinks v-if="layout.browses.length" :browses="layout.browses" />

    <p
      v-if="waiting"
      class="text-default-400 flex items-center gap-1.5 text-sm"
      aria-live="polite"
    >
      <KunIcon name="lucide:loader-circle" class="size-4 animate-spin" />
      正在回复…
    </p>

    <div v-if="layout.sources.length" class="flex flex-col gap-1">
      <p class="text-default-500 text-xs font-medium">来源</p>
      <ol class="flex flex-col gap-0.5 text-sm">
        <li v-for="(s, i) in layout.sources" :key="s.url">
          <a
            :href="s.url"
            target="_blank"
            rel="noopener noreferrer nofollow"
            class="text-primary hover:underline"
          >
            [{{ i + 1 }}] {{ s.title || hostOf(s.url) }}
          </a>
        </li>
      </ol>
    </div>

    <p
      v-if="message.status === 'failed'"
      class="text-danger flex items-center gap-1.5 text-sm"
    >
      <KunIcon name="lucide:circle-alert" class="size-4" />
      回复失败{{ message.error ? `：${message.error}` : '，请重试' }}
    </p>
    <p
      v-else-if="message.status === 'canceled'"
      class="text-default-400 text-sm"
    >
      已停止生成
    </p>

    <div v-if="!streaming" class="-mt-1 -ml-2 flex items-center gap-0.5">
      <AgentVersionSwitcher
        :ids="message.sibling_ids"
        :current="message.id"
        :disabled="busy"
        @switch="emit('switch', $event)"
      />
      <AgentIconAction
        v-if="copyText"
        icon="lucide:copy"
        label="复制"
        @click="useKunCopy(copyText)"
      />
      <AgentIconAction
        icon="lucide:refresh-cw"
        label="重新生成"
        :disabled="busy || !message.parent_id"
        @click="emit('regenerate')"
      />
      <template v-if="message.status === 'complete'">
        <AgentIconAction
          icon="lucide:thumbs-up"
          label="有帮助"
          :active="message.feedback === 'up'"
          @click="emit('feedback', 'up')"
        />
        <AgentIconAction
          icon="lucide:thumbs-down"
          label="没帮助"
          :active="message.feedback === 'down'"
          @click="emit('feedback', 'down')"
        />
      </template>
    </div>
  </div>
</template>
