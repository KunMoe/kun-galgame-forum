<script setup lang="ts">
definePageMeta({
  middleware: 'auth',
  key: 'chat',
  pageTransition: false
})

if (!useRuntimeConfig().public.agentEnabled) {
  throw createError({ statusCode: 404, statusMessage: 'Page Not Found' })
}

const route = useRoute()
const agent = useAgentStore()
useKunDisableSeo('AI 对话')

const openId = computed(() =>
  typeof route.params.id === 'string' && route.params.id
    ? route.params.id
    : null
)
const history = useAgentHistoryDrawer()
watch(
  () => route.fullPath,
  () => (history.value = false)
)

onMounted(() => void agent.init())
</script>

<template>
  <div class="h-[calc(100dvh-120px)] min-h-[420px]">
    <ClientOnly>
      <div
        v-if="agent.status === 'disabled'"
        class="text-default-500 flex h-full items-center justify-center text-sm"
      >
        本站的 AI 助手暂未开放。
      </div>
      <div
        v-else-if="agent.status === 'down'"
        class="text-default-500 flex h-full items-center justify-center text-sm"
      >
        AI 助手暂不可用，请稍后刷新页面再试。
      </div>
      <KunChatLayout
        v-else-if="agent.status === 'ready'"
        show-conversation
        sidebar-width="18rem"
      >
        <template #sidebar>
          <AgentSidebar :open-id="openId" />
        </template>
        <AgentPane :id="openId" />
      </KunChatLayout>
      <div v-else class="flex h-full items-center justify-center">
        <KunLoading />
      </div>

      <template v-if="agent.status === 'ready'">
        <KunDrawer
          v-model="history"
          placement="left"
          :responsive="false"
          :with-container="false"
          :is-show-close-button="false"
          size="sm"
          aria-label="对话历史"
        >
          <div class="flex items-center gap-2 px-3 pt-3">
            <NuxtLink
              to="/chat"
              class="flex min-w-0 items-center gap-2"
              :aria-label="`${kungal.titleShort} AI 对话`"
              @click="history = false"
            >
              <img src="/favicon.webp" alt="" class="size-7 rounded-lg" />
              <span class="text-foreground truncate font-bold">
                {{ kungal.titleShort }}
              </span>
            </NuxtLink>
            <KunButton
              variant="light"
              color="default"
              size="sm"
              is-icon-only
              aria-label="关闭"
              class-name="ml-auto shrink-0"
              @click="history = false"
            >
              <KunIcon name="lucide:x" class="size-4" />
            </KunButton>
          </div>
          <AgentSidebar :open-id="openId" class="min-h-0 flex-1" />
        </KunDrawer>
        <AgentMemoryModal />
      </template>
    </ClientOnly>
  </div>
</template>
