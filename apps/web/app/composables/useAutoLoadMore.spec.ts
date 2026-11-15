import { describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'

const visible = ref(false)
vi.mock('@vueuse/core', () => ({ useElementVisibility: () => visible }))

const { useAutoLoadMore } = await import('./useAutoLoadMore')

const setup = () => {
  const loading = ref(false)
  const hasMore = ref(true)
  const loadMore = vi.fn(() => {
    loading.value = true
  })
  const scope = effectScope()
  const { loadNext } = scope.run(() =>
    useAutoLoadMore(ref(null), { hasMore, loading, loadMore })
  )!
  const finish = async () => {
    loading.value = false
    await nextTick()
  }
  return { loading, hasMore, loadMore, loadNext, finish, scope }
}

describe('useAutoLoadMore', () => {
  it('loads while the sentinel stays near the viewport, four pages at most', async () => {
    visible.value = false
    const { loadMore, loadNext, finish, scope } = setup()
    visible.value = true
    await nextTick()
    expect(loadMore).toHaveBeenCalledTimes(1)
    for (let i = 0; i < 5; i++) {
      await finish()
    }
    expect(loadMore).toHaveBeenCalledTimes(4)

    loadNext()
    await nextTick()
    await finish()
    expect(loadMore).toHaveBeenCalledTimes(6)
    scope.stop()
  })

  it('stops at the end of the list', async () => {
    visible.value = false
    const { hasMore, loadMore, scope } = setup()
    hasMore.value = false
    visible.value = true
    await nextTick()
    expect(loadMore).not.toHaveBeenCalled()
    scope.stop()
  })
})
