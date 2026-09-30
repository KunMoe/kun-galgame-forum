import { describe, expect, it } from 'vitest'
import {
  KUN_DEFAULT_FEED_TABS,
  type KunFeedTab,
  parseFeedTabs,
  upgradeFeedTabs
} from './activity'

const v7Tabs = (): KunFeedTab[] =>
  structuredClone(KUN_DEFAULT_FEED_TABS).filter((t) => t.id !== 'following')

describe('upgradeFeedTabs', () => {
  it('adds the following tab after 话题 and keeps the kinds a user chose', () => {
    const tabs = v7Tabs()
    tabs[0]!.kinds = ['TOPIC_NORMAL', 'TOPIC_REPLY_CREATION']

    const out = upgradeFeedTabs(tabs, 7)

    expect(out.map((t) => t.id)).toEqual(KUN_DEFAULT_FEED_TABS.map((t) => t.id))
    expect(out[0]!.kinds).toEqual(['TOPIC_NORMAL', 'TOPIC_REPLY_CREATION'])
    expect(out[1]!.source).toBe('following')
  })

  it('resets tabs stored before version 7', () => {
    const out = upgradeFeedTabs(
      [{ id: 'topic', name: 'x', icon: '', kinds: [] }],
      6
    )
    expect(out).toEqual(KUN_DEFAULT_FEED_TABS)
  })

  it('changes nothing when every default tab is present', () => {
    const tabs = structuredClone(KUN_DEFAULT_FEED_TABS)
    expect(upgradeFeedTabs(tabs, 8)).toEqual(tabs)
  })
})

describe('parseFeedTabs', () => {
  it('parses the default tabs into an equal value', () => {
    expect(parseFeedTabs(KUN_DEFAULT_FEED_TABS)).toEqual(KUN_DEFAULT_FEED_TABS)
  })

  it('returns a copy', () => {
    const input = structuredClone(KUN_DEFAULT_FEED_TABS)
    const out = parseFeedTabs(input)
    out![0]!.kinds.push('EXTRA')
    expect(input[0]!.kinds).toEqual(KUN_DEFAULT_FEED_TABS[0]!.kinds)
  })

  it('accepts an empty array', () => {
    expect(parseFeedTabs([])).toEqual([])
  })

  it('rejects null, an object, and a string', () => {
    expect(parseFeedTabs(null)).toBeNull()
    expect(parseFeedTabs({ id: 'topic' })).toBeNull()
    expect(parseFeedTabs('topic')).toBeNull()
  })

  it('rejects a non-string kind', () => {
    expect(
      parseFeedTabs([
        { id: 'topic', name: '话题', icon: 'lucide:box', kinds: [1] }
      ])
    ).toBeNull()
  })

  it('rejects a tab missing icon', () => {
    expect(
      parseFeedTabs([{ id: 'topic', name: '话题', kinds: ['TOPIC_NORMAL'] }])
    ).toBeNull()
  })

  it('rejects a bogus source', () => {
    expect(
      parseFeedTabs([
        {
          id: 'topic',
          name: '话题',
          icon: 'lucide:box',
          kinds: [],
          source: 'bogus'
        }
      ])
    ).toBeNull()
  })
})
