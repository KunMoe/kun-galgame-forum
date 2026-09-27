<script setup lang="ts">
import { chatProblemMessage } from '#shared/utils/api/chat'

definePageMeta({
  middleware: 'auth',
  key: 'messages',
  pageTransition: false
})

if (!useRuntimeConfig().public.chatEnabled) {
  throw createError({ statusCode: 404, statusMessage: 'Page Not Found' })
}

const route = useRoute()
const chat = useChatStore()
const act = useChatActions()

useKunDisableSeo('私信')

const openId = computed(() =>
  typeof route.params.id === 'string' && route.params.id
    ? route.params.id
    : null
)
watch(openId, (id) => (chat.openId = id), { immediate: true })
onBeforeUnmount(() => (chat.openId = null))

const directTo = computed(() =>
  typeof route.query.to === 'string' && /^\d+$/.test(route.query.to)
    ? route.query.to
    : null
)
let opening = false
watch(
  () => [chat.status, directTo.value] as const,
  async ([status, to]) => {
    if (status !== 'ready' || !to || opening) {
      return
    }
    opening = true
    const r = await act.openDirect(to)
    opening = false
    if (r.ok) {
      await navigateTo(`/messages/${r.data.id}`, { replace: true })
      return
    }
    useMessage(chatProblemMessage(r.problem), 'warn')
    await navigateTo('/messages', { replace: true })
  },
  { immediate: true }
)

const relogin = () => startOAuthLogin({ returnTo: route.fullPath })
</script>

<template>
  <div class="h-[calc(100dvh-120px)] min-h-[420px]">
    <ClientOnly>
      <div
        v-if="chat.status === 'scope'"
        class="flex h-full flex-col items-center justify-center gap-3 text-center"
      >
        <KunIcon
          name="lucide:message-square-lock"
          class="text-primary size-10"
        />
        <p class="text-default-600 text-sm">
          这次登录还没有授权私信，重新登录一次即可使用。
        </p>
        <KunButton color="primary" @click="relogin">重新登录</KunButton>
      </div>

      <div
        v-else-if="chat.status === 'down'"
        class="text-default-500 flex h-full items-center justify-center text-sm"
      >
        私信服务暂不可用，请稍后刷新页面再试。
      </div>

      <div
        v-else-if="chat.status === 'ready'"
        class="bg-content1 border-default-200 h-full overflow-hidden rounded-lg border"
      >
        <KunChatLayout :show-conversation="!!openId">
          <template #sidebar>
            <ChatSidebar :open-id="openId" />
          </template>
          <ChatPane v-if="openId" :id="openId" />
          <template #empty>
            <div
              class="bg-default-100 text-default-500 flex h-full items-center justify-center text-sm"
            >
              选择一个对话开始聊天
            </div>
          </template>
        </KunChatLayout>
      </div>

      <div v-else class="flex h-full items-center justify-center">
        <KunLoading />
      </div>

      <template #fallback>
        <div class="flex h-full items-center justify-center">
          <KunLoading />
        </div>
      </template>
    </ClientOnly>
  </div>
</template>
