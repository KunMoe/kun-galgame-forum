<script setup lang="ts">
import type { KunTagBlockMode } from '~/store/modules/tagBlocks'

const props = defineProps<{
  tagId: number
  name: string
  descendantCount: number
}>()

const { modeOf, block, unblock } = useTagBlocks()
const mode = computed(() => modeOf(props.tagId))
const busy = ref(false)
const popover = ref<{ close: () => void } | null>(null)

const run = async (action: () => Promise<boolean>, done: string) => {
  busy.value = true
  const ok = await action()
  busy.value = false
  popover.value?.close()
  if (ok) {
    useMessage(done, 'success')
  }
}

const onBlock = (next: KunTagBlockMode) => {
  const label = `「${props.name}」`
  const done =
    next === 'tree' && props.descendantCount
      ? `已屏蔽${label}及其 ${props.descendantCount} 个子标签`
      : `已屏蔽${label}`
  run(() => block(props.tagId, next), done)
}

const onUnblock = () =>
  run(() => unblock(props.tagId), `已取消屏蔽「${props.name}」`)
</script>

<template>
  <div class="flex flex-wrap items-center gap-2">
    <template v-if="mode">
      <KunChip size="sm" variant="flat" color="danger">
        <KunIcon name="lucide:ban" class="size-3.5" />
        {{
          mode === 'tree' && descendantCount ? '已屏蔽 · 含子标签' : '已屏蔽'
        }}
      </KunChip>
      <KunButton
        v-if="descendantCount"
        size="sm"
        variant="light"
        :disabled="busy"
        @click="onBlock(mode === 'tree' ? 'exact' : 'tree')"
      >
        {{ mode === 'tree' ? '改为只屏蔽此标签' : '连同子标签一起屏蔽' }}
      </KunButton>
      <KunButton size="sm" variant="flat" :disabled="busy" @click="onUnblock">
        取消屏蔽
      </KunButton>
    </template>

    <KunButton
      v-else-if="!descendantCount"
      size="sm"
      variant="flat"
      color="danger"
      :disabled="busy"
      @click="onBlock('tree')"
    >
      <KunIcon name="lucide:ban" />
      屏蔽此标签
    </KunButton>

    <KunPopover
      v-else
      ref="popover"
      position="bottom-start"
      aria-label="屏蔽此标签"
      inner-class="w-80 max-w-[calc(100vw-2rem)] space-y-3 p-3"
    >
      <template #trigger>
        <KunButton size="sm" variant="flat" color="danger">
          <KunIcon name="lucide:ban" />
          屏蔽此标签
          <KunIcon name="lucide:chevron-down" class="size-3.5" />
        </KunButton>
      </template>

      <GalgameTagBlockCoverage :tag-id="tagId" :count="descendantCount" />

      <div class="flex flex-col gap-2">
        <KunButton
          size="sm"
          color="danger"
          full-width
          :disabled="busy"
          @click="onBlock('tree')"
        >
          连同子标签一起屏蔽
        </KunButton>
        <KunButton
          size="sm"
          variant="flat"
          full-width
          :disabled="busy"
          @click="onBlock('exact')"
        >
          只屏蔽「{{ name }}」
        </KunButton>
      </div>
    </KunPopover>
  </div>
</template>
