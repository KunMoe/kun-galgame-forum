import { describe, expect, it } from 'vitest'
import { canonicalPath, encodeLocation } from './kunLocation'

const isHeaderSafe = (s: string) => /^[\x21-\x7e]*$/.test(s)

describe('canonicalPath', () => {
  it('strips a copied zero-width suffix and moves the legacy prefix', () => {
    expect(canonicalPath('/galgame-resource/12342‌')).toBe(
      '/galgame/resource/12342'
    )
    expect(canonicalPath('/galgame/resource/12342​')).toBe(
      '/galgame/resource/12342'
    )
    expect(canonicalPath('/﻿topic/1⁠‎/')).toBe('/topic/1')
  })

  it('moves the legacy resource index and keeps unrelated prefixes', () => {
    expect(canonicalPath('/galgame-resource')).toBe('/galgame/resource')
    expect(canonicalPath('/galgame-resource/')).toBe('/galgame/resource')
    expect(canonicalPath('/galgame-resources/1')).toBe('/galgame-resources/1')
  })

  it('drops trailing slashes but keeps the root', () => {
    expect(canonicalPath('/topic/1//')).toBe('/topic/1')
    expect(canonicalPath('/')).toBe('/')
    expect(canonicalPath('/‌')).toBe('/')
  })

  it('leaves canonical paths alone', () => {
    for (const p of ['/topic/1', '/category/中文', '/website-tag/a%2Fb'])
      expect(canonicalPath(p)).toBe(p)
  })
})

describe('encodeLocation', () => {
  it('encodes non-Latin-1 paths once', () => {
    const to = encodeLocation('/category/中文')
    expect(to).toBe('/category/%E4%B8%AD%E6%96%87')
    expect(decodeURI(to)).toBe('/category/中文')
  })

  it('does not double-encode an existing escape', () => {
    expect(encodeLocation('/category/%E4%B8%AD%E6%96%87')).toBe(
      '/category/%E4%B8%AD%E6%96%87'
    )
    expect(encodeLocation('/website-tag/a%2Fb%25')).toBe(
      '/website-tag/a%2Fb%25'
    )
  })

  it('encodes a bare percent sign and characters outside RFC 3986', () => {
    expect(encodeLocation('/a/100%')).toBe('/a/100%25')
    expect(encodeLocation('/a b/"x"<y>')).toBe('/a%20b/%22x%22%3Cy%3E')
    expect(encodeLocation('/a/😀')).toBe('/a/%F0%9F%98%80')
  })

  it('keeps the query string intact and encodes what it must', () => {
    expect(encodeLocation('/topic/1?page=2&sort=hot')).toBe(
      '/topic/1?page=2&sort=hot'
    )
    expect(encodeLocation('/search?q=中文&tag=%E4%B8%AD')).toBe(
      '/search?q=%E4%B8%AD%E6%96%87&tag=%E4%B8%AD'
    )
  })

  it('always yields a header-safe ASCII Location', () => {
    for (const raw of [
      '/galgame/resource/12342‌',
      '/topic/中文?q=零宽​',
      'https://og.example/a?sig=x%2By&t=中'
    ])
      expect(isHeaderSafe(encodeLocation(raw))).toBe(true)
  })
})

describe('canonicalPath + encodeLocation', () => {
  it('answers the reported URL with one ASCII hop', () => {
    const decoded = decodeURI('/galgame-resource/12342%E2%80%8C')
    expect(encodeLocation(canonicalPath(decoded))).toBe(
      '/galgame/resource/12342'
    )
    expect(encodeLocation(canonicalPath('/topic/中文/') + '?page=2')).toBe(
      '/topic/%E4%B8%AD%E6%96%87?page=2'
    )
  })
})
