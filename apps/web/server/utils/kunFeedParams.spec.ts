import { describe, expect, it } from 'vitest'
import {
  feedCacheKey,
  parseSiteFeed,
  parseUserFeed,
  parseWorkFeed
} from './kunFeedParams'

const nitroEscape = (key: string) => key.replace(/\W/g, '')

describe('parseSiteFeed', () => {
  it('parses the four site feeds with their paths', () => {
    expect(parseSiteFeed('topic.xml', {})).toEqual({
      format: 'xml',
      path: '/rss/topic',
      params: { name: 'topic', includeNsfw: false, section: undefined },
      urlParams: { section: undefined }
    })
    expect(parseSiteFeed('galgame.atom', {})?.path).toBe('/rss/galgame')
    expect(parseSiteFeed('news.json', {})?.path).toBe('/rss/news')
    expect(parseSiteFeed('toolset.xml', {})).toEqual({
      format: 'xml',
      path: '/rss/toolset',
      params: { name: 'toolset' },
      urlParams: {}
    })
  })

  it('rejects unknown feeds and extensions', () => {
    expect(parseSiteFeed('reply.xml', {})).toBeNull()
    expect(parseSiteFeed('topic.html', {})).toBeNull()
    expect(parseSiteFeed('topic', {})).toBeNull()
  })

  it('reads include_nsfw only as 1 or true, first value of a repeated key', () => {
    const nsfw = (value: unknown) =>
      parseSiteFeed('galgame.xml', { include_nsfw: value })?.params
    expect(nsfw('1')).toEqual({ name: 'galgame', includeNsfw: true })
    expect(nsfw('true')).toEqual({ name: 'galgame', includeNsfw: true })
    expect(nsfw('yes')).toEqual({ name: 'galgame', includeNsfw: false })
    expect(nsfw(['1', '0'])).toEqual({ name: 'galgame', includeNsfw: true })
  })

  it('accepts a known section and rejects anything else', () => {
    expect(
      parseSiteFeed('topic.xml', { section: 'g-chatting' })?.params
    ).toEqual({ name: 'topic', includeNsfw: false, section: 'g-chatting' })
    expect(parseSiteFeed('topic.xml', { section: 'g-nope' })).toBeNull()
    expect(parseSiteFeed('topic.xml', { section: 'G-chatting' })).toBeNull()
    expect(parseSiteFeed('topic.xml', { section: '' })).toBeNull()
  })

  it('accepts news lanes and source keys and rejects anything else', () => {
    expect(
      parseSiteFeed('news.xml', { lane: 'column', source: 'ymgal' })?.params
    ).toEqual({ name: 'news', lane: 'column', source: 'ymgal' })
    expect(parseSiteFeed('news.xml', { lane: 'blog' })).toBeNull()
    expect(parseSiteFeed('news.xml', { lane: '' })).toBeNull()
    expect(parseSiteFeed('news.xml', { source: 'bad-key' })).toBeNull()
  })
})

describe('parseWorkFeed', () => {
  it('parses a work id and include_nsfw', () => {
    expect(parseWorkFeed('1383.atom', { include_nsfw: '1' })).toEqual({
      format: 'atom',
      path: '/rss/galgame/1383',
      params: { workId: '1383', includeNsfw: true },
      urlParams: { include_nsfw: true }
    })
  })

  it('rejects ids that are not positive decimal integers', () => {
    expect(parseWorkFeed('0.xml', {})).toBeNull()
    expect(parseWorkFeed('01.xml', {})).toBeNull()
    expect(parseWorkFeed('abc.xml', {})).toBeNull()
    expect(parseWorkFeed('1383.html', {})).toBeNull()
  })
})

describe('parseUserFeed', () => {
  it('parses the topic and galgame feeds of a user', () => {
    expect(parseUserFeed('8', 'topic.xml', {})).toEqual({
      format: 'xml',
      path: '/rss/user/8/topic',
      params: { userId: '8', name: 'topic', includeNsfw: false },
      urlParams: {}
    })
    expect(parseUserFeed('8', 'galgame.json', {})?.path).toBe(
      '/rss/user/8/galgame'
    )
  })

  it('rejects other feeds and bad user ids', () => {
    expect(parseUserFeed('8', 'reply.xml', {})).toBeNull()
    expect(parseUserFeed('08', 'topic.xml', {})).toBeNull()
    expect(parseUserFeed('x', 'topic.xml', {})).toBeNull()
  })
})

describe('feedCacheKey', () => {
  it('keeps feeds apart that a colon-joined key would merge after Nitro escaping', () => {
    const a = feedCacheKey(
      parseSiteFeed('news.xml', { lane: 'news', source: 'ymgal' })
    )
    const b = feedCacheKey(parseSiteFeed('news.xml', { source: 'newsymgal' }))
    expect(nitroEscape('news:xml:news:ymgal')).toBe(
      nitroEscape('news:xml::newsymgal')
    )
    expect(a).not.toBe(b)
  })

  it('is unchanged by Nitro escaping', () => {
    const key = feedCacheKey(parseWorkFeed('1383.xml', {}))
    expect(key).toMatch(/^[0-9a-f]{64}$/)
    expect(nitroEscape(key)).toBe(key)
  })

  it('ignores parameters the feed does not read', () => {
    expect(feedCacheKey(parseSiteFeed('toolset.xml', { foo: 'x' }))).toBe(
      feedCacheKey(parseSiteFeed('toolset.xml', {}))
    )
    expect(
      feedCacheKey(parseSiteFeed('toolset.xml', { include_nsfw: '1' }))
    ).toBe(feedCacheKey(parseSiteFeed('toolset.xml', {})))
    expect(
      feedCacheKey(parseSiteFeed('galgame.xml', { include_nsfw: '1' }))
    ).toBe(feedCacheKey(parseSiteFeed('galgame.xml', { include_nsfw: 'true' })))
  })

  it('separates formats, nsfw, sections, works and users', () => {
    const keys = [
      parseSiteFeed('topic.xml', {}),
      parseSiteFeed('topic.atom', {}),
      parseSiteFeed('topic.xml', { include_nsfw: '1' }),
      parseSiteFeed('topic.xml', { section: 'g-chatting' }),
      parseWorkFeed('12.xml', {}),
      parseWorkFeed('123.xml', {}),
      parseUserFeed('12', 'topic.xml', {}),
      parseUserFeed('12', 'galgame.xml', {})
    ].map(feedCacheKey)
    expect(new Set(keys).size).toBe(keys.length)
  })

  it('gives every unparseable request one fixed key', () => {
    expect(feedCacheKey(null)).toBe('invalid')
  })
})
