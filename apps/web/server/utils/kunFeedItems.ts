import type { GalgameResource } from '../../shared/utils/api/schemas'
import { catalogEntityName } from '../../shared/utils/catalogName'
import { documentPlainText } from '../../shared/utils/content/plainText'
import { deletedUserName } from '../../shared/utils/deletedUser'
import { truncateRunes } from '../../shared/utils/format'
import {
  resourceLanguageLabel,
  resourcePlatformLabel,
  resourceRuntimeLabel,
  resourceTypeLabel,
  VERSION_LABEL_OPTIONS
} from '../../shared/utils/galgameResourceVocab'
import type { KunFeedItem } from './kunFeed'

export const FEED_DESCRIPTION_RUNES = 233

export const siteLogo = (baseUrl: string) => `${baseUrl}/kungalgame.webp`

export const authorLink = (baseUrl: string, userId: string) =>
  `${baseUrl}/user/${userId}/info`

export const authorEntry = (
  author: { id: string; name: string | null },
  baseUrl: string
) => ({
  name: author.name ?? deletedUserName,
  link: authorLink(baseUrl, author.id)
})

export const noteExcerpt = (content: GalgameResource['content']) =>
  truncateRunes(
    documentPlainText(content, deletedUserName),
    FEED_DESCRIPTION_RUNES
  )

export const workCoverUrl = (
  work: { cover: { url: string } | null; is_nsfw: boolean },
  includeNsfw: boolean
): string | undefined => {
  if (!work.cover) {
    return undefined
  }
  if (work.is_nsfw && !includeNsfw) {
    return undefined
  }
  return work.cover.url
}

const labeledList = (label: string, values: string[]): string | undefined =>
  values.length ? `${label}: ${values.join(' / ')}` : undefined

const versionPart = (
  versionLabel: GalgameResource['version_label']
): string | undefined => {
  if (!versionLabel) {
    return undefined
  }
  const option = VERSION_LABEL_OPTIONS.find(
    (item) => item.value === versionLabel
  )
  return `版本: ${option?.label ?? versionLabel}`
}

const resourceMeta = (resource: GalgameResource): string => {
  const size = resource.size.trim()
  const parts = [
    labeledList(
      '语言',
      resource.resource_languages.map((lang) => resourceLanguageLabel(lang))
    ),
    labeledList(
      '平台',
      resource.resource_platforms.map((platform) =>
        resourcePlatformLabel(platform)
      )
    ),
    labeledList(
      '运行环境',
      resource.resource_runtimes.map((runtime) => resourceRuntimeLabel(runtime))
    ),
    versionPart(resource.version_label),
    size ? `大小: ${size}` : undefined
  ].filter((part): part is string => Boolean(part))
  return parts.join(' | ')
}

export const resourceFeedItem = (
  baseUrl: string,
  resource: GalgameResource,
  includeNsfw: boolean
): KunFeedItem | null => {
  const work = resource.work
  if (!work) {
    return null
  }
  const workName = catalogEntityName(work).name
  const typeLabel = resourceTypeLabel(resource.resource_type)
  const extra = resource.title.trim()
  const title = extra
    ? `【${typeLabel}】${workName} · ${extra}`
    : `【${typeLabel}】${workName}`
  const meta = resourceMeta(resource)
  const excerpt = noteExcerpt(resource.content).trim()
  const description = [meta, excerpt].filter(Boolean).join('\n\n')
  const image = workCoverUrl(work, includeNsfw)
  return {
    link: `${baseUrl}/galgame/resource/${resource.id}`,
    title,
    date: new Date(resource.created_at),
    description,
    author: [authorEntry(resource.author, baseUrl)],
    category: [{ name: typeLabel }],
    ...(image ? { image } : {})
  }
}

export const topicFeedItem = (
  baseUrl: string,
  topic: { id: string; title: string; created_at: string },
  author: { id: string; name: string | null },
  description: string
): KunFeedItem => ({
  link: `${baseUrl}/topic/${topic.id}`,
  title: topic.title,
  date: new Date(topic.created_at),
  description,
  author: [authorEntry(author, baseUrl)]
})
