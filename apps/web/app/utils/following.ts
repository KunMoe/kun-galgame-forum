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
  edit: { verb: '编辑了', unit: '个' },
  favorite: { verb: '收藏了', unit: '个' },
  follow: { verb: '关注了', unit: '个' },
  mark: { verb: '标记了', unit: '个' },
  play: { verb: '游玩了', unit: '个' }
}

const MARK_PHRASE: Record<string, string> = {
  work_wish: '想玩',
  work_doing: '在玩',
  work_done: '玩过',
  work_on_hold: '搁置了',
  work_dropped: '弃坑了'
}

const SITE_LABEL: Record<string, string> = {
  moyu: 'moyu.moe',
  letmoe: 'letmoe.com'
}

const spaced = (label: string) =>
  /^[A-Za-z0-9]/.test(label) ? ` ${label}` : label

export const followingGroupPhrase = (
  group: Pick<
    FollowingActivityGroup,
    'verb' | 'item_count' | 'object_kind' | 'object_label'
  >
) => {
  const { unit } = VERB_PHRASE[group.verb]
  const verb =
    (group.verb === 'mark' && MARK_PHRASE[group.object_kind]) ||
    VERB_PHRASE[group.verb].verb
  const count = group.item_count > 1 ? ` ${group.item_count} ${unit}` : ''
  return `${verb}${count}${spaced(group.object_label)}`
}

// catalog is the play record every NextMoe site shares, not a site to visit.
export const followingSiteLabel = (site: string) =>
  site === 'kungal' || site === 'catalog' ? '' : (SITE_LABEL[site] ?? site)
