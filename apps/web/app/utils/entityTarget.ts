import type { EntityTarget } from '#shared/utils/api/schemas'
import { catalogNameText } from '~/utils/catalogName'
import { resourceTypeLabel } from '#shared/utils/galgameResourceVocab'
import { toKunUser } from '~/utils/userRef'
import { ENTITY_PATHS, type EntityRefType } from '~/utils/entityUrl'

interface EntityKindMeta {
  noun: string
  icon: string
  command: string
}

export const ENTITY_KINDS: Record<EntityRefType, EntityKindMeta> = {
  work: { noun: 'Galgame 条目', icon: 'lucide:gamepad-2', command: 'galgame' },
  character: {
    noun: 'Galgame 角色',
    icon: 'lucide:contact-round',
    command: 'character'
  },
  company: {
    noun: 'Galgame 厂商',
    icon: 'lucide:building-2',
    command: 'company'
  },
  tag: { noun: 'Galgame 标签', icon: 'lucide:tag', command: 'tag' },
  series: { noun: 'Galgame 系列', icon: 'lucide:library', command: 'series' },
  credit_name: {
    noun: 'Galgame 人物',
    icon: 'lucide:user-round-pen',
    command: 'staff'
  },
  rating: {
    noun: 'Galgame 评分',
    icon: 'lucide:star',
    command: 'galgame-rate'
  },
  topic: { noun: '话题', icon: 'lucide:messages-square', command: 'topic' },
  galgame_resource: {
    noun: 'Galgame 资源',
    icon: 'lucide:package',
    command: 'resource'
  },
  user: { noun: '用户', icon: 'lucide:user-round', command: 'user' }
}

export const ENTITY_ICON = Object.fromEntries(
  Object.entries(ENTITY_KINDS).map(([k, v]) => [k, v.icon])
) as Record<EntityRefType, string>

export const ENTITY_NOUN = Object.fromEntries(
  Object.entries(ENTITY_KINDS).map(([k, v]) => [k, v.noun])
) as Record<EntityRefType, string>

export const entityKind = (target: EntityTarget): EntityRefType =>
  target.object === 'unavailable_entity'
    ? target.entity_type
    : target.object === 'credit_name'
      ? 'credit_name'
      : target.object

export const entityId = (target: EntityTarget): string =>
  target.object === 'unavailable_entity' ? target.entity_id : target.id

export const entityHref = (target: EntityTarget): string =>
  `/${ENTITY_PATHS[entityKind(target)]}/${entityId(target)}`

export const entityTitle = (
  target: EntityTarget,
  preferOriginal: boolean
): string => {
  switch (target.object) {
    case 'topic':
      return target.title
    case 'work':
    case 'character':
    case 'company':
    case 'tag':
    case 'series':
    case 'credit_name':
      return catalogNameText(target, preferOriginal)
    case 'galgame_resource': {
      const work = target.work
        ? catalogNameText(target.work, preferOriginal)
        : ''
      const title = target.title || resourceTypeLabel(target.resource_type)
      return work ? `${work} · ${title}` : title
    }
    case 'rating':
      return target.work
        ? `${catalogNameText(target.work, preferOriginal)} 的评分`
        : '评分'
    case 'user':
      return toKunUser(target).name
    case 'unavailable_entity':
      return `无法预览的${ENTITY_NOUN[target.entity_type]}`
  }
}
