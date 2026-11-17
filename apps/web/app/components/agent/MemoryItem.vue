<script setup lang="ts">
import { formatDate } from '#shared/utils/time'
const props = defineProps<{ memory: AgentMemory }>()

const store = useAgentMemory()
const editing = ref(false)
const draft = ref('')
const saving = ref(false)
const removing = ref(false)

const tooLong = computed(
  () => [...draft.value.trim()].length > AGENT_MEMORY_MAX_CHARS
)

const edit = () => {
  draft.value = props.memory.content
  editing.value = true
}

const save = async () => {
  const content = draft.value.trim()
  if (!content || tooLong.value) return
  if (content === props.memory.content) {
    editing.value = false
    return
  }
  saving.value = true
  if (await store.update(props.memory.id, content)) editing.value = false
  saving.value = false
}

const remove = async () => {
  removing.value = true
  await store.remove(props.memory.id)
  removing.value = false
}
</script>

<template>
  <li class="py-3">
    <form v-if="editing" class="flex flex-col gap-2" @submit.prevent="save">
      <KunTextarea
        v-model="draft"
        aria-label="修改记忆"
        :rows="2"
        auto-grow
        resize="none"
        autofocus
        :error="tooLong ? `最多 ${AGENT_MEMORY_MAX_CHARS} 字` : ''"
      />
      <div class="flex justify-end gap-2">
        <KunButton variant="light" size="sm" @click="editing = false">
          取消
        </KunButton>
        <KunButton
          type="submit"
          color="primary"
          size="sm"
          :loading="saving"
          :disabled="!draft.trim() || tooLong"
        >
          保存
        </KunButton>
      </div>
    </form>

    <div v-else class="flex items-start gap-1">
      <div class="flex min-w-0 flex-1 flex-col gap-1.5">
        <p class="text-sm leading-relaxed break-words whitespace-pre-wrap">
          {{ memory.content }}
        </p>
        <div
          class="text-default-400 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs"
        >
          <KunChip
            size="sm"
            variant="flat"
            :color="memory.origin === 'explicit' ? 'primary' : 'default'"
          >
            {{ memory.origin === 'explicit' ? '你让我记住的' : '自动整理' }}
          </KunChip>
          <span>{{ formatDate(memory.updated_at, { isShowYear: true }) }}</span>
          <NuxtLink
            v-if="memory.conversation_id"
            :to="`/chat/${memory.conversation_id}`"
            class="hover:text-primary inline-flex items-center gap-1"
            @click="store.close()"
          >
            <KunIcon name="lucide:message-square" class="size-3.5" />
            来源对话
          </NuxtLink>
        </div>
      </div>
      <AgentIconAction icon="lucide:pencil" label="修改" @click="edit" />
      <AgentIconAction
        icon="lucide:trash-2"
        label="删除"
        :disabled="removing"
        @click="remove"
      />
    </div>
  </li>
</template>
