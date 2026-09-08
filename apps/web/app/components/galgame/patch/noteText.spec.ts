import { describe, it, expect } from 'vitest'
import { patchNoteText } from './noteText'

describe('patchNoteText', () => {
  it('drops the markdown syntax that used to render as source', () => {
    expect(patchNoteText('**注意** 请先阅读')).toBe('注意 请先阅读')
  })

  it('keeps an autolink URL that markdownToText strips as an HTML tag', () => {
    expect(patchNoteText('原始补丁链接:\n<https://www.moyu.moe/resource/7718>')).toBe(
      '原始补丁链接:\nhttps://www.moyu.moe/resource/7718'
    )
    expect(patchNoteText('感谢uncen404(<https://www.moyu.moe/user/27824>) 的补丁制作')).toBe(
      '感谢uncen404(https://www.moyu.moe/user/27824) 的补丁制作'
    )
  })

  it('keeps an autolink that markdown emphasis wraps', () => {
    expect(patchNoteText('直达：**<https://2dfmax.top/lists/4708>**')).toBe(
      '直达：https://2dfmax.top/lists/4708'
    )
  })

  it('folds an inline link so the destination survives as text', () => {
    expect(patchNoteText('[教程](https://example.com/a)')).toBe(
      '教程 (https://example.com/a)'
    )
  })

  it('keeps paragraph breaks and turns <br /> into a newline', () => {
    expect(patchNoteText('一行\n\n<br />\n二行')).toBe('一行\n\n二行')
  })

  it('leaves a bare URL alone', () => {
    expect(patchNoteText('解压码：https://pan.quark.cn/s/abc')).toBe(
      '解压码：https://pan.quark.cn/s/abc'
    )
  })
})
