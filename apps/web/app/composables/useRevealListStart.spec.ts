import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { revealListStart, useRevealListStart } from './useRevealListStart'

const heading = (top: number, scrollMarginTop = '80px') => {
  const el = document.createElement('h2')
  el.tabIndex = -1
  el.style.scrollMarginTop = scrollMarginTop
  document.body.append(el)
  el.getBoundingClientRect = () => ({ top }) as DOMRect
  const scroll = vi.fn()
  el.scrollIntoView = scroll
  const focus = vi.spyOn(el, 'focus')
  return { el, scroll, focus }
}

const reducedMotion = (reduce: boolean) =>
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => ({ matches: reduce }) as MediaQueryList)
  )

afterEach(() => {
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

describe('revealListStart', () => {
  it('leaves scroll and focus alone while the list start is in view', () => {
    reducedMotion(false)
    const { el, scroll, focus } = heading(80)
    revealListStart(el)
    expect(scroll).not.toHaveBeenCalled()
    expect(focus).not.toHaveBeenCalled()
  })

  it('focuses without scrolling, then scrolls the start under the offset', () => {
    reducedMotion(false)
    const { el, scroll, focus } = heading(-1200)
    revealListStart(el)
    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
    expect(scroll).toHaveBeenCalledWith({ block: 'start', behavior: 'smooth' })
    expect(document.activeElement).toBe(el)
  })

  it('scrolls the container but focuses the heading inside it', () => {
    reducedMotion(false)
    const container = document.createElement('div')
    container.innerHTML = '<p>filters</p><h1>Galgame</h1><h3>first row</h3>'
    document.body.append(container)
    container.getBoundingClientRect = () => ({ top: -1200 }) as DOMRect
    const scroll = vi.fn()
    container.scrollIntoView = scroll
    revealListStart(container)
    const title = container.querySelector('h1')!
    expect(document.activeElement).toBe(title)
    expect(title.tabIndex).toBe(-1)
    expect(scroll).toHaveBeenCalledOnce()
  })

  it('jumps instead of animating under reduced motion', () => {
    reducedMotion(true)
    const { el, scroll } = heading(-1200)
    revealListStart(el)
    expect(scroll).toHaveBeenCalledWith({ block: 'start', behavior: 'auto' })
  })
})

describe('useRevealListStart', () => {
  const flush = async () => {
    for (let i = 0; i < 5; i++) await nextTick()
  }

  it('waits for the page to land and the load to finish', async () => {
    reducedMotion(false)
    const { el, scroll } = heading(-1200)
    const page = ref(1)
    const loading = ref(false)
    const reveal = useRevealListStart(el, page, loading)

    const done = reveal(2)
    await flush()
    expect(scroll).not.toHaveBeenCalled()

    page.value = 2
    loading.value = true
    await flush()
    expect(scroll).not.toHaveBeenCalled()

    loading.value = false
    await done
    expect(scroll).toHaveBeenCalledOnce()
  })

  it('drops a request that a later page change overtook', async () => {
    reducedMotion(false)
    const { el, scroll } = heading(-1200)
    const page = ref(1)
    const loading = ref(false)
    const reveal = useRevealListStart(el, page, loading)

    const first = reveal(2)
    page.value = 2
    loading.value = true
    await flush()
    const second = reveal(3)
    page.value = 3
    await flush()
    loading.value = false
    await Promise.all([first, second])
    expect(scroll).toHaveBeenCalledOnce()
  })
})
