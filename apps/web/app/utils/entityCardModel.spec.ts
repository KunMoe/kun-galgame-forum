// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import type {
  EntityTarget,
  Image,
  WorkPreview,
  WorkRef
} from '#shared/utils/api/schemas'
import { entityCardModel } from './entityCardModel'

const image = (sexual: Image['sexual'] = 'safe'): Image => ({
  url: 'https://cdn.example/c.webp',
  hash: 'c',
  width: 500,
  height: 700,
  thumbhash: null,
  sexual
})

const work = (over: Partial<WorkRef> = {}): WorkRef => ({
  object: 'work',
  id: '3',
  display_name: 'アストラエアの白き永遠',
  latin: null,
  localized: { 'zh-Hans': { value: '星辰恋曲的白色永恒', is_machine: false } },
  cover: image(),
  is_nsfw: false,
  ...over
})

const workPreview = (over: Partial<WorkRef> = {}): WorkPreview => ({
  ...work(over),
  maker: null,
  top_tags: [],
  release_date: null,
  release_date_precision: null
})

const author = {
  object: 'user' as const,
  id: '9',
  name: 'shin',
  avatar: null,
  avatar_frame: null
}

describe('entityCardModel', () => {
  it('blurs an NSFW cover under blur and drops it under hide', () => {
    const t: EntityTarget = workPreview({ is_nsfw: true })
    const blur = entityCardModel(t, false, 'blur')
    expect(blur.media).toMatchObject({ kind: 'image', isBlurred: true })
    expect(blur.badges.map((b) => b.label)).toContain('NSFW')
    expect(entityCardModel(t, false, 'hide').media.kind).toBe('icon')
    expect(entityCardModel(t, false, 'show').media).toMatchObject({
      kind: 'image',
      isBlurred: false
    })
  })

  it('treats a cover graded suggestive as sensitive even on a safe work', () => {
    const t: EntityTarget = workPreview({ cover: image('suggestive') })
    expect(entityCardModel(t, false, 'hide').media.kind).toBe('icon')
  })

  it('never shows a spoiler summary', () => {
    const t: EntityTarget = {
      object: 'rating',
      id: '1',
      work: work(),
      author,
      overall: 9,
      short_summary: '结局是……',
      has_spoilers: true,
      created_at: '2026-10-01T00:00:00Z'
    }
    const card = entityCardModel(t, false, 'show')
    expect(card.subtitle).not.toContain('结局')
    expect(card.score).toBe(9)
    expect(card.badges.map((b) => b.label)).toContain('剧透')
  })

  it('titles a resource card by its work, with the resource below', () => {
    const t: EntityTarget = {
      object: 'galgame_resource',
      id: '61797',
      work: work(),
      author,
      resource_type: 'game',
      resource_languages: ['zh-cn'],
      resource_platforms: ['win'],
      title: '【PC+安卓直装】',
      size: '20 GB',
      state: 'valid',
      download_count: 0,
      like_count: 0,
      created_at: '2026-10-01T00:00:00Z'
    }
    const card = entityCardModel(t, false, 'show')
    expect(card.title).toBe('星辰恋曲的白色永恒')
    expect(card.subtitle).toBe('【PC+安卓直装】')
    expect(card.href).toBe('/galgame/resource/61797')
  })

  it('links an unavailable target to the page it named', () => {
    const card = entityCardModel(
      { object: 'unavailable_entity', entity_type: 'rating', entity_id: '5' },
      false,
      'show'
    )
    expect(card.href).toBe('/galgame-rating/5')
    expect(card.isUnavailable).toBe(true)
    expect(card.title).toBe('无法预览的Galgame 评分')
  })

  it('leads a Galgame card with its maker, year and main tags', () => {
    const t: EntityTarget = {
      ...workPreview(),
      maker: {
        object: 'company',
        id: '2',
        display_name: 'FAVORITE',
        latin: null,
        localized: {}
      },
      top_tags: [
        {
          object: 'tag',
          id: '1',
          display_name: '妹',
          latin: null,
          localized: {},
          is_sexual: false
        }
      ],
      release_date: '2014-05-12',
      release_date_precision: 'day'
    }
    const card = entityCardModel(t, false, 'show')
    expect(card.info).toBe('FAVORITE · 2014 年')
    expect(card.chips).toEqual(['妹'])
    expect(card.subtitle).toBe('アストラエアの白き永遠')
  })

  it('says where a character comes from and what a person made', () => {
    const character: EntityTarget = {
      object: 'character',
      id: '5',
      display_name: '雪姫',
      latin: null,
      localized: {},
      image: image(),
      works: [work()]
    }
    expect(entityCardModel(character, false, 'show').info).toBe(
      '出自 《星辰恋曲的白色永恒》'
    )
    const person: EntityTarget = {
      object: 'credit_name',
      id: '6',
      display_name: '丸戸史明',
      latin: null,
      localized: {},
      photo: null,
      role_names: ['脚本'],
      works: [work()]
    }
    const card = entityCardModel(person, false, 'show')
    expect(card.chips).toEqual(['脚本'])
    expect(card.info).toBe('代表作 《星辰恋曲的白色永恒》')
  })

  it('picks a layout that fits how much the kind has to show', () => {
    const tag: EntityTarget = {
      object: 'tag',
      id: '1',
      display_name: '白发男主角',
      latin: null,
      localized: {},
      tag_kind: 'content',
      is_sexual: false,
      catalog_work_count: 484,
      description: ''
    }
    expect(entityCardModel(tag, false, 'show').layout).toBe('pill')
    const company: EntityTarget = {
      object: 'company',
      id: '2',
      display_name: 'Key',
      latin: null,
      localized: {},
      company_kind: 'game_brand',
      logo: image(),
      catalog_work_count: 29
    }
    expect(entityCardModel(company, false, 'show').layout).toBe('pill')
    expect(
      entityCardModel(workPreview({ cover: null }), false, 'show').layout
    ).toBe('compact')
    expect(entityCardModel(workPreview(), false, 'show').layout).toBe('full')
  })

  it('frames a landscape cover as landscape', () => {
    const wide = { ...image(), width: 560, height: 420 }
    expect(
      entityCardModel(workPreview({ cover: wide }), false, 'show').media
    ).toMatchObject({
      kind: 'image',
      shape: 'landscape'
    })
    expect(entityCardModel(workPreview(), false, 'show').media).toMatchObject({
      shape: 'portrait'
    })
  })
})
