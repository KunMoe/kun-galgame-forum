import { describe, expect, it } from 'vitest'
import { maskSpoilers } from './maskSpoilers'

describe('maskSpoilers', () => {
  it('masks each passage, including line breaks', () => {
    expect(maskSpoilers('前||秘密一||中||第一行\n第二行||后')).toBe(
      '前███中███后'
    )
  })

  it('preserves ordinary text and an unclosed marker', () => {
    expect(maskSpoilers('**普通** <文本> ||未闭合')).toBe(
      '**普通** <文本> ||未闭合'
    )
    expect(maskSpoilers('')).toBe('')
  })

  it('masks before a preview cuts off the closing marker', () => {
    expect(maskSpoilers(`前||${'秘密'.repeat(100)}||后`).slice(0, 100)).toBe(
      '前███后'
    )
  })
})
