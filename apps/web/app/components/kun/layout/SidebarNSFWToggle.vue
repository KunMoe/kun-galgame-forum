<script setup lang="ts">
withDefaults(
  defineProps<{
    isCollapsed?: boolean
  }>(),
  { isCollapsed: false }
)

const { isSignedIn, allowsNsfw: isEnabled, setAnonymousNsfw } =
  useContentStance()
const { open } = useSettingPanel()

// Signed in there is no single opposite to flip to — the account holds three
// values and 模糊 / 显示 need age attestation first — so the banner hands the
// reader to the one control that can ask for all of it.
const toggle = () => {
  if (isSignedIn.value) {
    open('content')
    return
  }
  setAnonymousNsfw(!isEnabled.value)
  if (import.meta.client) {
    location.reload()
  }
}
</script>

<template>
  <template v-if="!isEnabled">
    <KunTooltip
      v-if="isCollapsed"
      text="点击开启 NSFW 模式"
      position="right"
      class-name="block w-full"
    >
      <button
        type="button"
        :class="
          cn(
            'flex w-full cursor-pointer items-center justify-center rounded-lg border p-2 transition-colors',
            'border-danger/40 bg-danger/10 text-danger-700 hover:bg-danger/20 dark:text-danger-300'
          )
        "
        aria-label="NSFW 模式已关闭"
        @click="toggle"
      >
        <KunIcon class="size-5" name="lucide:eye-off" />
      </button>
    </KunTooltip>

    <div
      v-else
      role="button"
      tabindex="0"
      :aria-pressed="false"
      :class="
        cn(
          'w-full cursor-pointer rounded-lg border px-3 py-2 text-left transition-colors outline-none focus-visible:ring-2',
          'border-danger/40 bg-danger/10 text-danger-700 hover:bg-danger/20 focus-visible:ring-danger/40 dark:text-danger-300'
        )
      "
      @click="toggle"
      @keydown.enter.prevent="toggle"
      @keydown.space.prevent="toggle"
    >
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2 text-sm font-semibold">
          <KunIcon class="size-4 shrink-0" name="lucide:eye-off" />
          <span>NSFW 模式已关闭</span>
        </div>
        <div class="shrink-0" @click.stop>
          <KunSwitch :model-value="false" @update:model-value="toggle" />
        </div>
      </div>
      <p class="mt-1 text-xs opacity-80">
        部分 R18 Galgame 不可见, 点击切换 NSFW 模式
      </p>
    </div>
  </template>
</template>
