<script setup lang="ts">
defineProps<{
  conversation: AgentConversation
  selected: boolean
  running: boolean
}>()
const emit = defineEmits<{ action: [key: string] }>()
</script>

<template>
  <div
    :class="
      cn(
        'group rounded-kun-md relative',
        selected ? 'bg-default-100' : 'hover:bg-default-50'
      )
    "
  >
    <NuxtLink
      :to="`/chat/${conversation.id}`"
      :aria-current="selected ? 'page' : undefined"
      class="flex min-w-0 items-center gap-2 py-2 pr-9 pl-2.5 text-sm"
    >
      <span
        :class="
          cn(
            'min-w-0 flex-1 truncate',
            selected ? 'text-foreground' : 'text-default-700'
          )
        "
      >
        {{ conversation.title || '新对话' }}
      </span>
      <KunIcon
        v-if="running"
        name="lucide:loader-circle"
        aria-label="正在回复"
        class="text-default-400 size-4 shrink-0 animate-spin"
      />
    </NuxtLink>
    <div
      class="absolute top-1/2 right-1 -translate-y-1/2 transition-opacity focus-within:opacity-100 md:opacity-0 md:group-hover:opacity-100"
    >
      <AgentConversationMenu
        :starred="conversation.starred"
        @action="emit('action', $event)"
      />
    </div>
  </div>
</template>
