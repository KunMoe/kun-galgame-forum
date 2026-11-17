<script setup lang="ts">
const agent = useAgentStore()
const memory = useAgentMemory()
const popover = useTemplateRef<{ close: () => void }>('popover')
const openMemory = () => {
  popover.value?.close()
  memory.open()
}

const quota = computed(() => agent.quota)
const auto = computed(() => !!agent.persona?.web_search_auto)
const surcharge = computed(() =>
  quota.value
    ? `${quota.value.price_web_surcharge} NextMoe 萌萌点`
    : 'NextMoe 萌萌点'
)
const webDescription = computed(() =>
  auto.value
    ? `助手会自己判断要不要联网，打开后这一问先联网。真的搜索了才额外消耗 ${surcharge.value}`
    : `开启后每问额外消耗 ${surcharge.value}`
)
</script>

<template>
  <KunPopover
    ref="popover"
    position="top-start"
    inner-class="p-3 w-72"
    aria-label="对话设置"
  >
    <template #trigger>
      <KunButton
        variant="light"
        size="sm"
        is-icon-only
        aria-label="对话设置"
        :class="
          agent.webSearchAvailable && agent.webSearch ? 'text-primary' : ''
        "
        @click="agent.refreshQuota()"
      >
        <KunIcon name="lucide:settings-2" class="size-5" />
      </KunButton>
    </template>
    <div class="flex flex-col gap-3 text-sm">
      <KunSwitch
        v-if="agent.webSearchAvailable"
        v-model="agent.webSearch"
        :label="auto ? '优先联网' : '联网搜索'"
        :description="webDescription"
      />
      <KunDivider v-if="agent.webSearchAvailable" />
      <div v-if="quota" class="flex flex-col gap-1.5">
        <div class="flex items-center justify-between">
          <span class="text-default-500">今日免费额度</span>
          <span class="font-medium tabular-nums">
            {{ quota.free_left }} / {{ quota.free_per_day }}
          </span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-default-500">每问消耗</span>
          <span class="tabular-nums">
            {{ quota.price_assistant }} NextMoe 萌萌点
          </span>
        </div>
        <div
          v-if="quota.moemoepoint_balance !== null"
          class="flex items-center justify-between"
        >
          <span class="text-default-500">萌萌点余额</span>
          <span class="tabular-nums">{{ quota.moemoepoint_balance }}</span>
        </div>
        <p class="text-default-400 text-xs leading-relaxed">
          {{
            quota.paid_enabled
              ? '先用每日免费额度，用完后消耗 NextMoe 萌萌点。失败或停止的回复不计费。'
              : '每日免费额度用完后，今天就不能再问了。失败或停止的回复不计费。'
          }}
        </p>
      </div>
      <p v-else class="text-default-400">额度信息暂时读不到</p>
      <template v-if="agent.persona?.memory_available">
        <KunDivider />
        <KunButton
          variant="light"
          size="sm"
          full-width
          class-name="justify-start"
          @click="openMemory"
        >
          <KunIcon name="lucide:brain" class="size-4" />
          管理记忆
        </KunButton>
      </template>
    </div>
  </KunPopover>
</template>
