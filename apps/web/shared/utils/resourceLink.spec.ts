import { describe, expect, it } from 'vitest'
import {
  applyResourceLinkBlur,
  applyResourceLinkPaste,
  detectProviderKeyFromURL,
  isCleanLinkDump,
  parseResourceLinks,
  splitResourceLinkText
} from './resourceLink'

describe('parseResourceLinks', () => {
  it('passes a clean URL through and names the provider', () => {
    const parsed = parseResourceLinks('https://pan.baidu.com/s/1abcdefghijk')
    expect(parsed.links).toEqual(['https://pan.baidu.com/s/1abcdefghijk'])
    expect(parsed.code).toBe('')
    expect(parsed.providers).toEqual(['百度网盘'])
  })

  it('reads pwd from the query string', () => {
    const parsed = parseResourceLinks(
      'https://pan.baidu.com/s/1abcdefghijk?pwd=ab12'
    )
    expect(parsed.links).toEqual([
      'https://pan.baidu.com/s/1abcdefghijk?pwd=ab12'
    ])
    expect(parsed.code).toBe('ab12')
  })

  it('strips share-dump prose and fills the extraction code', () => {
    const parsed = parseResourceLinks(`通过网盘分享的文件：某某某
链接: https://pan.baidu.com/s/1abcdefghijk 提取码: ab12
复制这段内容后打开百度网盘手机App，操作更方便哦`)
    expect(parsed.links).toEqual(['https://pan.baidu.com/s/1abcdefghijk'])
    expect(parsed.code).toBe('ab12')
    expect(parsed.providers).toEqual(['百度网盘'])
  })

  it('stops the URL before adjacent CJK so 提取码 is not swallowed', () => {
    const parsed = parseResourceLinks(
      'https://pan.quark.cn/s/abc123提取码：qwer'
    )
    expect(parsed.links).toEqual(['https://pan.quark.cn/s/abc123'])
    expect(parsed.code).toBe('qwer')
    expect(parsed.providers).toEqual(['夸克网盘'])
  })

  it('collects several URLs from a mixed dump', () => {
    const parsed =
      parseResourceLinks(`百度：https://pan.baidu.com/s/aaa 提取码：ab12
夸克：https://pan.quark.cn/s/bbb
阿里：https://www.alipan.com/s/ccc`)
    expect(parsed.links).toEqual([
      'https://pan.baidu.com/s/aaa',
      'https://pan.quark.cn/s/bbb',
      'https://www.alipan.com/s/ccc'
    ])
    expect(parsed.code).toBe('ab12')
    expect(parsed.providers).toEqual(['百度网盘', '夸克网盘', '阿里云盘'])
  })

  it('keeps distinct extraction codes', () => {
    const parsed = parseResourceLinks(
      'https://pan.baidu.com/s/aaa?pwd=ab12 https://pan.baidu.com/s/bbb 提取码: cd34'
    )
    expect(parsed.code).toBe('ab12, cd34')
  })

  it('reads an unzip password separately from the extraction code', () => {
    const parsed = parseResourceLinks(
      'https://pan.baidu.com/s/aaa 提取码: ab12 解压码: kun-galgame'
    )
    expect(parsed.code).toBe('ab12')
    expect(parsed.password).toBe('kun-galgame')
  })

  it('accepts magnet URIs', () => {
    const magnet =
      'magnet:?xt=urn:btih:0ee024cfced3166973d5672471b223b7735604de&dn=x'
    const parsed = parseResourceLinks(`磁力：${magnet}`)
    expect(parsed.links).toEqual([magnet])
    expect(parsed.providers).toEqual(['其他 (自建网盘等不限速)'])
  })

  it('strips trailing punctuation copied with the URL', () => {
    const parsed = parseResourceLinks('https://pan.baidu.com/s/aaa。')
    expect(parsed.links).toEqual(['https://pan.baidu.com/s/aaa'])
  })

  it('returns nothing when there is no URL', () => {
    expect(parseResourceLinks('提取码: ab12')).toEqual({
      links: [],
      code: 'ab12',
      password: '',
      providers: []
    })
  })
})

describe('detectProviderKeyFromURL', () => {
  it('maps the site netdisk kinds from their share URLs', () => {
    expect(detectProviderKeyFromURL('https://pan.baidu.com/s/xxx')).toBe(
      'baidu'
    )
    expect(detectProviderKeyFromURL('https://www.alipan.com/s/xxx')).toBe(
      'aliyun'
    )
    expect(detectProviderKeyFromURL('https://pan.quark.cn/s/xxx')).toBe('quark')
    expect(detectProviderKeyFromURL('https://www.123pan.com/s/xxx')).toBe(
      'pan123'
    )
    expect(detectProviderKeyFromURL('https://cloud.189.cn/t/xxx')).toBe(
      'tianyiyun'
    )
    expect(detectProviderKeyFromURL('https://yun.139.com/xxx')).toBe('caiyun')
    expect(detectProviderKeyFromURL('https://pan.xunlei.com/s/xxx')).toBe(
      'xunlei'
    )
    expect(detectProviderKeyFromURL('https://drive.uc.cn/s/xxx')).toBe('uc')
    expect(detectProviderKeyFromURL('https://wwx.lanzoui.com/xxx')).toBe(
      'lanzou'
    )
    expect(detectProviderKeyFromURL('https://mega.nz/file/xxx')).toBe('other')
  })
})

describe('splitResourceLinkText', () => {
  it('splits on ASCII and Chinese commas', () => {
    expect(
      splitResourceLinkText(
        'https://a.example/1， https://b.example/2,https://c.example/3'
      )
    ).toEqual([
      'https://a.example/1',
      'https://b.example/2',
      'https://c.example/3'
    ])
  })
})

describe('isCleanLinkDump', () => {
  it('accepts comma-separated clean URLs', () => {
    expect(
      isCleanLinkDump('https://pan.baidu.com/s/aaa, https://pan.quark.cn/s/bbb')
    ).toBe(true)
  })

  it('rejects share-dump prose', () => {
    expect(
      isCleanLinkDump('链接: https://pan.baidu.com/s/aaa 提取码: ab12')
    ).toBe(false)
  })
})

describe('applyResourceLinkPaste', () => {
  it('replaces the field when the selection covers everything', () => {
    const result = applyResourceLinkPaste({
      pasted: 'https://pan.baidu.com/s/new 提取码: zz99',
      existingLinks: ['https://old.example/a'],
      existingCode: '',
      existingPassword: '',
      replaceAll: true
    })
    expect(result.applied).toBe(true)
    expect(result.links).toEqual(['https://pan.baidu.com/s/new'])
    expect(result.code).toBe('zz99')
    expect(result.notify).toContain('提取码')
    expect(result.notify).toContain('百度网盘')
  })

  it('merges when pasting into a field that already has links', () => {
    const result = applyResourceLinkPaste({
      pasted: 'https://pan.quark.cn/s/bbb',
      existingLinks: ['https://pan.baidu.com/s/aaa'],
      existingCode: 'ab12',
      existingPassword: '',
      replaceAll: false
    })
    expect(result.links).toEqual([
      'https://pan.baidu.com/s/aaa',
      'https://pan.quark.cn/s/bbb'
    ])
    expect(result.code).toBe('ab12')
  })

  it('does not overwrite an extraction code the user already typed', () => {
    const result = applyResourceLinkPaste({
      pasted: 'https://pan.baidu.com/s/aaa 提取码: new1',
      existingLinks: [],
      existingCode: 'old1',
      existingPassword: '',
      replaceAll: true
    })
    expect(result.code).toBe('old1')
  })

  it('rescues a code-only paste into the link field', () => {
    const result = applyResourceLinkPaste({
      pasted: '提取码: ab12',
      existingLinks: ['https://pan.baidu.com/s/aaa'],
      existingCode: '',
      existingPassword: '',
      replaceAll: false
    })
    expect(result.applied).toBe(true)
    expect(result.links).toEqual(['https://pan.baidu.com/s/aaa'])
    expect(result.code).toBe('ab12')
  })

  it('lets a paste with no link and no code through', () => {
    const result = applyResourceLinkPaste({
      pasted: '随便一段话',
      existingLinks: [],
      existingCode: '',
      existingPassword: '',
      replaceAll: true
    })
    expect(result.applied).toBe(false)
  })
})

describe('applyResourceLinkBlur', () => {
  it('cleans a typed dump on blur', () => {
    const result = applyResourceLinkBlur(
      '链接：https://pan.baidu.com/s/aaa 提取码：ab12',
      '',
      ''
    )
    expect(result.applied).toBe(true)
    expect(result.links).toEqual(['https://pan.baidu.com/s/aaa'])
    expect(result.code).toBe('ab12')
  })

  it('fills a query-string extraction code from an otherwise clean URL', () => {
    const result = applyResourceLinkBlur(
      'https://pan.baidu.com/s/aaa?pwd=ab12',
      '',
      ''
    )
    expect(result.applied).toBe(true)
    expect(result.links).toEqual(['https://pan.baidu.com/s/aaa?pwd=ab12'])
    expect(result.code).toBe('ab12')
  })

  it('is a no-op for already-clean links', () => {
    const result = applyResourceLinkBlur(
      'https://pan.baidu.com/s/aaa, https://pan.quark.cn/s/bbb',
      'ab12',
      ''
    )
    expect(result.applied).toBe(false)
    expect(result.links).toEqual([
      'https://pan.baidu.com/s/aaa',
      'https://pan.quark.cn/s/bbb'
    ])
    expect(result.code).toBe('ab12')
  })
})
