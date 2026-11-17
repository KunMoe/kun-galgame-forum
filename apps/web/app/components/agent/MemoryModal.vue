<script setup lang="ts">
const memory = useAgentMemory()
const s = memory.state

const draft = ref('')
const adding = ref(false)
const confirming = ref(false)
const clearing = ref(false)

const isOpen = computed({
  get: () => s.value.isOpen,
  set: (v) => {
    if (!v) memory.close()
  }
})
watch(isOpen, () => (confirming.value = false))

const memoryOn = computed({
  get: () => !!s.value.prefs?.memory,
  set: (v: boolean) => void memory.setPrefs({ memory: v })
})
const historyOn = computed({
  get: () => !!s.value.prefs?.history,
  set: (v: boolean) => void memory.setPrefs({ history: v })
})

const submit = async () => {
  const content = draft.value.trim()
  if (!content || adding.value) return
  adding.value = true
  if (await memory.add(content)) draft.value = ''
  adding.value = false
}

const clear = async () => {
  clearing.value = true
  await memory.clear()
  clearing.value = false
  confirming.value = false
}
</script>

<template>
  <KunModal
    v-model="isOpen"
    title="记忆"
    size="lg"
    scroll-behavior="inside"
    is-show-close-button
  >
    <div class="flex flex-col gap-4 p-1">
      <div v-if="!s.prefs && s.loading" class="flex justify-center py-10">
        <KunLoading />
      </div>
      <template v-else>
        <div class="flex flex-col gap-3">
          <KunSwitch
            v-model="memoryOn"
            label="参考记忆"
            description="助手会记住你的喜好和你让它记住的事，在之后的对话里参考。"
          />
          <KunSwitch
            v-if="s.prefs?.history_available"
            v-model="historyOn"
            label="参考过去的对话"
            description="助手能看到你在本站其他对话的标题，你提起以前聊过的内容时会去翻找。"
          />
          <p
            v-if="s.prefs && !s.prefs.memory"
            class="bg-default-100 rounded-kun-md text-default-600 p-3 text-sm leading-relaxed"
          >
            记忆已关闭。已有的记忆会保留，但助手不会读取，也不会再记住新的内容。
          </p>
        </div>

        <KunDivider />

        <div class="flex items-center gap-2">
          <h3 class="text-sm font-medium">已保存的记忆</h3>
          <span class="text-default-400 text-xs tabular-nums">
            {{ s.items.length }} / {{ s.max }}
          </span>
          <KunButton
            v-if="s.items.length && !confirming"
            variant="light"
            color="danger"
            size="sm"
            class="ml-auto"
            @click="confirming = true"
          >
            清空全部
          </KunButton>
        </div>

        <div
          v-if="confirming"
          class="border-danger-200 rounded-kun-md flex flex-wrap items-center gap-2 border p-3"
        >
          <p class="text-default-600 min-w-0 flex-1 text-sm">
            确定清空全部 {{ s.items.length }} 条记忆吗？清空后无法恢复。
          </p>
          <KunButton variant="light" size="sm" @click="confirming = false">
            取消
          </KunButton>
          <KunButton
            color="danger"
            size="sm"
            :loading="clearing"
            @click="clear"
          >
            清空
          </KunButton>
        </div>

        <form class="flex items-start gap-2" @submit.prevent="submit">
          <KunInput
            v-model="draft"
            placeholder="让助手记住一件事，例如：我喜欢纯爱和治愈系"
            aria-label="添加记忆"
            :maxlength="AGENT_MEMORY_MAX_CHARS"
            class="min-w-0 flex-1"
          />
          <KunButton
            type="submit"
            color="primary"
            :loading="adding"
            :disabled="!draft.trim()"
          >
            添加
          </KunButton>
        </form>

        <ul
          v-if="s.items.length"
          class="divide-default-100 flex flex-col divide-y"
        >
          <AgentMemoryItem v-for="m in s.items" :key="m.id" :memory="m" />
        </ul>
        <KunNull
          v-else
          description="还没有记忆。可以在对话里说「记住……」，也可以在上面添加。"
        />
      </template>
    </div>
  </KunModal>
</template>
