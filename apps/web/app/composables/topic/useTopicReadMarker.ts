import { useThrottleFn } from '@vueuse/core'
import { useTopicSubscription } from './useTopicSubscription'

const SEND_DELAY_MS = 1500

// Featured replies (pinned, best answer) sit above the list and can carry the
// newest floor, so only the list itself counts as read.
const highestSeenFloor = () => {
  const list = document.querySelector('[data-reply-list]')
  if (!list) {
    return 0
  }
  const bottom = window.innerHeight
  let highest = 0
  list.querySelectorAll<HTMLElement>('.kun-reply').forEach((el) => {
    if (el.getBoundingClientRect().top < bottom) {
      highest = Math.max(highest, parseInt(el.id, 10) || 0)
    }
  })
  return highest
}

export const useTopicReadMarker = (topicId: string) => {
  const { subscription, load, markRead } = useTopicSubscription(topicId)
  const isWatching = computed(
    () => subscription.value?.notification_level === 'watching'
  )
  let pending = 0
  let timer: ReturnType<typeof setTimeout> | undefined

  const flush = () => {
    clearTimeout(timer)
    timer = undefined
    const floor = pending
    pending = 0
    if (floor > (subscription.value?.last_read_floor ?? 0)) {
      void markRead(floor)
    }
  }

  const check = useThrottleFn(() => {
    if (!isWatching.value) {
      return
    }
    const floor = highestSeenFloor()
    if (floor <= Math.max(pending, subscription.value?.last_read_floor ?? 0)) {
      return
    }
    pending = floor
    clearTimeout(timer)
    timer = setTimeout(flush, SEND_DELAY_MS)
  }, 300)

  onMounted(async () => {
    window.addEventListener('scroll', check, { passive: true })
    await load()
    void check()
  })

  watch(isWatching, (watching) => {
    if (watching) {
      void check()
    }
  })

  onBeforeUnmount(() => {
    window.removeEventListener('scroll', check)
    if (pending) {
      flush()
    }
  })
}
