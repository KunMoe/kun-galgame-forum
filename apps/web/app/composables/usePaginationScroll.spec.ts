// @vitest-environment nuxt
import { describe, expect, it, vi } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import { defineComponent, h, nextTick, ref, watch } from 'vue'
import { usePaginationScroll } from './usePaginationScroll'

const mountList = async () => {
  const page = ref(2)
  const loading = ref(false)
  const filter = ref('all')
  const rows = ref('Old results')
  const wrapper = await mountSuspended(
    defineComponent({
      setup() {
        watch(page, () => {
          loading.value = true
        })
        const { paginationPage, isPaginationPending } = usePaginationScroll(
          page,
          loading,
          filter
        )
        return () =>
          h('div', [
            h(
              'button',
              { onClick: () => (paginationPage.value = 1) },
              'Previous page'
            ),
            h(
              'div',
              { ref: 'paginationStart', 'data-results': '' },
              isPaginationPending.value ? `Loading: ${rows.value}` : rows.value
            )
          ])
      }
    })
  )
  const results = wrapper.get('[data-results]')
  const scroll = vi.fn(() => results.text())
  Object.defineProperty(results.element, 'scrollIntoView', { value: scroll })
  return { wrapper, results, scroll, page, loading, filter, rows }
}

describe('usePaginationScroll', () => {
  it('keeps the old results during pagination and scrolls after the new rows render', async () => {
    const { wrapper, results, scroll, loading, rows } = await mountList()
    await wrapper.get('button').trigger('click')
    expect(results.text()).toBe('Loading: Old results')
    expect(scroll).not.toHaveBeenCalled()

    rows.value = 'New results'
    loading.value = false
    await nextTick()
    expect(scroll).toHaveBeenCalledExactlyOnceWith({
      behavior: 'smooth',
      block: 'start'
    })
    expect(scroll).toHaveReturnedWith('New results')
    wrapper.unmount()
  })

  it('does not scroll for a programmatic page reset', async () => {
    const { wrapper, results, scroll, page, loading } = await mountList()
    page.value = 1
    await nextTick()
    expect(results.text()).toBe('Old results')
    loading.value = false
    await nextTick()
    expect(scroll).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('cancels scrolling when a filter changes during the same page request', async () => {
    const { wrapper, results, scroll, loading, filter } = await mountList()
    await wrapper.get('button').trigger('click')
    filter.value = 'filtered'
    await nextTick()
    expect(results.text()).toBe('Old results')
    loading.value = false
    await nextTick()
    expect(scroll).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
