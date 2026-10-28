import type { EntityTarget, Image } from '#shared/utils/api/schemas'
import {
  resourceLanguageLabel,
  resourcePlatformLabel,
  resourceTypeLabel
} from '#shared/utils/galgameResourceVocab'
import { KUN_GALGAME_OFFICIAL_CATEGORY_MAP } from '~/constants/galgameOfficial'
import { KUN_GALGAME_TAG_CATEGORY_MAP } from '~/constants/galgameTag'
import { catalogNameText } from '~/utils/catalogName'
import {
  ENTITY_KINDS,
  entityHref,
  entityKind,
  entityTitle
} from '~/utils/entityTarget'
import type { EntityRefType } from '~/utils/entityUrl'
import { toKunUser } from '~/utils/userRef'

type Shape = 'landscape' | 'portrait' | 'logo' | 'photo'

export type EntityCardMedia =
  | { kind: 'image'; image: Image; shape: Shape; isBlurred: boolean }
  | { kind: 'stack'; images: { image: Image; isBlurred: boolean }[] }
  | { kind: 'avatar'; user: KunUser }
  | { kind: 'icon'; icon: string }

export interface EntityCardBadge {
  label: string
  tone: 'danger' | 'warning' | 'primary'
}

export type EntityCardLayout = 'full' | 'compact' | 'pill'

export interface EntityCardModel {
  kind: EntityRefType
  layout: EntityCardLayout
  href: string
  noun: string
  icon: string
  title: string
  subtitle: string
  description: string
  chips: string[]
  info: string
  person: KunUser | null
  score: number | null
  badges: EntityCardBadge[]
  media: EntityCardMedia
  isUnavailable: boolean
}

type Stance = 'hide' | 'blur' | 'show'
type Named = Parameters<typeof catalogNameText>[0]

const sensitiveImage = (image: Image | null | undefined) =>
  image?.sexual === 'suggestive' || image?.sexual === 'explicit'

const isNsfw = (t: EntityTarget): boolean => {
  switch (t.object) {
    case 'topic':
    case 'work':
      return t.is_nsfw
    case 'galgame_resource':
    case 'rating':
      return t.work?.is_nsfw ?? false
    case 'tag':
      return t.is_sexual
    default:
      return false
  }
}

const isLandscape = (image: Image) =>
  !!image.width && !!image.height && image.width > image.height * 1.15

const coverOf = (t: EntityTarget): { image: Image; shape: Shape } | null => {
  const pick = (image: Image | null | undefined, shape: Shape) =>
    image
      ? {
          image,
          shape:
            shape === 'portrait' && isLandscape(image) ? 'landscape' : shape
        }
      : null
  switch (t.object) {
    case 'topic':
      return pick(t.cover_images[0], 'landscape')
    case 'work':
      return pick(t.cover, 'portrait')
    case 'galgame_resource':
    case 'rating':
      return pick(t.work?.cover, 'portrait')
    case 'character':
      return pick(t.image, 'portrait')
    case 'company':
      return pick(t.logo, 'logo')
    case 'credit_name':
      return pick(t.photo, 'photo')
    default:
      return null
  }
}

const year = (date: string | null) => (date ? `${date.slice(0, 4)} 年` : '')

const quoted = (names: string[]) => names.map((n) => `《${n}》`).join('')

const otherName = (t: Named, preferOriginal: boolean) => {
  const shown = catalogNameText(t, preferOriginal)
  const other = catalogNameText(t, !preferOriginal)
  return other !== shown ? other : (t.latin ?? '')
}

type Body = Pick<
  EntityCardModel,
  'title' | 'subtitle' | 'description' | 'chips' | 'info' | 'person' | 'score'
>

const bodyOf = (t: EntityTarget, preferOriginal: boolean): Body => {
  const name = (n: Named) => catalogNameText(n, preferOriginal)
  const count = (n: number, prefix = '') => (n ? `${prefix}${n} 部作品` : '')
  const body: Body = {
    title: entityTitle(t, preferOriginal),
    subtitle: '',
    description: '',
    chips: [],
    info: '',
    person: null,
    score: null
  }
  switch (t.object) {
    case 'work':
      body.subtitle = otherName(t, preferOriginal)
      body.info = [t.maker ? name(t.maker) : '', year(t.release_date)]
        .filter(Boolean)
        .join(' · ')
      body.chips = t.top_tags.map(name)
      break
    case 'character':
      body.subtitle = otherName(t, preferOriginal)
      body.info = t.works.length ? `出自 ${quoted(t.works.map(name))}` : ''
      break
    case 'credit_name':
      body.subtitle = otherName(t, preferOriginal)
      body.chips = [...t.role_names]
      body.info = t.works.length ? `代表作 ${quoted(t.works.map(name))}` : ''
      break
    case 'company':
      body.subtitle = otherName(t, preferOriginal)
      body.chips = [KUN_GALGAME_OFFICIAL_CATEGORY_MAP[t.company_kind] ?? '']
      body.info = count(t.catalog_work_count)
      break
    case 'tag':
      body.chips = [KUN_GALGAME_TAG_CATEGORY_MAP[t.tag_kind] ?? '']
      body.description = t.description
      body.info = count(t.catalog_work_count)
      break
    case 'series':
      body.description = quoted(t.works.map(name))
      body.info = count(t.catalog_work_count, '共 ')
      break
    case 'rating':
      body.title = t.work ? name(t.work) : '评分'
      body.description = t.has_spoilers
        ? '这篇评分含有剧透，打开后查看'
        : t.short_summary || '作者没有写短评'
      body.person = toKunUser(t.author)
      body.score = t.overall
      break
    case 'topic':
      body.person = toKunUser(t.author)
      body.info = `${formatNumber(t.view_count)} 浏览 · ${t.reply_count} 回复 · ${t.like_count} 点赞`
      break
    case 'galgame_resource':
      body.title = t.work ? name(t.work) : resourceTypeLabel(t.resource_type)
      body.subtitle = t.title || resourceTypeLabel(t.resource_type)
      body.chips = [
        resourceTypeLabel(t.resource_type),
        ...t.resource_languages.map(resourceLanguageLabel),
        ...t.resource_platforms.map(resourcePlatformLabel)
      ]
      body.info = t.size
      body.person = toKunUser(t.author)
      break
    case 'unavailable_entity':
      body.subtitle = '已删除、已隐藏或仅部分用户可见'
      break
  }
  body.chips = body.chips.filter(Boolean)
  return body
}

const badgesOf = (t: EntityTarget): EntityCardBadge[] => {
  const out: EntityCardBadge[] = []
  if (isNsfw(t)) {
    out.push({ label: 'NSFW', tone: 'danger' })
  }
  if (t.object === 'galgame_resource' && t.state === 'expired') {
    out.push({ label: '链接失效', tone: 'danger' })
  }
  if (t.object === 'rating' && t.has_spoilers) {
    out.push({ label: '剧透', tone: 'warning' })
  }
  return out
}

const mediaOf = (t: EntityTarget, stance: Stance): EntityCardMedia => {
  if (t.object === 'user') {
    return { kind: 'avatar', user: toKunUser(t) }
  }
  if (t.object === 'series') {
    const images = t.works.flatMap((w) => {
      const sensitive = w.is_nsfw || sensitiveImage(w.cover)
      return w.cover && !(sensitive && stance === 'hide')
        ? [{ image: w.cover, isBlurred: sensitive && stance === 'blur' }]
        : []
    })
    if (images.length) {
      return { kind: 'stack', images }
    }
  }
  const cover = coverOf(t)
  const sensitive = isNsfw(t) || sensitiveImage(cover?.image)
  if (cover && !(sensitive && stance === 'hide')) {
    return {
      kind: 'image',
      ...cover,
      isBlurred: sensitive && stance === 'blur'
    }
  }
  return {
    kind: 'icon',
    icon:
      t.object === 'unavailable_entity'
        ? 'lucide:eye-off'
        : ENTITY_KINDS[entityKind(t)].icon
  }
}

export const entityCardModel = (
  t: EntityTarget,
  preferOriginal: boolean,
  stance: Stance
): EntityCardModel => {
  const kind = entityKind(t)
  const media = mediaOf(t, stance)
  const layout: EntityCardLayout =
    t.object === 'tag' || t.object === 'company'
      ? 'pill'
      : media.kind === 'icon'
        ? 'compact'
        : 'full'
  return {
    kind,
    layout,
    href: entityHref(t),
    noun: ENTITY_KINDS[kind].noun,
    icon: ENTITY_KINDS[kind].icon,
    ...bodyOf(t, preferOriginal),
    badges: badgesOf(t),
    media,
    isUnavailable: t.object === 'unavailable_entity'
  }
}
