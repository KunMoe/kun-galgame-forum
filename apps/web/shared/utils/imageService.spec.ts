import { describe, it, expect } from 'vitest'
import { imageHashUrl, parseImageToken } from './imageService'

const hash = '7835f792543f8564cf95e7f84d4828f2a3ef735293f0844bf8ddf8f39371171d'
const base = 'https://image.kungal.iloveren.link'

describe('parseImageToken', () => {
  it('accepts a 64-hex hash', () => {
    expect(parseImageToken(hash)).toEqual({ hash })
  })

  it('accepts a 64-hex hash with a variant', () => {
    expect(parseImageToken(`${hash}_320`)).toEqual({ hash, variant: '320' })
  })

  it('rejects 63 hex', () => {
    expect(parseImageToken(hash.slice(1))).toBeNull()
  })

  it('rejects uppercase hex', () => {
    expect(parseImageToken(hash.toUpperCase())).toBeNull()
  })

  it('rejects path traversal', () => {
    expect(parseImageToken(`../${hash}`)).toBeNull()
    expect(parseImageToken(`${hash}/../x`)).toBeNull()
  })

  it('rejects empty', () => {
    expect(parseImageToken('')).toBeNull()
  })

  it('rejects kohaku.webp', () => {
    expect(parseImageToken('kohaku.webp')).toBeNull()
  })
})

describe('imageHashUrl', () => {
  it('builds a sharded main URL', () => {
    expect(imageHashUrl(base, hash)).toBe(`${base}/78/35/${hash}.webp`)
  })

  it('builds a sharded variant URL', () => {
    expect(imageHashUrl(base, hash, '320')).toBe(
      `${base}/78/35/${hash}_320.webp`
    )
  })

  it('strips a trailing slash on the base', () => {
    expect(imageHashUrl(base + '/', hash)).toBe(`${base}/78/35/${hash}.webp`)
  })
})
