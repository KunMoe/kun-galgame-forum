import type { KunGalgameTagCategory } from '~/constants/galgameTag'
import type {
  CompanySummary,
  Engine,
  Character,
  SeriesSummary,
  TagSummary
} from '#shared/utils/api/schemas'
import type { CatalogName } from '#shared/utils/catalogName'

export const tagItemOf = (t: TagSummary): GalgameTagItem => ({
  id: Number(t.id),
  name: catalogVocabularyName(t),
  category: (t.is_sexual ? 'sexual' : t.tag_kind) as KunGalgameTagCategory,
  galgame_count: t.catalog_work_count
})

type NameOf = (n: CatalogName) => { name: string; original: string }

export const companyItemOf = (
  c: CompanySummary,
  nameOf: NameOf
): GalgameOfficialItem => ({
  id: Number(c.id),
  name: nameOf(c).name,
  link: '',
  category: c.company_kind as GalgameOfficialItem['category'],
  lang: '',
  alias: c.aliases,
  galgame_count: c.catalog_work_count,
  logo: c.logo?.url
})

export const engineItemOf = (e: Engine, nameOf: NameOf): GalgameEngineItem => ({
  id: Number(e.id),
  name: nameOf(e).name,
  description: e.description,
  alias: e.aliases,
  galgame_count: e.catalog_work_count
})

export const seriesCardOf = (
  s: SeriesSummary,
  nameOf: NameOf
): GalgameSeriesCard => ({
  id: Number(s.id),
  name: nameOf(s).name,
  is_nsfw: s.has_nsfw_works ?? s.sample_works.some((w) => w.is_nsfw),
  galgame_count: s.listed_work_count,
  catalog_galgame_count: s.catalog_work_count,
  sample_galgame: s.sample_works.map((w) => {
    const art = w.banner ?? w.cover
    return {
      name: nameOf(w).name,
      effective_banner_hash: art?.hash,
      effective_banner_url: art?.url,
      effective_banner_thumbhash: art?.thumbhash ?? undefined
    }
  })
})

const spoilerLevel = { none: 0, minor: 1, major: 2 } as const

export const characterViewOf = (c: Character, nameOf: NameOf) => {
  const { name, original } = nameOf(c)
  const meta = (img: Character['image']) =>
    img?.width && img.height
      ? { width: img.width, height: img.height, thumbhash: img.thumbhash ?? '' }
      : undefined
  return {
    id: Number(c.id),
    name,
    name_original: original,
    latin: c.latin ?? '',
    image: c.image?.url ?? '',
    figure: c.figure?.url ?? '',
    image_meta: meta(c.image),
    figure_meta: meta(c.figure),
    intro: pickCatalogIntro(c.intros)?.value ?? '',
    intros: c.intros.map((i) => ({
      lang: i.locale,
      intro: i.value,
      source: i.data_source ?? '',
      machine: i.is_machine
    })),
    traits: c.traits.map((t) => ({
      id: Number(t.id),
      name: catalogVocabularyName(t),
      group: catalogVocabularyName(t.trait_group),
      spoiler: spoilerLevel[t.spoiler_level],
      lie: t.is_lie
    })),
    links: c.links.map((l) => ({
      source: l.site,
      url: l.url,
      name: catalogLinkLabel(l.site, l.url)
    }))
  }
}

export type CharacterView = ReturnType<typeof characterViewOf>
