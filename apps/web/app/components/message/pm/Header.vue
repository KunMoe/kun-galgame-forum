<script setup lang="ts">
import type { Conversation } from '#shared/utils/api/schemas'
import { toKunUser, deletedUserName } from '~/utils/userRef'

const props = defineProps<{
  conversation?: Conversation
  missing: boolean
}>()

const title = computed(() => {
  if (props.missing) {
    return '用户不存在'
  }
  if (!props.conversation) {
    return ''
  }
  return props.conversation.peer.name ?? deletedUserName
})

const user = computed(() =>
  props.conversation ? toKunUser(props.conversation.peer) : undefined
)
</script>

<template>
  <header class="flex items-center gap-2">
    <KunButton size="lg" :is-icon-only="true" variant="light" href="/message">
      <KunIcon name="lucide:chevron-left" />
    </KunButton>

    <KunAvatar v-if="user" :user="user" />

    <h2 class="relative flex items-center gap-2">
      <span>{{ title }}</span>
    </h2>
  </header>
</template>
