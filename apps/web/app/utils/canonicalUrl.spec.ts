import { describe, expect, it } from 'vitest'
import { kunCanonicalUrl } from './canonicalUrl'

const origin = kungal.domain.main

describe('kunCanonicalUrl', () => {
  it('drops a trailing slash and every query but a later page', () => {
    expect(kunCanonicalUrl({ path: '/topic/1/', query: { reply: '3' } })).toBe(
      `${origin}/topic/1`
    )
    expect(kunCanonicalUrl({ path: '/galgame', query: { page: '2' } })).toBe(
      `${origin}/galgame?page=2`
    )
    expect(kunCanonicalUrl({ path: '/galgame', query: { page: '1' } })).toBe(
      `${origin}/galgame`
    )
    expect(kunCanonicalUrl({ path: '/', query: {} })).toBe(`${origin}/`)
    expect(
      kunCanonicalUrl({
        path: '/galgame/53',
        query: { tab: 'quiz', page: '2' }
      })
    ).toBe(`${origin}/galgame/53`)
  })
})
