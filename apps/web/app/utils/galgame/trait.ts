import type { TraitSummary } from '#shared/utils/api/schemas'

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
