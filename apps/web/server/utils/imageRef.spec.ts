import { describe, it, expect } from 'vitest'
import {
  imageRefCdnUrl,
  parseAbsoluteImageUrl,
  parseImageRef,
  transformStickerPacks
} from './imageRef'

const hash = '7835f792543f8564cf95e7f84d4828f2a3ef735293f0844bf8ddf8f39371171d'
const base = 'https://image.kungal.iloveren.link'

describe('parseImageRef', () => {
  it('accepts a 64-hex hash', () => {
    expect(parseImageRef(hash)).toEqual({ hash })
  })

  it('accepts a 64-hex hash with a variant', () => {
    expect(parseImageRef(`${hash}_320`)).toEqual({ hash, variant: '320' })
  })

  it('rejects 63 hex', () => {
    expect(parseImageRef(hash.slice(1))).toBeNull()
  })

  it('rejects uppercase hex', () => {
    expect(parseImageRef(hash.toUpperCase())).toBeNull()
  })

  it('rejects path traversal', () => {
    expect(parseImageRef(`../${hash}`)).toBeNull()
    expect(parseImageRef(`${hash}/../x`)).toBeNull()
  })

  it('rejects empty', () => {
    expect(parseImageRef('')).toBeNull()
  })

  it('rejects kohaku.webp', () => {
    expect(parseImageRef('kohaku.webp')).toBeNull()
  })
})

describe('imageRefCdnUrl', () => {
  it('builds a sharded main URL', () => {
    expect(imageRefCdnUrl(base, { hash })).toBe(`${base}/78/35/${hash}.webp`)
  })

  it('builds a sharded variant URL', () => {
    expect(imageRefCdnUrl(base, { hash, variant: '320' })).toBe(
      `${base}/78/35/${hash}_320.webp`
    )
  })

  it('strips a trailing slash on the base', () => {
    expect(imageRefCdnUrl(base + '/', { hash })).toBe(
      `${base}/78/35/${hash}.webp`
    )
  })
})

describe('transformStickerPacks', () => {
  it('prefers hash when present and emits a token src', () => {
    const out = transformStickerPacks({
      packs: [
        {
          name: 'p',
          stickers: [{ src: 'https://other.example/x.webp', name: 'a', hash }]
        }
      ]
    })
    expect(out.packs[0]!.stickers[0]!.src).toBe(`/image/${hash}_320`)
  })

  it('extracts the hash from an absolute src on a known host', () => {
    const src = `${base}/78/35/${hash}_320.webp`
    const out = transformStickerPacks({
      packs: [{ name: 'p', stickers: [{ src, name: 'a' }] }]
    })
    expect(out.packs[0]!.stickers[0]!.src).toBe(`/image/${hash}_320`)
  })

  it('leaves a sticker on an unknown host untouched', () => {
    const src = `https://cdn.example/78/35/${hash}.webp`
    const out = transformStickerPacks({
      packs: [{ name: 'p', stickers: [{ src, name: 'a' }] }]
    })
    expect(out.packs[0]!.stickers[0]!.src).toBe(src)
  })
})

describe('parseAbsoluteImageUrl', () => {
  it('accepts a known host with matching shards', () => {
    expect(parseAbsoluteImageUrl(`${base}/78/35/${hash}_320.webp`)).toEqual({
      hash,
      variant: '320'
    })
  })

  it('rejects wrong shards', () => {
    expect(parseAbsoluteImageUrl(`${base}/00/00/${hash}.webp`)).toBeNull()
  })
})
