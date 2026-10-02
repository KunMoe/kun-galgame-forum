// @vitest-environment nuxt
import { beforeEach, describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import GalgameCovers from './Covers.vue'
import type { WorkCover } from '#shared/utils/api/schemas'

const cover = (
  id: string,
  sexual: 'safe' | 'suggestive' | 'explicit' | null,
  slot: WorkCover['cover_slot'] = 'main'
): WorkCover => ({
  object: 'work_cover',
  id,
  image: {
    hash: `cover${id}`,
    url: `https://cdn.test/cover${id}.webp`,
    width: 600,
    height: 840,
    thumbhash: null,
    sexual
  },
  cover_slot: slot,
  site: 'vndb',
  sort_order: Number(id),
  vote_count: 0
})

const covers = [
  cover('1', 'safe'),
  cover('2', 'suggestive'),
  cover('3', 'explicit', 'pkgback'),
  cover('4', null, 'pkgfront')
]

const mountOpen = async () => {
  await mountSuspended(GalgameCovers, {
    props: { workId: 1, covers, isNsfw: false, modelValue: true },
    attachTo: document.body
  })
  return document.body.innerHTML
}

const rendered = (html: string) =>
  ['1', '2', '3', '4'].filter((id) =>
    html.includes(`https://cdn.test/cover${id}.webp`)
  )

describe('GalgameCovers', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    const user = usePersistUserStore()
    user.id = 0
    user.setContentStance(false, 'hide')
    usePersistSettingsStore().showKUNGalgameContentLimit = 'sfw'
  })

  it('leaves explicit covers out for a reader who hides adult content', async () => {
    const html = await mountOpen()
    expect(rendered(html)).toEqual(['1', '2', '4'])
    expect(html).toContain('1 张成人向封面已按您的内容设置隐藏')
  })

  it('lists every cover once NSFW is on', async () => {
    usePersistSettingsStore().showKUNGalgameContentLimit = 'nsfw'
    const html = await mountOpen()
    expect(rendered(html)).toEqual(['1', '2', '3', '4'])
    expect(html).not.toContain('已按您的内容设置隐藏')
  })
})
