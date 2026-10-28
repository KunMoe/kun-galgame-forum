import type { Ref } from 'vue'

const TOP_BAR_OFFSET = 88

export const useTopicRailPhase = () => {
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

  onMounted(() => {
    updatePhase()
    window.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll, { passive: true })
  })
  onBeforeUnmount(() => {
    window.removeEventListener('scroll', onScroll)
    window.removeEventListener('resize', onScroll)
  })

  return { isPastPoll, isPastReplyTool, scrollToReplyTool }
}

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

export const useCenterActiveToc = (
  railRef: Ref<HTMLElement | null>,
  activeIds: Ref<string[]>
) => {
  const centerActiveRun = () => {
    const rail = railRef.value
    const ids = activeIds.value
    if (!rail || !ids.length || !rail.offsetParent) {
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
}
