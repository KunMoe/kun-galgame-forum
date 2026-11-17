<script setup lang="ts">
const props = defineProps<{ openId: string | null }>()

const agent = useAgentStore()
const memory = useAgentMemory()
const renaming = ref<AgentConversation | null>(null)
const deleting = ref<AgentConversation | null>(null)

const search = useAgentConversationSearch()
const personaName = computed(() => agent.persona?.name || 'AI 助手')

const sections = computed(() =>
  search.active.value
    ? [{ key: 'search', label: '', items: search.items.value }]
    : [
        { key: 'starred', label: '收藏', items: agent.starred },
        { key: 'recent', label: '最近', items: agent.conversations }
      ].filter((s) => s.items.length)
)
const cursor = computed(() =>
  search.active.value ? search.cursor.value : agent.cursor
)
const loadingMore = computed(() =>
  search.active.value ? search.loadingMore.value : agent.loadingMore
)
const loadMore = () =>
  search.active.value ? search.loadMore() : agent.loadMore()

const act = (c: AgentConversation, key: string) => {
  if (key === 'star') void agent.update(c.id, { starred: !c.starred })
  else if (key === 'rename') renaming.value = c
  else if (key === 'delete') deleting.value = c
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex items-center gap-2 px-3 pt-3 pb-2">
      <h1 class="text-lg font-semibold">对话</h1>
      <KunButton
        href="/chat?new"
        variant="flat"
        color="primary"
        size="sm"
        class="ml-auto"
      >
        <KunIcon name="lucide:square-pen" class="size-4" />
        新对话
      </KunButton>
    </div>

    <div class="px-3 pb-2">
      <KunInput
        v-model="search.term.value"
        placeholder="搜索对话"
        aria-label="搜索对话"
        size="sm"
        :maxlength="100"
        is-clearable
      >
        <template #prefix>
          <KunIcon name="lucide:search" class="text-default-400 size-4" />
        </template>
      </KunInput>
    </div>

    <div class="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto p-1.5">
      <section v-for="sec in sections" :key="sec.key" class="flex flex-col">
        <h2
          v-if="sec.label && agent.starred.length"
          class="text-default-400 px-2 pt-2 pb-1 text-xs font-medium"
        >
          {{ sec.label }}
        </h2>
        <AgentConversationRow
          v-for="c in sec.items"
          :key="c.id"
          :conversation="c"
          :selected="c.id === props.openId"
          :running="agent.running.has(c.id) || !!c.active_run_id"
          @action="act(c, $event)"
        />
      </section>

      <template v-if="search.active.value">
        <KunLoading
          v-if="search.pending.value && !search.items.value.length"
          class="py-6"
        />
        <KunNull
          v-else-if="!search.items.value.length"
          description="没有找到相关对话"
        />
      </template>
      <KunNull
        v-else-if="!sections.length"
        :description="`还没有对话，和${personaName}聊聊吧`"
      />

      <KunButton
        v-if="cursor"
        variant="light"
        size="sm"
        class="mx-auto my-2 shrink-0"
        :loading="loadingMore"
        @click="loadMore()"
      >
        加载更多
      </KunButton>
    </div>

    <div
      v-if="agent.persona?.memory_available"
      class="border-default-100 shrink-0 border-t p-1.5"
    >
      <KunButton
        variant="light"
        size="sm"
        full-width
        class-name="justify-start text-default-700"
        @click="memory.open()"
      >
        <KunIcon name="lucide:brain" class="size-4" />
        记忆
      </KunButton>
    </div>

    <AgentRenameModal :conversation="renaming" @close="renaming = null" />
    <AgentDeleteModal
      :conversation="deleting"
      :open-id="props.openId"
      @close="deleting = null"
    />
  </div>
</template>
