import type { WallSubjectType } from '#shared/utils/api/schemas'

export type CommunityCommentTarget =
  | { kind: 'galgame'; galgameId: number }
  | { kind: 'rating'; ratingId: number }
  | { kind: 'website'; websiteId: number }
  | { kind: 'toolset'; toolsetId: number }
  | { kind: 'resource'; resourceId: number }
  | { kind: 'quiz'; quizId: number }

export interface CommunityCommentSurface {
  subjectType: WallSubjectType
  subjectId: string
  maxLength: number
  anchorPrefix: string
  composerPlaceholder: string
  showsReplyTarget: boolean
  // The follow and read-receipt routes are still the legacy /community/wall/*
  // faces, which address a wall by the community service's own anchor.
  wallAnchor: { anchor_kind: number; anchor_id: string }
}

const MENTION_PLACEHOLDER =
  '请温柔的发表你的看法吧～「评论给」已废除，@用户名 即可通知对方'

const SITE_GAME = 1
const SITE_RESOURCE = 2

const resourceWall = (prefix: string, id: number) => ({
  anchor_kind: SITE_RESOURCE,
  anchor_id: `${prefix}:${id}`
})

export const communityCommentSurface = (
  target: CommunityCommentTarget
): CommunityCommentSurface => {
  switch (target.kind) {
    case 'galgame':
      return {
        subjectType: 'galgame',
        subjectId: String(target.galgameId),
        maxLength: 5000,
        anchorPrefix: 'galgame-comment',
        composerPlaceholder: MENTION_PLACEHOLDER,
        showsReplyTarget: false,
        wallAnchor: { anchor_kind: SITE_GAME, anchor_id: String(target.galgameId) }
      }
    case 'rating':
      return {
        subjectType: 'galgame_rating',
        subjectId: String(target.ratingId),
        maxLength: 1314,
        anchorPrefix: 'rating-comment',
        composerPlaceholder: '发布对这个评分的观点，请不要锐评',
        showsReplyTarget: true,
        wallAnchor: resourceWall('rating', target.ratingId)
      }
    case 'website':
      return {
        subjectType: 'website',
        subjectId: String(target.websiteId),
        maxLength: 1007,
        anchorPrefix: 'website-comment',
        composerPlaceholder: '说说你对这个网站的看法吧～',
        showsReplyTarget: true,
        wallAnchor: resourceWall('website', target.websiteId)
      }
    case 'toolset':
      return {
        subjectType: 'toolset',
        subjectId: String(target.toolsetId),
        maxLength: 1007,
        anchorPrefix: 'toolset-comment',
        composerPlaceholder: '对这个工具有任何使用疑问，都可以在这里提出～',
        showsReplyTarget: true,
        wallAnchor: resourceWall('toolset', target.toolsetId)
      }
    case 'resource':
      return {
        subjectType: 'galgame_resource',
        subjectId: String(target.resourceId),
        maxLength: 1007,
        anchorPrefix: 'resource-comment',
        composerPlaceholder: '这个资源能正常使用吗？有问题可以在这里反馈～',
        showsReplyTarget: true,
        wallAnchor: resourceWall('resource', target.resourceId)
      }
    case 'quiz':
      return {
        subjectType: 'galgame_quiz',
        subjectId: String(target.quizId),
        maxLength: 1007,
        anchorPrefix: 'quiz-comment',
        composerPlaceholder: '聊聊这道题目吧～请不要直接剧透答案',
        showsReplyTarget: true,
        wallAnchor: resourceWall('quiz', target.quizId)
      }
  }
}
