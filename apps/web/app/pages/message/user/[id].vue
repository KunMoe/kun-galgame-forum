<script setup lang="ts">
import type { Conversation } from '#shared/utils/api/schemas'

definePageMeta({
  middleware: [
    'auth',
    (to) => {
      if (useRuntimeConfig().public.chatEnabled) {
        return navigateTo(
          { path: '/messages', query: { to: String(to.params.id) } },
          { replace: true }
        )
      }
    }
  ]
})

const route = useRoute()

const userId = computed(() => String((route.params as { id: string }).id))

const {
  data: conversation,
  problem,
  pending
} = useKunApi<Conversation>(
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

    <div v-if="pending" class="flex-1 space-y-3 pt-4" aria-hidden="true">
      <KunSkeleton height="2.75rem" width="55%" rounded="lg" />
      <KunSkeleton
        height="2.75rem"
        width="40%"
        rounded="lg"
        class-name="ml-auto"
      />
      <KunSkeleton height="2.75rem" width="50%" rounded="lg" />
    </div>

    <MessagePmContainer
      v-else
      :user-id="userId"
      :conversation="conversation"
      :composer-disabled="composerDisabled"
    />
  </div>
</template>
