<script setup lang="ts">
import type { QuotePreviewState } from '~/composables/topic/useQuoteContent'
import { contentPlainText } from '~/utils/contentPlainText'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  preview: QuotePreviewState
}>()

const emit = defineEmits<{
  keep: []
  leave: []
}>()

const excerpt = computed(() => {
  const text = contentPlainText(props.preview.reply?.content ?? null).trim()
  const runes = [...text]
  return runes.length > 120 ? `${runes.slice(0, 120).join('')}…` : text
})

const author = computed(() =>
  props.preview.reply ? toKunUser(props.preview.reply.author) : null
)
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="preview.visible"
        class="kun-quote-preview border-default-200 bg-default-50/90 fixed z-50 max-h-48 w-72 origin-top-left overflow-y-auto rounded-xl border p-3 shadow-lg backdrop-blur-md"
        :style="{ top: `${preview.top}px`, left: `${preview.left}px` }"
        @mouseenter="emit('keep')"
        @mouseleave="emit('leave')"
      >
        <KunLoading v-if="preview.loading" description="加载中..." />

        <template v-else-if="preview.reply && author">
          <div class="mb-1.5 flex items-center gap-2">
            <KunAvatar :user="author" size="sm" />
            <span class="text-default-800 truncate text-sm font-medium">
              {{ author.name }}
            </span>
            <span class="text-default-400 ml-auto shrink-0 text-xs">
              #{{ preview.reply.floor }}
            </span>
          </div>
          <p class="text-default-600 text-sm break-words whitespace-pre-wrap">
            {{ excerpt || '(无文字内容)' }}
          </p>
        </template>

        <div v-else class="text-default-500 text-sm">该回复不存在或已删除</div>
      </div>
    </Transition>
  </Teleport>
</template>
