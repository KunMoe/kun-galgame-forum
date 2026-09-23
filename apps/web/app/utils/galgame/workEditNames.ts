import type { Work } from '#shared/utils/api/schemas'
import {
  catalogVocabularyName,
  type CatalogName
} from '#shared/utils/catalogName'
import type { GalgameEditNames } from '~/constants/galgameEdit'

type NameOf = (n: CatalogName) => { name: string; original: string }

const toMap = (
  items: Array<{ id: string } & CatalogName>,
  label: (n: CatalogName) => string
) => new Map(items.map((item) => [Number(item.id), label(item)]))

export const mapsFromWork = (
  work: Work | null | undefined,
  nameOf: NameOf
): GalgameEditNames => {
  const staff = new Map<number, string>()
  for (const group of work?.credits ?? []) {
    for (const person of group.people) {
      const id = Number(person.id)
      if (!staff.has(id)) {
        staff.set(id, nameOf(person).name)
      }
    }
  }
  const label = (n: CatalogName) => nameOf(n).name
  return {
    tag: toMap(work?.tags ?? [], catalogVocabularyName),
    official: toMap(work?.companies ?? [], label),
    engine: toMap(work?.engines ?? [], label),
    series: toMap(work?.series ?? [], label),
    character: toMap(work?.roster ?? [], label),
    staff,
    covers: work?.covers,
    screenshots: work?.screenshots
  }
}
