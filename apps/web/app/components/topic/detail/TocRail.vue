<script setup lang="ts">
import { TOPIC_TOC } from '~/composables/topic/useTopicTOC'
import {
  useCenterActiveToc,
  useTopicRailPhase
} from '~/composables/topic/useTopicRail'

const { headings, activeIds } = inject(TOPIC_TOC)!

const headingItems = computed(() =>
  headings.value.filter((h) => h.type === 'heading')
)
const replyItems = computed(() =>
  headings.value.filter((h) => h.type === 'reply')
)

const { isPastReplyTool, scrollToReplyTool } = useTopicRailPhase()

const railRef = ref<HTMLElement | null>(null)
useCenterActiveToc(railRef, activeIds)
</script>

<template>
  <aside
    ref="railRef"
    class="sticky top-20 hidden max-h-[calc(100dvh-6rem)] w-52 shrink-0 flex-col gap-4 overflow-hidden pt-3 xl:flex"
  >
    <div
      v-if="headingItems.length"
      class="scrollbar-hide max-h-[40vh] shrink-0 overflow-y-auto"
    >
      <h3 class="mb-2 text-sm font-medium">文章目录</h3>
      <TopicDetailTocSection :items="headingItems" :active-ids="activeIds" />
    </div>

    <div v-if="replyItems.length" class="flex min-h-0 flex-1 flex-col">
      <Transition
        enter-active-class="transition-all duration-300 ease-out"
        enter-from-class="-translate-y-1 opacity-0"
        leave-active-class="transition-all duration-200 ease-in"
        leave-to-class="-translate-y-1 opacity-0"
      >
        <div v-if="isPastReplyTool" class="shrink-0 pb-2">
          <KunButton
            class-name="w-full justify-center"
            size="sm"
            variant="flat"
            @click="scrollToReplyTool"
          >
            <KunIcon name="lucide:arrow-up-to-line" />
            回到回复顶部
          </KunButton>
        </div>
      </Transition>

      <div class="scrollbar-hide min-h-0 flex-1 overflow-y-auto">
        <h3 class="mb-2 text-sm font-medium">回复列表</h3>
        <TopicDetailTocSection :items="replyItems" :active-ids="activeIds" />
      </div>
    </div>
  </aside>
</template>
