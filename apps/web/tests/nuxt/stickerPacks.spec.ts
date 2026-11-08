// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'

const hash = 'ab'.repeat(32)
const other = 'cd'.repeat(32)

describe('useStickerPacks().isSticker', () => {
  it('knows a sticker by hash in both the CDN and the token form', () => {
    useState('kun-sticker-packs').value = [
      {
        name: 'pack',
        stickers: [
          { name: 's', src: `https://img.example/ab/ab/${hash}_320.webp` }
        ]
      }
    ]
    const { isSticker } = useStickerPacks()

    expect(isSticker(`https://img.example/ab/ab/${hash}_320.webp`)).toBe(true)
    expect(isSticker(`/image/${hash}_320`)).toBe(true)
    expect(isSticker(`/image/${other}`)).toBe(false)
    expect(isSticker('https://elsewhere.example/a.png')).toBe(false)
  })

  it('answers no before the packs have loaded', () => {
    useState('kun-sticker-packs').value = []
    expect(useStickerPacks().isSticker(`/image/${hash}_320`)).toBe(false)
  })
})
