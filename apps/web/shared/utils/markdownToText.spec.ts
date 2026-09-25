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

  it('drops an image cut off by an excerpt limit', () => {
    for (const cut of [
      '好![鲲 Galgame 表情包 \\[5\\] - 41](/image/61ba40',
      '好![鲲 Galgame 表',
      '好![鲲]',
      '好![鲲 Galgame 表情包 \\[5\\] - 41](/image/61ba_320 "鲲 Gal'
    ]) {
      expect(markdownToText(cut)).toBe('好')
    }
  })

  it('keeps the text of a link whose url was cut off', () => {
    expect(markdownToText('看[这里](https://exa')).toBe('看这里')
  })

  it('leaves an open bracket that is not at the cut alone', () => {
    expect(markdownToText('a ![b\nc')).toBe('a ![b c')
  })

  it('still unescapes outside images and links', () => {
    expect(markdownToText('a \\* b \\\\ c')).toBe('a * b \\ c')
  })
})
