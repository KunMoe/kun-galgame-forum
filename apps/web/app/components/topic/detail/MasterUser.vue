<script setup lang="ts">
import { useTopicTOC, TOPIC_TOC_SOURCE } from '~/composables/topic/useTopicTOC'

const props = defineProps<{
  user: {
    id: number
    name: string
    avatar: string
    moemoepoint: number
  }
}>()

const user = computed(() => props.user)

const source = inject(TOPIC_TOC_SOURCE)!
const { headings, activeIds } = useTopicTOC(source)

const headingItems = computed(() =>
  headings.value.filter((h) => h.type === 'heading')
)
const replyItems = computed(() =>
  headings.value.filter((h) => h.type === 'reply')
)

const TOP_BAR_OFFSET = 88
const isPastPoll = ref(false)
const isPastReplyTool = ref(false)
let ticking = false

const updatePhase = () => {
  ticking = false
  const anchor = document.getElementById('comments-anchor')
  if (!anchor) {
    isPastPoll.value = false
    isPastReplyTool.value = false
    return
  }
  const rect = anchor.getBoundingClientRect()
  isPastPoll.value = rect.top <= TOP_BAR_OFFSET
  isPastReplyTool.value = rect.bottom <= TOP_BAR_OFFSET
}

const onScroll = () => {
  if (ticking) {
    return
  }
  ticking = true
  requestAnimationFrame(updatePhase)
}

const scrollToReplyTool = () => {
  document
    .getElementById('comments-anchor')
    ?.scrollIntoView({ behavior: 'smooth' })
}

const railRef = ref<HTMLElement | null>(null)

const getScrollParent = (el: HTMLElement): HTMLElement | null => {
  let p = el.parentElement
  while (p) {
    const oy = getComputedStyle(p).overflowY
    if (oy === 'auto' || oy === 'scroll') {
      return p
    }
    p = p.parentElement
  }
  return null
}

const centerActiveRun = () => {
  const rail = railRef.value
  const ids = activeIds.value
  if (!rail || !ids.length) {
    return
  }
  let firstEl: HTMLElement | null = null
  let top = Infinity
  let bottom = -Infinity
  rail.querySelectorAll<HTMLElement>('a[data-toc-id]').forEach((el) => {
    if (!ids.includes(el.dataset.tocId ?? '')) {
      return
    }
    firstEl ??= el
    const r = el.getBoundingClientRect()
    top = Math.min(top, r.top)
    bottom = Math.max(bottom, r.bottom)
  })
  if (!firstEl) {
    return
  }
  const container = getScrollParent(firstEl)
  if (!container) {
    return
  }
  const base = container.getBoundingClientRect().top - container.scrollTop
  const target = (top + bottom) / 2 - base - container.clientHeight / 2
  const maxScroll = container.scrollHeight - container.clientHeight
  container.scrollTo({
    top: Math.min(maxScroll, Math.max(0, target)),
    behavior: 'smooth'
  })
}

watch(activeIds, () => centerActiveRun(), { flush: 'post' })

onMounted(() => {
  updatePhase()
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll, { passive: true })
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', onScroll)
})
</script>

<template>
  <div
    ref="railRef"
    class="sticky top-20 hidden max-h-[calc(100dvh-6rem)] w-52 shrink-0 flex-col gap-3 overflow-hidden lg:flex"
  >
    <div
      class="grid shrink-0 transition-all duration-500 ease-out"
      :class="
        isPastPoll ? 'grid-rows-[0fr] opacity-0' : 'grid-rows-[1fr] opacity-100'
      "
    >
      <div class="flex min-h-0 flex-col items-center gap-3 overflow-hidden">
        <KunAvatar
          class-name="aspect-square h-auto w-full hover:scale-100"
          size="original"
          image-class-name="size-full rounded-lg"
          :user="user"
        />

        <KunLink
          underline="hover"
          :aria-label="user.name"
          :to="`/user/${user.id}`"
        >
          {{ user.name }}
        </KunLink>

        <p class="text-secondary flex items-center gap-1">
          <KunIcon class="text-inherit" name="lucide:lollipop" />
          {{ user.moemoepoint }}
        </p>

        <div
          v-if="headingItems.length"
          class="scrollbar-hide max-h-[30vh] w-full overflow-y-auto"
        >
          <h3 class="mb-2 text-sm font-medium">文章目录</h3>
          <TopicDetailTocSection
            :items="headingItems"
            :active-ids="activeIds"
          />
        </div>
      </div>
    </div>

    <div v-if="replyItems.length" class="flex min-h-0 flex-1 flex-col">
      <Transition
        enter-active-class="transition-all duration-300 ease-out"
        enter-from-class="-translate-y-1 opacity-0"
        leave-active-class="transition-all duration-200 ease-in"
        leave-to-class="-translate-y-1 opacity-0"
      >
        <KunLink
          v-if="isPastPoll"
          :to="`/user/${user.id}`"
          :aria-label="user.name"
          underline="none"
          color="default"
          class-name="hover:bg-default-100 mb-2 flex shrink-0 items-center gap-2.5 rounded-lg p-1.5 transition-colors"
        >
          <KunAvatar
            :user="user"
            :is-navigation="false"
            size="original"
            class-name="size-10 shrink-0"
            image-class-name="size-10 rounded-full"
          />
          <div class="flex min-w-0 flex-col">
            <span class="truncate text-base font-medium">{{ user.name }}</span>
            <span class="text-secondary flex items-center gap-1 text-sm">
              <KunIcon class="text-inherit" name="lucide:lollipop" />
              {{ user.moemoepoint }}
            </span>
          </div>
        </KunLink>
      </Transition>

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
  </div>
</template>
