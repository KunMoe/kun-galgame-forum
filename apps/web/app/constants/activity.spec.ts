import { describe, expect, it } from 'vitest'
import {
  KUN_DEFAULT_FEED_TABS,
  type KunFeedTab,
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
