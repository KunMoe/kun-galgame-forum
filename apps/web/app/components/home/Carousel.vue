<script setup lang="ts">
import { useMouseInElement } from '@vueuse/core'

const { data: pinnedResponse } = await useApi(
  'docs:pinned-carousel',
  (api, { signal }) =>
    api.GET('/docs', {
      params: { query: { is_pinned: true, sort: 'views_desc', limit: 10 } },
      signal
    })
)

const pinnedPosts = computed(() => pinnedResponse.value?.items || [])

const currentSlide = ref(0)
const autoplayInterval = ref<NodeJS.Timeout>()
const carouselRef = ref<HTMLElement | null>(null)
const { isOutside } = useMouseInElement(carouselRef)

const AUTOPLAY_DELAY = 3000
const DRAG_THRESHOLD = 50

const isDragging = ref(false)
const dragStart = ref(0)
const dragOffset = ref(0)

const nextSlide = () => {
  if (!pinnedPosts.value.length) return
  currentSlide.value = (currentSlide.value + 1) % pinnedPosts.value.length
}

const prevSlide = () => {
  if (!pinnedPosts.value.length) return
  currentSlide.value =
    (currentSlide.value - 1 + pinnedPosts.value.length) %
    pinnedPosts.value.length
}

const startAutoplay = () => {
  stopAutoplay()
  autoplayInterval.value = setInterval(nextSlide, AUTOPLAY_DELAY)
}

const stopAutoplay = () => {
  if (autoplayInterval.value) {
    clearInterval(autoplayInterval.value)
    autoplayInterval.value = undefined
  }
}

const startDrag = (e: MouseEvent | TouchEvent) => {
  isDragging.value = true
  dragStart.value =
    e instanceof MouseEvent
      ? e.clientX
      : e.touches[0]
        ? e.touches[0].clientX
        : 0
  stopAutoplay()
}

const onDrag = (e: MouseEvent | TouchEvent) => {
  if (!isDragging.value) return
  const currentX =
    e instanceof MouseEvent
      ? e.clientX
      : e.touches[0]
        ? e.touches[0].clientX
        : 0
  const diff = currentX - dragStart.value
  const containerWidth = carouselRef.value?.offsetWidth || 0
  dragOffset.value = (diff / containerWidth) * 100
}

const endDrag = () => {
  if (!isDragging.value) return
  if (Math.abs(dragOffset.value) > DRAG_THRESHOLD) {
    if (dragOffset.value > 0) {
      prevSlide()
    } else {
      nextSlide()
    }
  }
  isDragging.value = false
  dragOffset.value = 0
  startAutoplay()
}

const manualSlide = (direction: 'next' | 'prev') => {
  if (direction === 'next') {
    nextSlide()
  } else {
    prevSlide()
  }
  startAutoplay()
}

watch(isOutside, (outside) => {
  if (outside) {
    startAutoplay()
  } else {
    stopAutoplay()
  }
})

onMounted(() => startAutoplay())
onBeforeUnmount(() => stopAutoplay())
</script>

<template>
  <KunCard :is-hoverable="false" class-name="overflow-hidden p-0">
    <div
      ref="carouselRef"
      class="group relative cursor-grab touch-pan-y overflow-hidden"
      @mousedown="startDrag"
      @mousemove="onDrag"
      @mouseup="endDrag"
      @mouseleave="endDrag"
      @dragstart.prevent
      @touchstart.passive="startDrag"
      @touchmove.passive="onDrag"
      @touchend="endDrag"
      @touchcancel="endDrag"
    >
      <template v-if="pinnedPosts.length">
        <div
          class="flex"
          :style="{
            transform: `translateX(${-currentSlide * 100 + dragOffset}%)`,
            transition: isDragging ? 'none' : 'transform 300ms ease-in-out'
          }"
        >
          <div
            v-for="(post, index) in pinnedPosts"
            :key="post.id"
            class="w-full flex-shrink-0"
          >
            <div class="relative h-40 w-full select-none">
              <KunImageNative
                :src="post.banner?.url || '/kungalgame.webp'"
                :alt="post.title"
                :loading="index === 0 ? 'eager' : 'lazy'"
                :fetchpriority="index === 0 ? 'high' : undefined"
                class-name="pointer-events-none h-full w-full object-cover select-none"
              />
              <div
                class="bg-content1/85 absolute right-0 bottom-0 left-0 m-2 space-y-1 rounded-lg p-2.5 backdrop-blur-sm"
              >
                <KunLink
                  underline="none"
                  class-name="text-foreground hover:text-primary text-base font-bold transition-colors"
                  :to="`/doc/${post.slug}`"
                >
                  <h2 class="line-clamp-1">{{ post.title }}</h2>
                </KunLink>
                <p class="text-default-700 line-clamp-2 text-xs">
                  {{ post.description }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <KunButton
          :is-icon-only="true"
          color="default"
          variant="flat"
          @click="manualSlide('prev')"
          class-name="absolute hidden group-hover:flex top-1/2 left-2 -translate-y-1/2 rounded-full"
        >
          <KunIcon name="lucide:chevron-left" />
        </KunButton>
        <KunButton
          :is-icon-only="true"
          color="default"
          variant="flat"
          @click="manualSlide('next')"
          class-name="absolute hidden group-hover:flex top-1/2 right-2 -translate-y-1/2 rounded-full"
        >
          <KunIcon name="lucide:chevron-right" />
        </KunButton>

        <div class="absolute top-2 left-1/2 flex -translate-x-1/2 space-x-1.5">
          <span
            v-for="(_, index) in pinnedPosts"
            :key="index"
            class="h-1.5 w-1.5 rounded-full transition-colors"
            :class="
              currentSlide === index ? 'bg-foreground/50' : 'bg-foreground/20'
            "
          />
        </div>
      </template>

      <KunNull v-else description="暂时没有置顶文档" />
    </div>
  </KunCard>
</template>
