import { createHmac } from 'node:crypto'
import { kungal } from '../../app/config/kungal'
import { KUN_GALGAME_RESOURCE_TYPE_MAP } from '../../app/constants/galgame'
import { KUN_GALGAME_OFFICIAL_CATEGORY_MAP } from '../../app/constants/galgameOfficial'
import { KUN_TOPIC_SECTION } from '../../app/constants/topic'
import { deletedUserName } from '#shared/utils/deletedUser'
import type { GalgameCharacterDetail } from '../../shared/types/galgame-character'
import type { GalgameDetail } from '../../shared/types/galgame'
import type { GalgameOfficialDetail } from '../../shared/types/galgame-official'
import type { GalgameStaffDetail } from '../../shared/types/galgame-staff'
import type { Topic } from '../../shared/utils/api/schemas'
import { createApiClient } from '../../shared/utils/api/client'
import { settle } from '../../shared/utils/api/problem'
import { documentPlainText } from '../../shared/utils/content/plainText'
import { truncateRunes } from '../../shared/utils/format'
import { getEffectivePortrait } from '../../shared/utils/getEffectiveBanner'
import { markdownToText } from '../../shared/utils/markdownToText'
import type { KunOgCardKind } from '../../shared/utils/ogCard'
import { fetchKunApi } from './kunFeed'

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
  const game = await fetchKunApi<GalgameDetail>(`/galgame/${id}`, {
    galgame_id: id
  })
  if (
    !game ||
    game.content_limit === 'nsfw' ||
    game.indexed === false
  ) {
    return null
  }
  const rated = !!game.rating_count && game.rating !== undefined
  return {
    template: 'work',
    fields: {
      title: text(game.name, 200),
      originalName:
        game.name_original === game.name
          ? undefined
          : text(game.name_original, 200),
      cover: getEffectivePortrait(game) || undefined,
      label: text(game.official[0]?.name, 80),
      releaseDate: game.release_date_tba
        ? undefined
        : text(game.release_date, 40),
      rating: rated ? game.rating : undefined,
      ratingCount: rated ? game.rating_count : undefined,
      badges: [
        ...game.type
          .slice(0, 2)
          .map((type) => KUN_GALGAME_RESOURCE_TYPE_MAP[type] || type),
        ...(game.age_limit === 'r18' ? ['18+'] : [])
      ]
        .map((badge) => truncateRunes(badge, 24))
        .slice(0, 4)
    }
  }
}

const buildCharacter = async (id: number): Promise<KunOgCard | null> => {
  const character = await fetchKunApi<GalgameCharacterDetail>(
    `/galgame-character/${id}`,
    { limit: 1 }
  )
  if (!character || character.moved_to) {
    return null
  }
  const work = character.works[0]
  const original = character.name_original || character.latin
  return {
    template: 'character',
    fields: {
      name: text(character.name, 120),
      originalName:
        original === character.name ? undefined : text(original, 120),
      portrait: character.figure || character.image || undefined,
      work: text(work?.name, 200),
      voice: work?.voices[0]?.name
        ? `CV. ${truncateRunes(work.voices[0].name, 74)}`
        : undefined
    }
  }
}

const buildStaff = async (id: number): Promise<KunOgCard | null> => {
  const staff = await fetchKunApi<GalgameStaffDetail>(`/galgame-staff/${id}`, {
    limit: 3
  })
  if (!staff || staff.moved_to) {
    return null
  }
  const original = staff.name_original || staff.latin
  return {
    template: 'person',
    fields: {
      name: text(staff.name, 120),
      originalName: original === staff.name ? undefined : text(original, 120),
      photo: staff.photo || undefined,
      works: staff.works
        .slice(0, 3)
        .map((work) => truncateRunes(work.name, 120)),
      badges: staff.roles.slice(0, 4).map((role) => truncateRunes(role, 24))
    }
  }
}

const buildOfficial = async (id: number): Promise<KunOgCard | null> => {
  const official = await fetchKunApi<GalgameOfficialDetail>(
    `/galgame-official/${id}`,
    { limit: 1 }
  )
  if (!official || official.moved_to) {
    return null
  }
  const category = KUN_GALGAME_OFFICIAL_CATEGORY_MAP[official.category]
  return {
    template: 'label',
    fields: {
      name: text(official.name, 120),
      originalName:
        official.original === official.name
          ? undefined
          : text(official.original, 120),
      logo: official.logo || undefined,
      workCount: official.galgame_count || undefined,
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
