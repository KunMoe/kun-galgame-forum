// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type { RelatedWork, WorkMedium } from '#shared/utils/api/schemas'
import RelatedPanel from './Panel.vue'

const work = (
  id: number,
  medium: WorkMedium,
  name: string,
  phrases: string[] = ['被改编为']
): RelatedWork => ({
  object: 'work',
  id: String(id),
  display_name: name,
  latin: null,
  localized: {},
  cover: null,
  is_nsfw: false,
  medium,
  release_date: null,
  release_date_precision: null,
  relations: phrases.map((phrase) => ({ relation_type: 'x', phrase }))
})

const cardNames = (wrapper: VueWrapper) =>
  wrapper.findAll('h2').map((h) => h.text())

describe('related works', () => {
  it('draws nothing when there are none', async () => {
    const wrapper = await mountSuspended(RelatedPanel, { props: { works: [] } })
    expect(wrapper.text()).toBe('')
  })

  it('cards galgames with their phrases and lists other media as text', async () => {
    const wrapper = await mountSuspended(RelatedPanel, {
      props: {
        works: [
          work(2679, 'galgame', 'Rewrite: Cradles Tale', [
            '有续作',
            '有 Fandisc'
          ]),
          work(213783, 'anime', 'Rewrite'),
          work(213225, 'manga', 'Rewrite: SIDE-B'),
          work(9, 'doujin_game', 'Fan Game')
        ]
      }
    })

    expect(cardNames(wrapper)).toEqual(['Rewrite: Cradles Tale'])
    expect(wrapper.text()).toContain('有续作 · 有 Fandisc')
    expect(wrapper.find('a[href="/galgame/2679"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/galgame/213783"]').exists()).toBe(false)
    expect(wrapper.text()).toContain(
      '其他媒介：Rewrite（动画） · Rewrite: SIDE-B（漫画） · Fan Game（同人游戏）'
    )
  })

  it('folds galgames past twelve and other media past six', async () => {
    const games = Array.from({ length: 14 }, (_, i) =>
      work(100 + i, 'galgame', `game ${i}`)
    )
    const others = Array.from({ length: 8 }, (_, i) =>
      work(500 + i, 'novel', `novel ${i}`)
    )
    const wrapper = await mountSuspended(RelatedPanel, {
      props: { works: [...games, ...others] }
    })

    expect(cardNames(wrapper)).toHaveLength(12)
    expect(wrapper.text()).toContain('novel 5（小说）')
    expect(wrapper.text()).not.toContain('novel 6（小说）')

    const buttons = wrapper.findAll('button')
    await buttons.find((b) => b.text() === '展开全部（14）')!.trigger('click')
    await buttons.find((b) => b.text() === '展开全部（8）')!.trigger('click')

    expect(cardNames(wrapper)).toHaveLength(14)
    expect(wrapper.text()).toContain('novel 7（小说）')
  })
})
