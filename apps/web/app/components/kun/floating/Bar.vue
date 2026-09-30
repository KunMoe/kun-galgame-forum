<script setup lang="ts">
const SCROLL_THRESHOLD = 100
const DIRECTION_THRESHOLD = 5
const AUTO_HIDE_TIMEOUT = 2000

const MASCOT = {
  top: '/image/scroll-top.webp',
  bottom: '/image/scroll-bottom.webp'
} as const

const isVisible = ref(false)
const isIdle = ref(false)
const isHovered = ref(false)
const scrollingDown = ref(true)
const isAtBottom = ref(false)

let lastScrollY = 0
let idleTimer: ReturnType<typeof setTimeout> | undefined

const direction = computed(() =>
  isAtBottom.value || !scrollingDown.value ? 'top' : 'bottom'
)
const label = computed(() =>
  direction.value === 'top' ? '回到顶部' : '滚动到底部'
)

const scrollToEdge = () => {
  window.scrollTo({
    top: direction.value === 'top' ? 0 : document.documentElement.scrollHeight,
    behavior: 'smooth'
  })
}

const onScroll = () => {
  const { scrollY, innerHeight } = window
  const { scrollHeight } = document.documentElement
  isVisible.value = scrollY > SCROLL_THRESHOLD
  isAtBottom.value = scrollY + innerHeight >= scrollHeight - 2

  const delta = scrollY - lastScrollY
  if (Math.abs(delta) > DIRECTION_THRESHOLD) {
    if (!isAtBottom.value) {
      scrollingDown.value = delta > 0
    }
    lastScrollY = scrollY
  }

  isIdle.value = false
  clearTimeout(idleTimer)
  idleTimer = setTimeout(() => {
    isIdle.value = true
  }, AUTO_HIDE_TIMEOUT)
}

const onPointerEnter = (event: PointerEvent) => {
  if (event.pointerType === 'mouse') {
    isHovered.value = true
  }
}

watch(
  isVisible,
  () => {
    if (window.matchMedia('(hover: hover)').matches) {
      for (const src of Object.values(MASCOT)) {
        new Image().src = src
      }
    }
  },
  { once: true }
)

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll()
})

onUnmounted(() => {
  window.removeEventListener('scroll', onScroll)
  clearTimeout(idleTimer)
})
</script>

<template>
  <div
    v-if="isVisible"
    :class="
      cn('fixed right-3 bottom-[60px] z-100', isIdle && !isHovered && 'hidden')
    "
    @pointerenter="onPointerEnter"
    @pointerleave="isHovered = false"
  >
    <Transition name="kun-mascot">
      <img
        v-if="isHovered"
        :key="direction"
        :src="MASCOT[direction]"
        :class="
          cn(
            'pointer-events-none absolute right-0 bottom-full size-48 max-w-none origin-bottom select-none',
            `kun-mascot-${direction}`
          )
        "
        alt=""
        aria-hidden="true"
      />
    </Transition>

    <KunTooltip :text="label" position="left">
      <button
        type="button"
        :aria-label="label"
        class="bg-primary-100 text-primary-600 hover:bg-primary-200 focus-visible:ring-primary/50 flex size-12 cursor-pointer items-center justify-center rounded-2xl shadow-md transition-colors focus-visible:ring-2 focus-visible:outline-none pointer-fine:size-16"
        @click="scrollToEdge"
      >
        <KunIcon
          class="size-6 text-inherit pointer-fine:size-7"
          :name="direction === 'top' ? 'lucide:arrow-up' : 'lucide:arrow-down'"
        />
      </button>
    </KunTooltip>
  </div>
</template>

<style scoped>
.kun-mascot-enter-active.kun-mascot-top {
  animation: kun-mascot-jump 520ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

.kun-mascot-enter-active.kun-mascot-bottom {
  animation: kun-mascot-settle 560ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

.kun-mascot-leave-active {
  transition:
    opacity 150ms ease-in,
    transform 150ms ease-in;
}

.kun-mascot-leave-to {
  opacity: 0;
  transform: translateY(12px) scale(0.9);
}

@keyframes kun-mascot-jump {
  0% {
    opacity: 0;
    transform: translateY(40px) scale(0.6);
  }
  55% {
    opacity: 1;
    transform: translateY(-18px) scale(1.04);
  }
  100% {
    opacity: 1;
    transform: none;
  }
}

@keyframes kun-mascot-settle {
  0% {
    opacity: 0;
    transform: translateY(-36px) scale(0.9);
  }
  55% {
    opacity: 1;
    transform: translateY(6px) scale(1.04, 0.94);
  }
  100% {
    opacity: 1;
    transform: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .kun-mascot-enter-active.kun-mascot-top,
  .kun-mascot-enter-active.kun-mascot-bottom {
    animation: none;
    transition: opacity 150ms ease-out;
  }

  .kun-mascot-enter-from {
    opacity: 0;
  }

  .kun-mascot-leave-to {
    transform: none;
  }
}
</style>
