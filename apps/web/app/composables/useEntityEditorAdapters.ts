import type { KunEditorAdapters } from '@kungal/editor-core'
import {
  ENTITY_KINDS,
  entityId,
  entityKind,
  entityTitle
} from '~/utils/entityTarget'
import { entityFromUrl, entityUrl, type EntityRefType } from '~/utils/entityUrl'
import { asEntityLinkRef, isEntityRefType } from '~/utils/entityRefType'

export const ENTITY_MENU_TYPES: EntityRefType[] = [
  'work',
  'character',
  'company',
  'tag',
  'series',
  'credit_name',
  'rating',
  'topic',
  'galgame_resource'
]

export const useEntityEditorAdapters = (): Partial<KunEditorAdapters> => {
  const { searchIds, previews } = useEntitySearch()
  const { pick } = useEntityPicker()
  const { showKUNGalgamePreferOriginalName } = storeToRefs(
    usePersistSettingsStore()
  )
  const titleOf = (target: Parameters<typeof entityTitle>[0]) =>
    entityTitle(target, showKUNGalgamePreferOriginalName.value)

  return {
    entityFromUrl,
    entityTypes: ENTITY_MENU_TYPES.map((type) => ({
      type,
      label: `${ENTITY_KINDS[type].noun}卡片`,
      icon: ENTITY_KINDS[type].icon,
      command: ENTITY_KINDS[type].command,
      description: `搜索并插入${ENTITY_KINDS[type].noun}`
    })),
    searchEntities: async (type, query, signal) => {
      if (!isEntityRefType(type)) {
        return []
      }
      const ids = await searchIds(type, query, signal)
      const targets = await previews(
        ids.map((id) => ({ type, id })),
        signal
      )
      return targets
        .filter((t) => t.object !== 'unavailable_entity')
        .map((t) => {
          const kind = entityKind(t)
          const id = entityId(t)
          return {
            ref: { type: kind, id, url: entityUrl(kind, id) },
            title: titleOf(t)
          }
        })
    },
    entityTitle: async (ref, signal) => {
      const known = asEntityLinkRef(ref)
      if (!known) {
        return null
      }
      const [target] = await previews([known], signal)
      return target && target.object !== 'unavailable_entity'
        ? titleOf(target)
        : null
    },
    entityPrompt: ({ type, display }) =>
      isEntityRefType(type)
        ? pick(type, display === 'chip' ? 'chip' : 'card')
        : Promise.resolve(null)
  }
}
