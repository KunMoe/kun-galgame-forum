<script setup lang="ts">
import type { Conversation } from '#shared/utils/api/schemas'
import { contentPlainText } from '~/utils/contentPlainText'
import { toKunUser, deletedUserName } from '~/utils/userRef'

const props = defineProps<{
  conversation: Conversation
}>()

const peerName = computed(
  () => props.conversation.peer.name ?? deletedUserName
)
const peerUser = computed(() => toKunUser(props.conversation.peer))

const preview = computed(() => {
  const last = props.conversation.last_message
  if (!last) {
    return ''
  }
  if (last.state === 'recalled') {
    const senderName = last.sender.name ?? deletedUserName
    return `${senderName}撤回了一条消息`
  }
  return contentPlainText(last.content)
})
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    class-name="hover:bg-primary/20 flex cursor-pointer flex-nowrap gap-3 rounded-lg p-2 transition-colors hover:opacity-80"
    :to="`/message/user/${conversation.peer.id}`"
  >
    <KunAvatar :user="peerUser" size="xl" :is-navigation="false" />
    <div class="justify-space flex w-full flex-col">
      <div class="flex items-center justify-between">
        <span class="font-bold">{{ peerName }}</span>
        <span
          class="text-default-500 text-sm"
          v-if="conversation.last_message_at"
        >
          <KunTime :time="conversation.last_message_at" />
        </span>
      </div>

      <div class="flex items-center justify-between text-sm">
        <span class="line-clamp-1 break-all">
          {{ preview }}
        </span>
        <KunChip
          class-name="whitespace-nowrap"
          color="primary"
          v-if="conversation.unread_count"
        >
          {{ conversation.unread_count }}
        </KunChip>
        <KunChip
          class-name="whitespace-nowrap"
          color="default"
          v-else
        >
          {{ conversation.message_count }}
        </KunChip>
      </div>
    </div>
  </KunLink>
</template>
