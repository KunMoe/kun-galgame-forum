import { kungal } from '~/config/kungal'

export type EntityRefType =
  | 'topic'
  | 'work'
  | 'galgame_resource'
  | 'user'
  | 'character'
  | 'company'
  | 'tag'
  | 'series'
  | 'credit_name'
  | 'rating'

export interface EntityLinkRef {
  type: EntityRefType
  id: string
  url: string
}

export const ENTITY_PATHS: Record<EntityRefType, string> = {
  topic: 'topic',
  work: 'galgame',
  galgame_resource: 'galgame/resource',
  user: 'user',
  character: 'galgame/character',
  company: 'galgame/official',
  tag: 'galgame/tag',
  series: 'galgame/series',
  credit_name: 'galgame/staff',
  rating: 'galgame-rating'
}

const PATH_TYPES = Object.fromEntries(
  Object.entries(ENTITY_PATHS).map(([type, path]) => [path, type])
) as Record<string, EntityRefType>

const ENTITY_PATH =
  /^\/(topic|galgame\/(?:resource|character|official|tag|series|staff)|galgame-rating|galgame|user)\/([1-9][0-9]{0,18})\/?$/

const siteApex = new URL(kungal.domain.main).host.replace(/^www\./, '')

export const entityUrl = (type: EntityRefType, id: string): string =>
  `${kungal.domain.main}/${ENTITY_PATHS[type]}/${id}`

export const entityFromUrl = (raw: string): EntityLinkRef | null => {
  let url: URL
  try {
    url = new URL(raw, kungal.domain.main)
  } catch {
    return null
  }
  const absolute = /^[a-z][a-z0-9+.-]*:/i.test(raw)
  if (!absolute && (!raw.startsWith('/') || raw.startsWith('//'))) {
    return null
  }
  if (absolute && /^[a-z]+:\/\/[^/]*:\d/i.test(raw)) {
    return null
  }
  if (
    (url.protocol !== 'https:' && url.protocol !== 'http:') ||
    (url.host !== siteApex && url.host !== `www.${siteApex}`) ||
    url.search !== '' ||
    url.hash !== '' ||
    raw.includes('?') ||
    raw.includes('#') ||
    url.username !== '' ||
    url.password !== ''
  ) {
    return null
  }
  const match = ENTITY_PATH.exec(url.pathname)
  if (!match) {
    return null
  }
  const type = PATH_TYPES[match[1]!]!
  const id = match[2]!
  return { type, id, url: entityUrl(type, id) }
}
