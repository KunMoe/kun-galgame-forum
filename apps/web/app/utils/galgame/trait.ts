import type { TraitRef, TraitSummary } from '#shared/utils/api/schemas'
import { catalogVocabularyName } from '#shared/utils/catalogName'

// Engages in, Subject of and their (Sexual) twins file the same act on each
// side under one name: 2279 and 2312 both read 女同强奸.
const PAIRED_GROUP_IDS = new Set(['2625', '2689', '2709', '625'])

export const traitSide = (t: TraitRef) =>
  PAIRED_GROUP_IDS.has(t.trait_group_id) && t.id !== t.trait_group_id
    ? catalogVocabularyName(t.trait_group)
    : ''

export const traitFullName = (t: TraitRef) => {
  const name = catalogVocabularyName(t)
  const side = traitSide(t)
  return side ? `${name} · ${side}` : name
}

export const traitContext = (t: TraitSummary) => {
  if (!t.parents.length) {
    return ''
  }
  const group = catalogVocabularyName(t.trait_group)
  const parent = catalogVocabularyName(t.parents[0]!)
  return parent === group ? group : `${group} › ${parent}`
}

// Level-one traits under `rootId` with the level-two traits that hang from each,
// as a Trait's subtraits list both levels flat.
export const traitSections = (rootId: string, subtraits: TraitSummary[]) => {
  const top = subtraits.filter((t) => t.parents.some((p) => p.id === rootId))
  const topIds = new Set(top.map((t) => t.id))
  return top.map((trait) => ({
    trait,
    children: subtraits.filter(
      (t) => !topIds.has(t.id) && t.parents.some((p) => p.id === trait.id)
    )
  }))
}
