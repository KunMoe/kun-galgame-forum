import {
  ENTITY_PATHS,
  type EntityLinkRef,
  type EntityRefType
} from '~/utils/entityUrl'

const TYPES = new Set<string>(Object.keys(ENTITY_PATHS))

export const isEntityRefType = (type: string): type is EntityRefType =>
  TYPES.has(type)

export const asEntityLinkRef = (ref: {
  type: string
  id: string
  url: string
}): EntityLinkRef | null =>
  isEntityRefType(ref.type) && /^[1-9][0-9]*$/.test(ref.id)
    ? { type: ref.type, id: ref.id, url: ref.url }
    : null
