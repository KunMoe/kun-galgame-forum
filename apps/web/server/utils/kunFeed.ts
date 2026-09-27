import { Feed } from 'feed'
import type { KunFeedFormat } from '../../shared/utils/feedUrl'

export type KunFeedItem = {
  link: string
  title: string
  date: Date
  description: string
  author: { name: string; link?: string }[]
  category?: { name: string }[]
  image?: string
}

export type KunFeedChannel = {
  title: string
  description: string
  link: string
  image: string
  items: KunFeedItem[]
}

export type KunFeedResult =
  | { ok: true; channel: KunFeedChannel }
  | { ok: false; status: 404 | 502 }

const FORMATS = new Set<KunFeedFormat>(['xml', 'atom', 'json'])

const CONTENT_TYPE: Record<KunFeedFormat, string> = {
  xml: 'application/xml; charset=utf-8',
  atom: 'application/atom+xml; charset=utf-8',
  json: 'application/feed+json; charset=utf-8'
}

export const parseFeedFile = (
  file: string
): { name: string; format: KunFeedFormat } | null => {
  const lastDot = file.lastIndexOf('.')
  if (lastDot <= 0) {
    return null
  }
  const name = file.slice(0, lastDot)
  const ext = file.slice(lastDot + 1)
  if (!name || !FORMATS.has(ext as KunFeedFormat)) {
    return null
  }
  return { name, format: ext as KunFeedFormat }
}

const newestDate = (items: KunFeedItem[]): Date => {
  let latest: Date | undefined
  for (const item of items) {
    if (!latest || item.date > latest) {
      latest = item.date
    }
  }
  return latest ?? new Date()
}

export const renderKunFeed = (
  channel: KunFeedChannel,
  format: KunFeedFormat,
  baseUrl: string,
  links: { rss: string; atom: string; json: string }
): { body: string; contentType: string } => {
  const feed = new Feed({
    id: channel.link,
    title: channel.title,
    description: channel.description,
    link: channel.link,
    language: 'zh-CN',
    image: channel.image,
    favicon: `${baseUrl}/favicon.ico`,
    generator: '萌萌 RSS 生成器',
    copyright: `版权所有 © ${new Date().getFullYear()} 鲲 Galgame 保留所有权利`,
    updated: newestDate(channel.items),
    feedLinks: links
  })
  for (const item of channel.items) {
    feed.addItem({
      // rss2 marks any id as isPermaLink="false" and json1 has no fallback
      // for a missing id, so only JSON Feed gets one.
      ...(format === 'json' ? { id: item.link } : {}),
      link: item.link,
      title: item.title,
      date: item.date,
      published: item.date,
      description: item.description,
      author: item.author,
      ...(item.category ? { category: item.category } : {}),
      ...(item.image ? { image: item.image } : {})
    })
  }
  const body =
    format === 'xml'
      ? feed.rss2()
      : format === 'atom'
        ? feed.atom1()
        : feed.json1()
  return { body, contentType: CONTENT_TYPE[format] }
}
