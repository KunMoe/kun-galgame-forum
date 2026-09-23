// @vitest-environment nuxt
import { describe, it, expect, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { createGalgameEditConfig } from './galgameEdit'
import { KUN_GALGAME_OFFICIAL_KIND_DEVELOPER } from './galgameOfficial'

mockNuxtImport('uploadWorkEditImage', () =>
  vi.fn(async () => ({
    hash: 'h-new',
    url: 'https://img.test/h-new.webp',
    width: 1,
    height: 1,
    thumbhash: null,
    sexual: null
  }))
)
mockNuxtImport('useMessage', () => vi.fn())

const K = (name: string) => `catalog.work.${name}`
const file = new File(['x'], 'x.webp', { type: 'image/webp' })

describe('galgameEdit — image upload hooks', () => {
  const config = createGalgameEditConfig()

  it('a new cover row carries ONLY engine-known keys', async () => {
    const item = await config[K('covers')]!.uploadImage!(file, [])
    expect(item).toStrictEqual({ image_hash: 'h-new' })
  })

  it('a new screenshot row carries ONLY engine-known keys', async () => {
    const item = await config[K('screenshots')]!.uploadImage!(file, [])
    expect(item).toStrictEqual({ image_hash: 'h-new' })
  })

  it('an already-attached hash is skipped, not duplicated', async () => {
    const item = await config[K('covers')]!.uploadImage!(file, [
      { image_hash: 'h-new' }
    ])
    expect(item).toBeNull()
  })

  it('no normalizeItems hook — the engine derives sort_order itself', async () => {
    expect(config[K('covers')]!.normalizeItems).toBeUndefined()
    expect(config[K('screenshots')]!.normalizeItems).toBeUndefined()
  })
})

describe('galgameEdit — relation items render as names', () => {
  const config = createGalgameEditConfig({
    tag: new Map([[84, '恋爱']]),
    official: new Map([[2, 'Key']])
  })

  it('resolves an id through the host map and falls back to #id', () => {
    const fmt = config[K('tag_ids')]!.formatItem!
    expect(fmt(84)).toBe('恋爱')
    expect(fmt(999)).toBe('#999')
  })

  it('a 会社 edge shows name and attribution kind', () => {
    const fmt = config[K('labels')]!.formatItem!
    expect(
      fmt({ label_id: 2, kind: KUN_GALGAME_OFFICIAL_KIND_DEVELOPER })
    ).toBe('Key · 开发商')
  })
})
