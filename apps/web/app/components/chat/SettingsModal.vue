<script setup lang="ts">
import type { ChatAllow, ChatSettings } from '#shared/types/chat'
import { chatProblemMessage } from '#shared/utils/api/chat'

const open = defineModel<boolean>({ default: false })

const api = useChatApi()
const settings = ref<ChatSettings | null>(null)
const busy = ref(false)

const incomingOptions: {
  label: string
  value: ChatAllow
  description: string
}[] = [
  {
    label: '所有人',
    value: 'all',
    description: '任何人的私信都直接进入你的对话列表'
  },
  {
    label: '我关注的人',
    value: 'following',
    description: '其他人的私信先进入「消息请求」'
  },
  {
    label: '没有人',
    value: 'none',
    description: '所有新对话都先进入「消息请求」'
  }
]

watch(open, async (v) => {
  if (!v) {
    return
  }
  const r = await api.get<ChatSettings>('/settings')
  if (r.ok) {
    settings.value = r.data
  } else {
    useMessage(chatProblemMessage(r.problem), 'warn')
    open.value = false
  }
})

const save = async (patch: Partial<Omit<ChatSettings, 'object'>>) => {
  busy.value = true
  const r = await api.patch<ChatSettings>('/settings', patch)
  busy.value = false
  if (r.ok) {
    settings.value = r.data
  } else {
    useMessage(chatProblemMessage(r.problem), 'warn')
  }
}
</script>

<template>
  <KunModal v-model="open" title="私信设置" inner-class-name="w-full max-w-md">
    <KunLoading v-if="!settings" />
    <div v-else class="space-y-5">
      <div class="space-y-2">
        <p class="font-medium">谁的私信可以直接进入对话列表</p>
        <KunRadioGroup
          :model-value="settings.allow_incoming"
          :options="incomingOptions"
          :disabled="busy"
          variant="card"
          aria-label="谁的私信可以直接进入对话列表"
          @update:model-value="(v: ChatAllow) => save({ allow_incoming: v })"
        />
      </div>
      <KunSwitch
        :model-value="settings.accept_requests"
        :disabled="busy"
        color="primary"
        label="接收消息请求（关闭后，上面范围之外的人无法给你发私信）"
        @update:model-value="(v: boolean) => save({ accept_requests: v })"
      />
      <p class="text-default-500 text-xs">
        私信设置跟着你的 NextMoe 账号走，在鲲 Galgame、一起萌和 App 中通用。
      </p>
    </div>
  </KunModal>
</template>
