// @vitest-environment nuxt
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'

const hash = 'ab'.repeat(32)
const other = 'cd'.repeat(32)

const { get } = vi.hoisted(() => ({ get: vi.fn() }))

mockNuxtImport('useApiClient', () => () => ({ GET: get }))

beforeEach(() => {
  useState('kun-sticker-packs').value = []
  useState('kun-sticker-images').value = {}
  get.mockResolvedValue({
    response: new Response(null, { status: 200 }),
    data: {
      items: [
        {
          display_name: 'pack',
          stickers: [
            {
              display_name: 's',
              image: { hash, width: 640, height: 480, thumbhash: 'th' }
            }
          ]
        }
      ]
    }
  })
})

describe('useStickerPacks', () => {
  it('knows a sticker by hash in both the CDN and the token form', async () => {
    const { load, isSticker, image } = useStickerPacks()
    const [pack] = await load()
    const src = pack!.stickers[0]!.src

    expect(src).toContain(`${hash}_320`)
    expect(isSticker(src)).toBe(true)
    expect(isSticker(`/image/${hash}_320`)).toBe(true)
    expect(isSticker(`/image/${other}`)).toBe(false)
    expect(isSticker('https://elsewhere.example/a.png')).toBe(false)
    expect(image(src)).toEqual({
      hash,
      width: 640,
      height: 480,
      thumbhash: 'th'
    })
  })

  it('answers no before the packs have loaded', () => {
    expect(useStickerPacks().isSticker(`/image/${hash}_320`)).toBe(false)
  })
})
