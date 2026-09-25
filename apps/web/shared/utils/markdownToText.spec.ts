import { describe, expect, it } from 'vitest'
import { markdownToText } from './markdownToText'

describe('markdownToText', () => {
  it('drops an image whose alt holds escaped brackets', () => {
    expect(
      markdownToText('看这个![鲲 Galgame 表情包 \\[1\\] - 56](/image/abc)好')
    ).toBe('看这个好')
  })

  it('keeps link text with escaped brackets, unescaped', () => {
    expect(markdownToText('[第 \\[2\\] 话](https://example.com)')).toBe(
      '第 [2] 话'
    )
  })

  it('still unescapes outside images and links', () => {
    expect(markdownToText('a \\* b \\\\ c')).toBe('a * b \\ c')
  })
})
