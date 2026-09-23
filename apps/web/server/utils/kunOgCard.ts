import { createHmac } from 'node:crypto'
import { kungal } from '../../app/config/kungal'
import { KUN_GALGAME_OFFICIAL_CATEGORY_MAP } from '../../app/constants/galgameOfficial'
import { KUN_TOPIC_SECTION } from '../../app/constants/topic'
import { deletedUserName } from '#shared/utils/deletedUser'
import type { Topic } from '../../shared/utils/api/schemas'
import { createApiClient } from '../../shared/utils/api/client'
import {
  catalogEntityName,
  type CatalogName
} from '../../shared/utils/catalogName'
import { settle } from '../../shared/utils/api/problem'
import { documentPlainText } from '../../shared/utils/content/plainText'
import { truncateRunes } from '../../shared/utils/format'
import type { KunOgCardKind } from '../../shared/utils/ogCard'
import { resourceTypeLabel } from '../../shared/utils/galgameResourceVocab'

export interface KunOgCard {
  template: string
  fields: Record<string, unknown>
}

const text = (value: string | null | undefined, max: number) => {
  const trimmed = (value ?? '').trim()
  return trimmed ? truncateRunes(trimmed, max) : undefined
}

/**
 * The renderer signs `${template}\n${d}` and rejects anything else, so the template name in the
 * path is part of the message — signing `d` alone would let one card URL be replayed against
 * every other template.
 */
export const kunOgSignedUrl = (card: KunOgCard): string | null => {
  const { ogSiteKey, ogBaseUrl } = useRuntimeConfig()
  if (!ogSiteKey) {
    return null
  }
  const d = Buffer.from(JSON.stringify(card.fields), 'utf8').toString(
    'base64url'
  )
  const sig = createHmac('sha256', ogSiteKey)
    .update(`${card.template}\n${d}`)
    .digest('base64url')
  // eslint-disable-next-line no-restricted-syntax -- the OG card service's API, not the forum's
  return `${ogBaseUrl}/v1/og/${card.template}?d=${d}&sig=${sig}`
}

export const kunOgSiteCard = (origin: string): KunOgCard => ({
  template: 'site',
  fields: {
    name: kungal.titleShort,
    slogan: '世界上最萌的 Galgame 论坛 · 资源资料库 · 永远免费',
    logo: `${origin}/kungalgame.webp`
  }
})

export const topicToOgCard = (topic: Topic): KunOgCard | null => {
  if (topic.is_nsfw || topic.state === 'hidden') {
    return null
  }
  const section = topic.sections[0]
  return {
    template: 'topic',
    fields: {
      title: text(topic.title, 200),
      excerpt: text(documentPlainText(topic.content, deletedUserName), 120),
      section: section ? KUN_TOPIC_SECTION[section] : undefined,
      author: text(topic.author.name ?? deletedUserName, 80),
      authorAvatar: topic.author.avatar?.url || undefined,
      views: topic.view_count,
      replies: topic.reply_count,
      likes: topic.like_count
    }
  }
}

export const fetchTopicOgCard = async (
  api: ReturnType<typeof createApiClient>,
  topicId: string
): Promise<KunOgCard | null> => {
  const result = await settle(
    api.GET('/topics/{topic_id}', {
      params: { path: { topic_id: topicId } }
    })
  )
  if (!result.ok) {
    return null
  }
  return topicToOgCard(result.data)
}

const buildTopic = async (id: number): Promise<KunOgCard | null> => {
  const api = createApiClient({
    origin: useRuntimeConfig().apiBaseUrl,
    timeoutMs: 10000
  })
  return fetchTopicOgCard(api, String(id))
}

const buildGalgame = async (id: number): Promise<KunOgCard | null> => {
  const api = createApiClient({
    origin: useRuntimeConfig().apiBaseUrl,
    timeoutMs: 10000
  })
  const result = await settle(
    api.GET('/works/{work_id}', {
      params: { path: { work_id: String(id) } }
    })
  )
  if (!result.ok) {
    return null
  }
  const game = result.data
  if (game.is_nsfw || !game.is_published) {
    return null
  }
  const rated = game.rating_count > 0 && game.rating_score != null
  const { name, original } = catalogEntityName(game)
  const company = game.companies[0]
  return {
    template: 'work',
    fields: {
      title: text(name, 200),
      originalName: original ? text(original, 200) : undefined,
      cover: game.cover?.url || undefined,
      label: company ? text(catalogEntityName(company).name, 80) : undefined,
      releaseDate: text(game.release_date, 40),
      rating: rated ? game.rating_score : undefined,
      ratingCount: rated ? game.rating_count : undefined,
      badges: [
        ...game.resource_types
          .slice(0, 2)
          .map((type) => resourceTypeLabel(type)),
        ...(game.content_rating === 'r18' ? ['18+'] : [])
      ]
        .map((badge) => truncateRunes(badge, 24))
        .slice(0, 4)
    }
  }
}

const catalogApi = () =>
  createApiClient({ origin: useRuntimeConfig().apiBaseUrl, timeoutMs: 8000 })

const originalOf = (n: CatalogName): string | undefined => {
  const { name, original } = catalogEntityName(n)
  const other = original || n.latin || ''
  return other && other !== name ? text(other, 120) : undefined
}

const buildCharacter = async (id: number): Promise<KunOgCard | null> => {
  const api = catalogApi()
  const path = { character_id: String(id) }
  const [character, appearances] = await Promise.all([
    settle(api.GET('/characters/{character_id}', { params: { path } })),
    settle(
      api.GET('/characters/{character_id}/appearances', {
        params: { path, query: { limit: 1 } }
      })
    )
  ])
  if (!character.ok) {
    return null
  }
  const c = character.data
  const first = appearances.ok ? appearances.data.items[0] : undefined
  const voice = first?.voices[0]
  return {
    template: 'character',
    fields: {
      name: text(catalogEntityName(c).name, 120),
      originalName: originalOf(c),
      portrait: c.figure?.url || c.image?.url || undefined,
      work: first ? text(catalogEntityName(first.work_summary).name, 200) : undefined,
      voice: voice
        ? `CV. ${truncateRunes(catalogEntityName(voice).name, 74)}`
        : undefined
    }
  }
}

const buildStaff = async (id: number): Promise<KunOgCard | null> => {
  const api = catalogApi()
  const path = { credit_name_id: String(id) }
  const [person, credits] = await Promise.all([
    settle(api.GET('/credit-names/{credit_name_id}', { params: { path } })),
    settle(
      api.GET('/credit-names/{credit_name_id}/credits', {
        params: { path, query: { limit: 3 } }
      })
    )
  ])
  if (!person.ok) {
    return null
  }
  const items = credits.ok ? credits.data.items : []
  const roles = [
    ...new Set(items.flatMap((c) => c.credit_roles.map((r) => r.display_name)))
  ]
  return {
    template: 'person',
    fields: {
      name: text(catalogEntityName(person.data).name, 120),
      originalName: originalOf(person.data),
      photo: person.data.photo?.url || undefined,
      works: items.map((c) =>
        truncateRunes(catalogEntityName(c.work_summary).name, 120)
      ),
      badges: roles.slice(0, 4).map((role) => truncateRunes(role, 24))
    }
  }
}

const buildOfficial = async (id: number): Promise<KunOgCard | null> => {
  const company = await settle(
    catalogApi().GET('/companies/{company_id}', {
      params: { path: { company_id: String(id) } }
    })
  )
  if (!company.ok) {
    return null
  }
  const c = company.data
  const category = KUN_GALGAME_OFFICIAL_CATEGORY_MAP[c.company_kind]
  return {
    template: 'label',
    fields: {
      name: text(catalogEntityName(c).name, 120),
      originalName: originalOf(c),
      logo: c.logo?.url || undefined,
      workCount: c.catalog_work_count || undefined,
      badges: category ? [truncateRunes(category, 24)] : []
    }
  }
}

const builders: Record<
  KunOgCardKind,
  (id: number) => Promise<KunOgCard | null>
> = {
  topic: buildTopic,
  galgame: buildGalgame,
  character: buildCharacter,
  staff: buildStaff,
  official: buildOfficial
}

/**
 * Never throws: a crawler asking for a card must get an image, so every failure degrades to the
 * site card rather than to a 500 with no og:image at all.
 */
export const kunOgEntityCard = async (
  kind: KunOgCardKind,
  id: number
): Promise<KunOgCard | null> => {
  try {
    return await builders[kind](id)
  } catch (error) {
    console.warn(`[og] ${kind}/${id} card failed:`, (error as Error).message)
    return null
  }
}
