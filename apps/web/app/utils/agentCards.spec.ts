import { describe, expect, it } from 'vitest'
import {
  agentCardEntity,
  agentCardHref,
  agentCardIsWork,
  agentCardPreviewable,
  agentCardRating,
  agentCardRatingsText,
  agentCardTitle,
  agentTagChips
} from './agentCards'

const card = (over: Partial<AgentEntityCard>): AgentEntityCard => ({
  kind: 'work',
  id: '3032',
  title: '千恋＊万花',
  content_limit: 'sfw',
  ...over
})

describe('agentCardTitle', () => {
  it('prefers a Chinese localized name over the display title', () => {
    expect(
      agentCardTitle(
        card({
          localized: {
            ja: { value: 'センレンバンカ' },
            'zh-Hans': { value: '千恋万花' }
          }
        })
      )
    ).toBe('千恋万花')
    expect(
      agentCardTitle(card({ localized: { ja: { value: 'センレンバンカ' } } }))
    ).toBe('千恋＊万花')
  })
})

describe('agentCardHref', () => {
  it('opens each catalog kind on its own page', () => {
    expect(agentCardHref(card({}))).toBe('/galgame/3032')
    expect(agentCardHref(card({ kind: 'character', id: '39099' }))).toBe(
      '/galgame/character/39099'
    )
    expect(agentCardHref(card({ kind: 'person', id: '88794' }))).toBe(
      '/galgame/staff/88794'
    )
    expect(agentCardHref(card({ kind: 'company', id: '12' }))).toBe(
      '/galgame/official/12'
    )
    expect(agentCardHref(card({ kind: 'series', id: '7' }))).toBe(
      '/galgame/series/7'
    )
  })

  it('previews a forum topic card through the topic preview', () => {
    const topic = card({ kind: 'kungal:topic', id: '88', url: '/topic/88' })
    expect(agentCardHref(topic)).toBe('/topic/88')
    expect(agentCardEntity(topic)).toEqual({ type: 'topic', id: '88' })
    expect(agentCardIsWork(topic)).toBe(false)
  })

  it('names a person by its credit name', () => {
    expect(agentCardEntity(card({ kind: 'person', id: '5' }))).toEqual({
      type: 'credit_name',
      id: '5'
    })
  })

  it('has no preview for another site card or a malformed id', () => {
    expect(agentCardPreviewable(card({ kind: 'letmoe:game', id: '148' }))).toBe(
      false
    )
    expect(agentCardHref(card({ kind: 'letmoe:game', id: '148' }))).toBeNull()
    expect(agentCardPreviewable(card({ id: '0' }))).toBe(false)
  })
})

describe('agentTagChips', () => {
  it('links each tag card to its tag page', () => {
    expect(
      agentTagChips([card({ kind: 'tag', id: '5', title: '催泪' }), card({})])
    ).toEqual([{ id: '5', name: '催泪', href: '/galgame/tag/5' }])
  })
})

describe('agentCardRating', () => {
  it('prefers Bangumi and skips unscored sources', () => {
    const c = card({
      ratings: [
        { source: 'vndb', score: 8.7, votes: 11510 },
        { source: 'bangumi', score: 8.9, votes: 6522 }
      ]
    })
    expect(agentCardRating(c)).toEqual({ label: 'Bangumi', score: '8.9' })
    expect(agentCardRatingsText(c)).toBe(
      'Bangumi 8.9（6522 人） · VNDB 8.7（11510 人）'
    )
    expect(
      agentCardRating(
        card({ ratings: [{ source: 'bangumi', score: 0, votes: 0 }] })
      )
    ).toBeNull()
    expect(agentCardHref(card({ kind: 'tag', id: '5' }))).toBe('/galgame/tag/5')
  })
})
