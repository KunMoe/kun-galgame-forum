import { unrefElement, until, type MaybeElementRef } from '@vueuse/core'
import { nextTick, toValue, type MaybeRefOrGetter } from 'vue'

const HEADING = 'h1, h2, h3, h4, h5, h6'

export const revealListStart = (el: Element | null | undefined) => {
  if (!(el instanceof HTMLElement)) {
    return
  }
  const offset = Number.parseFloat(getComputedStyle(el).scrollMarginTop) || 0
  if (el.getBoundingClientRect().top >= offset) {
    return
  }
  const heading = el.matches(HEADING)
    ? el
    : (el.querySelector<HTMLElement>(HEADING) ?? el)
  if (!heading.hasAttribute('tabindex')) {
    heading.tabIndex = -1
  }
  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
  heading.focus({ preventScroll: true })
  el.scrollIntoView({ block: 'start', behavior: reduce ? 'auto' : 'smooth' })
}

export const useRevealListStart = (
  target: MaybeElementRef,
  page: MaybeRefOrGetter<number>,
  loading: MaybeRefOrGetter<boolean>
) => {
  let request = 0

  return async (nextPage: number) => {
    const current = ++request
    await until(() => toValue(page)).toBe(nextPage, { timeout: 10_000 })
    if (current !== request || toValue(page) !== nextPage) {
      return
    }
    await nextTick()
    await until(() => toValue(loading)).toBe(false)
    if (current !== request) {
      return
    }
    await nextTick()
    revealListStart(unrefElement(target))
  }
}
