<script setup lang="ts">
import { usePreferredReducedMotion } from '@vueuse/core'
import type Typed from 'typed.js'

const props = defineProps<{ greeting: AgentGreeting }>()

const CURSOR_LINGER = 1600

const el = useTemplateRef<HTMLElement>('el')
const reduced = usePreferredReducedMotion()
const done = ref(false)
let typed: Typed | null = null

onMounted(async () => {
  if (reduced.value === 'reduce' || !el.value) {
    done.value = true
    return
  }
  const { default: TypedJS } = await import('typed.js')
  if (!el.value) return
  typed = new TypedJS(el.value, {
    strings: [props.greeting.text],
    contentType: 'text',
    typeSpeed: 45,
    startDelay: 200,
    showCursor: true,
    cursorChar: '▍',
    onComplete: (self) => {
      done.value = true
      setTimeout(() => self.cursor?.remove(), CURSOR_LINGER)
    }
  })
})

onBeforeUnmount(() => typed?.destroy())
</script>

<template>
  <div class="flex flex-col items-center gap-2">
    <p class="text-default-500 min-h-6 max-w-md">
      <span v-if="reduced === 'reduce'">{{ props.greeting.text }}</span>
      <span v-else ref="el" :aria-label="props.greeting.text" />
    </p>
    <KunButton
      v-if="props.greeting.conversationId && done"
      :href="`/chat/${props.greeting.conversationId}`"
      variant="light"
      color="primary"
      size="sm"
    >
      <KunIcon name="lucide:history" class="size-4" />
      继续这段对话
    </KunButton>
  </div>
</template>
