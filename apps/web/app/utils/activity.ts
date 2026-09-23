import type {
  Activity,
  ActivityType,
  ListActivitiesQuery,
  WorkRef
} from '#shared/utils/api/schemas'
import { markdownToText } from '#shared/utils/markdownToText'

export type ActivityStreamQuery = Omit<ListActivitiesQuery, 'cursor' | 'limit'>

const LEGACY_RENAMED: Record<string, ActivityType> = {
  MESSAGE_SOLUTION: 'best_answer_set'
}

// The home feed tabs persist the pre-v1 tokens in the settings store, so they
// are translated here instead of migrating every stored preference.
export const feedTabQuery = (legacyTypes: string): ActivityStreamQuery => {
  const tokens = legacyTypes.split(',').filter(Boolean)
  const types = new Set<ActivityType>()
  let normal = false
  let help = false
  for (const token of tokens) {
    if (token === 'TOPIC_NORMAL') {
      normal = true
      types.add('topic_creation')
    } else if (token === 'TOPIC_RESOURCE_HELP') {
      help = true
      types.add('topic_creation')
    } else if (token === 'MESSAGE_UPVOTE') {
      continue
    } else {
      types.add(LEGACY_RENAMED[token] ?? (token.toLowerCase() as ActivityType))
    }
  }
  const query: ActivityStreamQuery = {
    activity_types: [...types],
    topic_sections: normal && help ? 'all' : help ? 'help' : 'normal'
  }
  if (types.size === 1 && types.has('topic_creation')) {
    query.sort = 'bumped_desc'
  }
  return query
}

export const activitySummaryText = (
  a: Activity,
  nameOf: (work: WorkRef) => string
): string => {
  const name = a.work ? nameOf(a.work) : ''
  switch (a.activity_type) {
    case 'galgame_creation':
      return name
    case 'galgame_resource_creation':
      return `在《${name}》发布了下载资源`
    case 'galgame_edit':
      return `编辑了《${name}》`
    case 'galgame_pr_creation':
      return `对《${name}》提出了更新请求`
    case 'galgame_rating_creation':
      return a.galgame_rating?.short_summary
        ? `${name} · ${a.galgame_rating.short_summary}`
        : name
    default:
      return markdownToText(a.excerpt_markdown)
  }
}
