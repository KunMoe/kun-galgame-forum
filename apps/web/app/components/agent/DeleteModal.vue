<script setup lang="ts">
const props = defineProps<{
  conversation: AgentConversation | null
  openId: string | null
}>()
const emit = defineEmits<{ close: [] }>()

const agent = useAgentStore()
const busy = ref(false)

const open = computed({
  get: () => !!props.conversation,
  set: (v) => {
    if (!v) emit('close')
  }
})

const confirm = async () => {
  const c = props.conversation
  if (!c) return
  busy.value = true
  const ok = await agent.remove(c.id)
  busy.value = false
  emit('close')
  if (ok && c.id === props.openId) await navigateTo('/chat')
}
</script>

<template>
  <KunModal v-model="open" title="删除对话" size="sm" is-show-close-button>
    <div class="flex flex-col gap-4 p-1">
      <p class="text-default-600 text-sm">
        确定删除「{{
          conversation?.title || '新对话'
        }}」吗？对话里的消息会一起删除，无法恢复。
      </p>
      <div class="flex justify-end gap-2">
        <KunButton variant="light" @click="emit('close')">取消</KunButton>
        <KunButton color="danger" :loading="busy" @click="confirm">
          删除
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
