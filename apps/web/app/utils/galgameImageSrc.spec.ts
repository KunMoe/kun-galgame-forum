import { describe, it, expect } from 'vitest'
import { galgameImageSrc } from './galgameImageSrc'
import { imageHashUrl } from '#shared/utils/imageService'

describe('galgameImageSrc', () => {
  it('prefers the server-injected cdn_url when present', () => {
    expect(
      galgameImageSrc({
        cdn_url: 'https://image.example/ab/cd/hash.webp',
        image_hash: 'abcd'
      })
    ).toBe('https://image.example/ab/cd/hash.webp')
  })

  it('resolves image_hash to an ABSOLUTE, sharded CDN URL when cdn_url is missing', () => {
    expect(galgameImageSrc({ image_hash: 'deadbeef' })).toMatch(
      /^https:\/\/[^/]+\/de\/ad\/deadbeef\.webp$/
    )
  })

  it('resolves when cdn_url is an empty string', () => {
    expect(galgameImageSrc({ cdn_url: '', image_hash: 'feedface' })).toMatch(
      /^https:\/\/[^/]+\/fe\/ed\/feedface\.webp$/
    )
  })
})

describe('imageHashUrl via galgameImageSrc', () => {
  it('builds {base}/<aa>/<bb>/<hash>.webp (trailing slash trimmed)', () => {
    expect(imageHashUrl('https://cdn.test/', 'deadbeef')).toBe(
      'https://cdn.test/de/ad/deadbeef.webp'
    )
  })

  it('appends a variant suffix', () => {
    expect(imageHashUrl('https://cdn.test', 'deadbeef', 'mini')).toBe(
      'https://cdn.test/de/ad/deadbeef_mini.webp'
    )
  })
})
