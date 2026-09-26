<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import ContentDocument from '~/components/content/Document.vue'
import type { DirectMessage } from '#shared/utils/api/schemas'
import { toKunUser, deletedUserName } from '~/utils/userRef'

const props = defineProps<{
  message: DirectMessage
}>()

const emit = defineEmits<{
  (
    event: 'context-menu',
    payload: { event: MouseEvent; message: DirectMessage }
  ): void
}>()

const isSent = computed(() => props.message.viewer.is_mine)
const isMobile = useMediaQuery('(max-width: 640px)')
const canRecall = computed(
  () => props.message.viewer.is_mine && props.message.state === 'sent'
)
const senderName = computed(
  () => props.message.sender.name ?? deletedUserName
)
const senderUser = computed(() => toKunUser(props.message.sender))
const recallText = computed(() => `${senderName.value}撤回了一条消息`)
const recallCursorClass = computed(() => {
  if (!canRecall.value) {
    return ''
  }

  return isMobile.value ? 'cursor-pointer' : 'cursor-context-menu'
})

const handleContextMenu = (event: MouseEvent) => {
  if (!canRecall.value || isMobile.value) {
    return
  }
  event.stopPropagation()
  emit('context-menu', { event, message: props.message })
}

const handleClick = (event: MouseEvent) => {
  if (!canRecall.value || !isMobile.value) {
    return
  }
  event.stopPropagation()
  emit('context-menu', { event, message: props.message })
}
</script>

<template>
  <div
    class="flex w-full"
    :class="[
      message.state === 'recalled'
        ? 'items-center justify-center py-2'
        : 'items-end gap-2',
      message.state === 'recalled' ? '' : isSent ? 'flex-row-reverse' : 'flex-row'
    ]"
  >
    <template v-if="message.state === 'recalled'">
      <span
        class="bg-default-100 text-default-500 rounded-full px-3 py-1 text-xs sm:text-sm"
      >
        {{ recallText }}
      </span>
    </template>

    <template v-else>
      <KunAvatar :user="senderUser" class="mb-auto" />

      <div
        class="relative max-w-[75%] rounded-lg border p-3 transition-colors"
        :class="[
          isSent
            ? 'bg-primary/20 border-primary/20'
            : 'bg-background border-default-300',
          recallCursorClass
        ]"
        @contextmenu.prevent="handleContextMenu"
        @click="handleClick"
      >
        <div class="flex items-end">
          <span
            class="text-sm font-medium"
            :class="isSent ? 'text-primary' : 'text-secondary'"
          >
            {{ senderName }}
          </span>
        </div>

        <div class="mt-1 text-sm leading-relaxed">
          <ContentDocument :document="message.content" compact />
          <div class="text-default-500 mt-0.5 text-right text-xs">
            <KunTime :time="message.created_at" />
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
