import { describe, expect, it } from 'vitest'
import { parseFeedFile, renderKunFeed, type KunFeedChannel } from './kunFeed'

const baseUrl = 'https://www.kungal.com'

const links = {
  rss: 'https://www.kungal.com/rss/topic.xml',
  atom: 'https://www.kungal.com/rss/topic.atom',
  json: 'https://www.kungal.com/rss/topic.json'
}

const older = new Date('2024-01-01T00:00:00.000Z')
const newer = new Date('2026-06-01T12:00:00.000Z')

const channel = (items: KunFeedChannel['items'] = []): KunFeedChannel => ({
  title: '鲲 Galgame 论坛 - 新话题',
  description: '鲲 Galgame 论坛最新发布的话题',
  link: 'https://www.kungal.com/topic',
  image: 'https://www.kungal.com/kungalgame.webp',
  items
})

const sampleItems = (): KunFeedChannel['items'] => [
  {
    link: 'https://www.kungal.com/topic/1',
    title: 'First',
    date: older,
    description: 'one',
    author: [{ name: 'Ada', link: 'https://www.kungal.com/user/1/info' }]
  },
  {
    link: 'https://www.kungal.com/topic/2',
    title: 'Second',
    date: newer,
    description: 'two',
    author: [{ name: 'Bob', link: 'https://www.kungal.com/user/2/info' }]
  }
]

type XmlNode = {
  name: string
  attrs: Record<string, string>
  children: XmlNode[]
  text: string
}

const decodeXml = (value: string) =>
  value
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'")

const parseXml = (input: string): XmlNode => {
  const source = input.replace(/^\uFEFF/, '')
  let i = 0
  const starts = (lit: string) => source.slice(i, i + lit.length) === lit
  const skipWs = () => {
    while (i < source.length && /\s/.test(source[i]!)) i++
  }
  const parseName = () => {
    const start = i
    while (i < source.length && /[:A-Za-z0-9_.-]/.test(source[i]!)) i++
    return source.slice(start, i)
  }
  const parseAttrs = () => {
    const attrs: Record<string, string> = {}
    skipWs()
    while (i < source.length && /[:A-Za-z_]/.test(source[i]!)) {
      const name = parseName()
      skipWs()
      if (source[i] !== '=') {
        break
      }
      i++
      skipWs()
      const quote = source[i]
      i++
      const start = i
      while (i < source.length && source[i] !== quote) i++
      attrs[name] = decodeXml(source.slice(start, i))
      i++
      skipWs()
    }
    return attrs
  }
  const parseNode = (): XmlNode => {
    skipWs()
    if (starts('<?')) {
      const end = source.indexOf('?>', i)
      i = end + 2
      return parseNode()
    }
    if (source[i] !== '<') {
      throw new Error(`expected tag at ${i}`)
    }
    i++
    const name = parseName()
    const attrs = parseAttrs()
    if (starts('/>')) {
      i += 2
      return { name, attrs, children: [], text: '' }
    }
    if (source[i] !== '>') {
      throw new Error(`expected > at ${i}`)
    }
    i++
    const children: XmlNode[] = []
    let text = ''
    while (i < source.length) {
      if (starts('<![CDATA[')) {
        i += 9
        const end = source.indexOf(']]>', i)
        text += source.slice(i, end)
        i = end + 3
        continue
      }
      if (starts('</')) {
        i += 2
        parseName()
        skipWs()
        i++
        break
      }
      if (source[i] === '<') {
        children.push(parseNode())
        continue
      }
      const start = i
      while (i < source.length && source[i] !== '<') i++
      text += decodeXml(source.slice(start, i))
    }
    return { name, attrs, children, text }
  }
  return parseNode()
}

const localNameOf = (name: string) => {
  const colon = name.lastIndexOf(':')
  return colon === -1 ? name : name.slice(colon + 1)
}

const collect = (node: XmlNode, localName: string): XmlNode[] => [
  ...(localNameOf(node.name) === localName ? [node] : []),
  ...node.children.flatMap((child) => collect(child, localName))
]

const childText = (node: XmlNode, localName: string) => {
  const child = node.children.find(
    (item) => localNameOf(item.name) === localName
  )
  return child ? child.text.trim() : null
}

const selfHref = (body: string) =>
  collect(parseXml(body), 'link').find((node) => node.attrs.rel === 'self')
    ?.attrs.href ?? null

const copyrightYear = (body: string, tag: 'copyright' | 'rights') => {
  const text = collect(parseXml(body), tag)[0]?.text ?? ''
  return text.match(/© (\d{4})/)?.[1]
}

describe('parseFeedFile', () => {
  it('parses topic.xml', () => {
    expect(parseFeedFile('topic.xml')).toEqual({ name: 'topic', format: 'xml' })
  })

  it('parses topic.atom', () => {
    expect(parseFeedFile('topic.atom')).toEqual({
      name: 'topic',
      format: 'atom'
    })
  })

  it('parses topic.json', () => {
    expect(parseFeedFile('topic.json')).toEqual({
      name: 'topic',
      format: 'json'
    })
  })

  it('returns null for topic.html', () => {
    expect(parseFeedFile('topic.html')).toBeNull()
  })

  it('returns null for topic with no extension', () => {
    expect(parseFeedFile('topic')).toBeNull()
  })

  it('splits on the last dot', () => {
    expect(parseFeedFile('a.b.xml')).toEqual({ name: 'a.b', format: 'xml' })
  })

  it('returns null for an empty string', () => {
    expect(parseFeedFile('')).toBeNull()
  })
})

describe('renderKunFeed', () => {
  it('returns the RSS content type, self link, items in order, and newest updated', () => {
    const rendered = renderKunFeed(
      channel(sampleItems()),
      'xml',
      baseUrl,
      links
    )
    expect(rendered.contentType).toBe('application/xml; charset=utf-8')
    expect(selfHref(rendered.body)).toBe(links.rss)
    const tree = parseXml(rendered.body)
    const items = collect(tree, 'item')
    expect(items.map((item) => childText(item, 'guid'))).toEqual([
      'https://www.kungal.com/topic/1',
      'https://www.kungal.com/topic/2'
    ])
    expect(collect(tree, 'guid').map((node) => node.attrs.isPermaLink)).toEqual(
      ['true', 'true']
    )
    expect(items.map((item) => childText(item, 'title'))).toEqual([
      'First',
      'Second'
    ])
    expect(items.map((item) => childText(item, 'pubDate'))).toEqual([
      older.toUTCString(),
      newer.toUTCString()
    ])
    expect(collect(tree, 'lastBuildDate')[0]?.text).toBe(newer.toUTCString())
  })

  it('returns the Atom content type, self link, items in order, and newest updated', () => {
    const rendered = renderKunFeed(
      channel(sampleItems()),
      'atom',
      baseUrl,
      links
    )
    expect(rendered.contentType).toBe('application/atom+xml; charset=utf-8')
    expect(selfHref(rendered.body)).toBe(links.atom)
    const tree = parseXml(rendered.body)
    const entries = collect(tree, 'entry')
    expect(entries.map((entry) => childText(entry, 'id'))).toEqual([
      'https://www.kungal.com/topic/1',
      'https://www.kungal.com/topic/2'
    ])
    expect(entries.map((entry) => childText(entry, 'title'))).toEqual([
      'First',
      'Second'
    ])
    expect(entries.map((entry) => childText(entry, 'updated'))).toEqual([
      older.toISOString(),
      newer.toISOString()
    ])
    expect(childText(tree, 'updated')).toBe(newer.toISOString())
    expect(childText(tree, 'icon')).toBe(`${baseUrl}/favicon.ico`)
  })

  it('returns the JSON Feed content type, self link, items in order, and newest date', () => {
    const rendered = renderKunFeed(
      channel(sampleItems()),
      'json',
      baseUrl,
      links
    )
    expect(rendered.contentType).toBe('application/feed+json; charset=utf-8')
    const body = JSON.parse(rendered.body) as {
      feed_url: string
      items: {
        id: string
        url: string
        title: string
        date_modified: string
        date_published: string
      }[]
    }
    expect(body.feed_url).toBe(links.json)
    expect(body.items.map((item) => item.id)).toEqual([
      'https://www.kungal.com/topic/1',
      'https://www.kungal.com/topic/2'
    ])
    expect(body.items.map((item) => item.title)).toEqual(['First', 'Second'])
    expect(body.items.map((item) => item.url)).toEqual(
      body.items.map((item) => item.id)
    )
    expect(body.items.map((item) => item.date_modified)).toEqual([
      older.toISOString(),
      newer.toISOString()
    ])
    expect(body.items.map((item) => item.date_published)).toEqual([
      older.toISOString(),
      newer.toISOString()
    ])
  })

  it('renders a valid empty channel in all three formats', () => {
    const empty = channel([])
    const xml = parseXml(renderKunFeed(empty, 'xml', baseUrl, links).body)
    expect(collect(xml, 'item')).toHaveLength(0)
    const atom = parseXml(renderKunFeed(empty, 'atom', baseUrl, links).body)
    expect(collect(atom, 'entry')).toHaveLength(0)
    const json = JSON.parse(
      renderKunFeed(empty, 'json', baseUrl, links).body
    ) as {
      feed_url: string
      items: unknown[]
    }
    expect(json.feed_url).toBe(links.json)
    expect(json.items).toEqual([])
  })

  it('uses the current year in the copyright', () => {
    const year = String(new Date().getFullYear())
    const xml = renderKunFeed(channel([]), 'xml', baseUrl, links)
    const atom = renderKunFeed(channel([]), 'atom', baseUrl, links)
    expect(copyrightYear(xml.body, 'copyright')).toBe(year)
    expect(copyrightYear(atom.body, 'rights')).toBe(year)
  })
})
