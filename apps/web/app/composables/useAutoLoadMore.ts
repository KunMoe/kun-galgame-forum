import { useElementVisibility, type MaybeElementRef } from '@vueuse/core'
import { toValue, watch, type MaybeRefOrGetter, type WatchSource } from 'vue'

const MAX_AUTO_LOADS = 4

export const useAutoLoadMore = (
  sentinel: MaybeElementRef,
  options: {
    hasMore: MaybeRefOrGetter<boolean>
    loading: MaybeRefOrGetter<boolean>
    loadMore: () => unknown
    resetOn?: WatchSource
  }
) => {
  const visible = useElementVisibility(sentinel, { rootMargin: '200px' })
  const autoLoads = ref(0)

  watch(
    () =>
      visible.value && toValue(options.hasMore) && !toValue(options.loading),
    (ready) => {
      if (ready && autoLoads.value < MAX_AUTO_LOADS) {
        autoLoads.value++
        options.loadMore()
      }
    }
  )

  if (options.resetOn) {
    watch(options.resetOn, () => {
      autoLoads.value = 0
    })
  }

  const loadNext = () => {
    autoLoads.value = 0
    return options.loadMore()
  }

  return { loadNext }
}
