import type { EntityRefType } from '~/utils/entityUrl'

const ENTITY_TYPES: Record<string, EntityRefType> = {
  work: 'work',
  character: 'character',
  person: 'credit_name',
  company: 'company',
  series: 'series',
  tag: 'tag',
  'kungal:topic': 'topic'
}

const PAGES: Record<string, string> = {
  work: '/galgame',
  character: '/galgame/character',
  person: '/galgame/staff',
  company: '/galgame/official',
  series: '/galgame/series',
  tag: '/galgame/tag',
  'kungal:topic': '/topic'
}

const KIND_LABELS: Record<string, string> = {
  work: '作品',
  character: '角色',
  person: '人物',
  company: '会社',
  series: '系列',
  tag: '标签',
  'kungal:topic': '话题'
}

export const agentCardKey = (c: Pick<AgentEntityCard, 'kind' | 'id'>) =>
  `${c.kind}-${c.id}`

export const agentCardTitle = (c: AgentEntityCard) =>
  c.localized?.['zh-Hans']?.value ||
  c.localized?.zh?.value ||
  c.localized?.['zh-Hant']?.value ||
  c.title

export const agentCardKindLabel = (kind: string) => KIND_LABELS[kind] ?? ''

export const agentCardIsWork = (c: Pick<AgentEntityCard, 'kind'>) =>
  c.kind === 'work'

export const agentCardEntity = (c: Pick<AgentEntityCard, 'kind' | 'id'>) => {
  const type = ENTITY_TYPES[c.kind]
  return type && /^[1-9]\d*$/.test(c.id) ? { type, id: c.id } : null
}

export const agentCardPreviewable = (c: AgentEntityCard) => !!agentCardEntity(c)

export const agentCardHref = (c: AgentEntityCard) => {
  const page = PAGES[c.kind]
  return page ? `${page}/${c.id}` : null
}

const RATING_SOURCES: Record<string, string> = {
  bangumi: 'Bangumi',
  vndb: 'VNDB'
}

const scoredRatings = (c: AgentEntityCard) =>
  (c.ratings ?? [])
    .filter((r) => r.score > 0 && r.source in RATING_SOURCES)
    .sort(
      (a, b) =>
        Object.keys(RATING_SOURCES).indexOf(a.source) -
        Object.keys(RATING_SOURCES).indexOf(b.source)
    )

export const agentCardRating = (c: AgentEntityCard) => {
  const r = scoredRatings(c).at(0)
  return r
    ? { label: RATING_SOURCES[r.source]!, score: r.score.toFixed(1) }
    : null
}

export const agentCardRatingsText = (c: AgentEntityCard) =>
  scoredRatings(c)
    .map(
      (r) =>
        `${RATING_SOURCES[r.source]} ${r.score.toFixed(1)}（${r.votes} 人）`
    )
    .join(' · ')

export const agentTagChips = (cards: AgentEntityCard[]) =>
  cards
    .filter((c) => c.kind === 'tag' && /^[1-9]\d*$/.test(c.id))
    .map((c) => ({
      id: c.id,
      name: agentCardTitle(c),
      href: `/galgame/tag/${c.id}`
    }))
