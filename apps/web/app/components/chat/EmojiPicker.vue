<script setup lang="ts">
import type { KunEditorAdapters } from '@kungal/editor-core'
import type { KunStickerPick } from '@kungal/editor-vue'

const KunStickerPanel = defineAsyncComponent(() =>
  import('@kungal/editor-vue').then((m) => m.KunStickerPanel)
)

const emit = defineEmits<{ emoji: [text: string] }>()

const adapters: KunEditorAdapters = {}

const pick = (item: KunStickerPick) => {
  if (item.kind === 'emoji') {
    emit('emoji', item.text)
  }
}
</script>

<template>
  <KunPopover position="top-start" :auto-position="true">
    <template #trigger>
      <KunButton
        :is-icon-only="true"
        variant="light"
        rounded="full"
        class="size-10 shrink-0"
        aria-label="表情"
      >
        <KunIcon name="lucide:smile" class="text-xl" />
      </KunButton>
    </template>
    <KunStickerPanel :adapters="adapters" locale="zh-cn" @pick="pick" />
  </KunPopover>
</template>
