<script setup lang="ts">
const chat = useChatStore()

const unread = computed(() => chat.state?.unread_conversation_count ?? 0)
const requests = computed(() => chat.state?.request_count ?? 0)
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    class-name="hover:bg-primary/20 flex cursor-pointer flex-nowrap gap-3 rounded-lg p-2 transition-colors hover:opacity-80"
    to="/messages"
  >
    <div
      class="bg-default-100 flex h-12 w-12 shrink-0 items-center justify-center rounded-full"
    >
      <KunIcon name="lucide:message-circle" class="text-default-500 text-xl" />
    </div>
    <div class="flex w-full flex-col justify-center">
      <span class="font-bold">私信</span>
      <div class="flex items-center justify-between gap-2 text-sm">
        <span class="text-default-500 line-clamp-1">
          {{ requests ? `${requests} 个消息请求` : '和其他用户的一对一对话' }}
        </span>
        <ClientOnly>
          <KunChip
            v-if="unread && !chat.muted"
            class-name="whitespace-nowrap"
            color="primary"
          >
            {{ unread }}
          </KunChip>
        </ClientOnly>
      </div>
    </div>
  </KunLink>
</template>
