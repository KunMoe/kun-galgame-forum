import type { TopicSource } from '#shared/utils/api/schemas'
import { toTopicAccessRoles, toTopicAccessScope } from '~/constants/topic'
import { toKunUser } from '~/utils/userRef'

export const COVER_IMAGE_TOKEN = /^\/image\/([0-9a-f]{64})$/

export const coverHashFromToken = (token: string): string => {
  const match = token.match(COVER_IMAGE_TOKEN)
  return match?.[1] ?? token
}

export const coverTokenFromHash = (hash: string): string => `/image/${hash}`

export const applyTopicSource = (source: TopicSource) => {
  const temp = useTempEditStore()
  temp.id = Number(source.topic_id)
  temp.title = source.title
  temp.content = source.content_markdown
  temp.section = [source.section]
  temp.isNSFW = source.is_nsfw
  temp.coverImages = source.cover_images.map((image) =>
    coverTokenFromHash(image.hash)
  )
  temp.accessScope = toTopicAccessScope(source.access_scope)
  temp.accessRoles = toTopicAccessRoles(source.access_grants.roles)
  temp.accessUsers = source.access_grants.users.map(toKunUser)
  temp.accessUserIds = temp.accessUsers.map((user) => user.id)
  temp.isTopicRewriting = true
}
