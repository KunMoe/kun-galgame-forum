import type { FollowingActivityGroup } from '#shared/utils/api/schemas'

const VERB_PHRASE: Record<
  FollowingActivityGroup['verb'],
  { verb: string; unit: string }
> = {
  publish: { verb: '发布了', unit: '个' },
  reply: { verb: '发表了', unit: '条' },
  comment: { verb: '发表了', unit: '条' },
  rate: { verb: '发表了', unit: '条' },
  like: { verb: '推荐了', unit: '个' },
  edit: { verb: '编辑了', unit: '个' }
}

const SITE_LABEL: Record<string, string> = {
  moyu: 'moyu.moe',
  letmoe: 'letmoe.com'
}

const spaced = (label: string) =>
  /^[A-Za-z0-9]/.test(label) ? ` ${label}` : label

export const followingGroupPhrase = (
  group: Pick<FollowingActivityGroup, 'verb' | 'item_count' | 'object_label'>
) => {
  const { verb, unit } = VERB_PHRASE[group.verb]
  const count = group.item_count > 1 ? ` ${group.item_count} ${unit}` : ''
  return `${verb}${count}${spaced(group.object_label)}`
}

export const followingSiteLabel = (site: string) =>
  site === 'kungal' ? '' : (SITE_LABEL[site] ?? site)
