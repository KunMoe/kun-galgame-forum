<script setup lang="ts">
const props = defineProps<{ conversation: AgentConversation | null }>()
const emit = defineEmits<{ close: [] }>()

const agent = useAgentStore()
const title = ref('')
const busy = ref(false)

const open = computed({
  get: () => !!props.conversation,
  set: (v) => {
    if (!v) emit('close')
  }
})

watch(
  () => props.conversation,
  (c) => (title.value = c?.title ?? '')
)

const save = async () => {
  const value = title.value.trim()
  if (!props.conversation || !value) return
  busy.value = true
  const ok = await agent.update(props.conversation.id, { title: value })
  busy.value = false
  if (ok) emit('close')
}
</script>

<template>
  <KunModal v-model="open" title="重命名对话" size="sm" is-show-close-button>
    <form class="flex flex-col gap-4 p-1" @submit.prevent="save">
      <KunInput v-model="title" label="标题" :maxlength="100" autofocus />
      <div class="flex justify-end gap-2">
        <KunButton variant="light" @click="emit('close')">取消</KunButton>
        <KunButton
          type="submit"
          color="primary"
          :loading="busy"
          :disabled="!title.trim()"
        >
          保存
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
