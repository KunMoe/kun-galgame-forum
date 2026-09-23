import type { ComputedRef } from 'vue'
import { storeToRefs } from 'pinia'

const MODERATOR_PERMISSIONS = [
  'topic.edit_any',
  'topic.hide',
  'topic.view_hidden',
  'topic.view_restricted',
  'topic.set_best_answer',
  'reply.edit_any',
  'reply.delete_any',
  'reply.pin',
  'comment.topic.edit',
  'comment.topic.delete',
  'comment.galgame.edit',
  'comment.galgame.delete',
  'comment.rating.edit',
  'comment.rating.delete',
  'comment.website.edit',
  'comment.website.delete',
  'comment.toolset.edit',
  'comment.toolset.delete',
  'comment.resource.edit',
  'comment.resource.delete',
  'comment.quiz.edit',
  'comment.quiz.delete',
  'poll.create_any',
  'poll.edit_any',
  'poll.delete_any',
  'poll.view_restricted',
  'lottery.create_any',
  'lottery.manage_any',
  'lottery.view_restricted',
  'galgame.ban_resource_publish',
  'galgame.claim.review',
  'collection.edit_any',
  'collection.delete_any',
  'quiz.edit_any',
  'quiz.delete_any',
  'resource.edit_any',
  'resource.delete_any',
  'rating.delete_any',
  'toolset.edit_any',
  'toolset.delete_any',
  'toolset.resource.edit_any',
  'toolset.resource.delete_any',
  'toolset.upload_bypass',
  'doc.create',
  'doc.edit',
  'doc.delete',
  'website.create',
  'website.edit',
  'website.delete',
  'friend_link.create',
  'friend_link.edit',
  'friend_link.delete',
  'update_log.create',
  'update_log.edit',
  'update_log.delete',
  'trust.review'
] as const

const ADMIN_ONLY_PERMISSIONS = [
  'admin.dashboard',
  'user.purge_content',
  'topic.delete_any',
  'update_log.reopen'
] as const

const ADMIN_PERMISSIONS = [
  ...MODERATOR_PERMISSIONS,
  ...ADMIN_ONLY_PERMISSIONS
] as const

export type ForumPermission =
  | (typeof MODERATOR_PERMISSIONS)[number]
  | (typeof ADMIN_ONLY_PERMISSIONS)[number]

const ROLE_PERMISSIONS: Record<string, ReadonlySet<ForumPermission>> = {
  moderator: new Set(MODERATOR_PERMISSIONS),
  admin: new Set(ADMIN_PERMISSIONS),
  ren: new Set(ADMIN_PERMISSIONS)
}

export const useCan = (permission: ForumPermission): ComputedRef<boolean> => {
  const { roles } = storeToRefs(usePersistUserStore())
  const mine = useState<string[] | null>('kun-perm-mine', () => null)

  return computed(() => {
    const list = mine.value
    if (list) {
      return list.includes(permission)
    }
    return roles.value.some(
      (role) => ROLE_PERMISSIONS[role]?.has(permission) ?? false
    )
  })
}

export const useMyPermissions = (): ComputedRef<
  (permission: string) => boolean
> => {
  const { roles } = storeToRefs(usePersistUserStore())
  const mine = useState<string[] | null>('kun-perm-mine', () => null)

  return computed(() => {
    const list = mine.value
    if (list) {
      return (permission: string) => list.includes(permission)
    }
    return (permission: string) =>
      roles.value.some(
        (role) =>
          ROLE_PERMISSIONS[role]?.has(permission as ForumPermission) ?? false
      )
  })
}
