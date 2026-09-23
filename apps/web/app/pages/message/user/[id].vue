<script setup lang="ts">
import type { Conversation } from '#shared/utils/api/schemas'

definePageMeta({
  middleware: 'auth'
})

const route = useRoute()

const userId = computed(() => String((route.params as { id: string }).id))

const { data: conversation, problem } = await useApi<Conversation>(
  () => `me-conversation:${userId.value}`,
  (api, { signal }) =>
    api.GET('/me/conversations/{user_id}', {
      params: { path: { user_id: userId.value } },
      signal
    })
)

const peerMissing = computed(() => problem.value?.status === 404)
const composerDisabled = computed(
  () => peerMissing.value || Boolean(problem.value)
)

useKunDisableSeo('私信')
</script>

<template>
  <div class="flex h-full min-w-0 flex-1 flex-col pl-3">
    <MessagePmHeader :conversation="conversation" :missing="peerMissing" />

    <MessagePmContainer
      :user-id="userId"
      :conversation="conversation"
      :composer-disabled="composerDisabled"
    />
  </div>
</template>
