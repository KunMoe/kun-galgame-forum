import { describe, expect, it } from 'vitest'
import { kunFeedUrl } from './feedUrl'

const base = 'https://www.kungal.com'

describe('kunFeedUrl', () => {
  it('builds a URL with no params and the default xml format', () => {
    expect(kunFeedUrl(base, '/rss/topic')).toBe(
      'https://www.kungal.com/rss/topic.xml'
    )
  })

  it('appends one param', () => {
    expect(
      kunFeedUrl(base, '/rss/galgame', 'xml', { include_nsfw: true })
    ).toBe('https://www.kungal.com/rss/galgame.xml?include_nsfw=1')
  })

  it('serialises several params in a fixed order', () => {
    expect(
      kunFeedUrl(base, '/rss/news', 'xml', {
        source: 'ymgal',
        lane: 'column',
        section: 'g-news',
        include_nsfw: true
      })
    ).toBe(
      'https://www.kungal.com/rss/news.xml?include_nsfw=1&section=g-news&lane=column&source=ymgal'
    )
  })

  it('drops empty and false params', () => {
    expect(
      kunFeedUrl(base, '/rss/topic', 'xml', {
        include_nsfw: false,
        section: '',
        lane: '',
        source: ''
      })
    ).toBe('https://www.kungal.com/rss/topic.xml')
  })

  it('builds each format', () => {
    expect(kunFeedUrl(base, '/rss/topic', 'xml')).toBe(
      'https://www.kungal.com/rss/topic.xml'
    )
    expect(kunFeedUrl(base, '/rss/topic', 'atom')).toBe(
      'https://www.kungal.com/rss/topic.atom'
    )
    expect(kunFeedUrl(base, '/rss/topic', 'json')).toBe(
      'https://www.kungal.com/rss/topic.json'
    )
  })
})
