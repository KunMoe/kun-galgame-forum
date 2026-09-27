import { describe, expect, it } from 'vitest'
import type { Comment, UserRef } from '#shared/utils/api/schemas'
import { threadComments } from './threadComments'

const user = (id: string): UserRef => ({
  object: 'user',
  id,
  name: `u${id}`,
  avatar: null,
  avatar_frame: null
})

const comment = (
  id: string,
  parent: string | null,
  created: string
): Comment => ({
  object: 'comment',
  id,
  reply_id: '9',
  reply_floor: 2,
  parent_comment_id: parent,
  author: user(id),
  in_reply_to_user: user('1'),
  content: {
    object: 'document',
    children: [
      { object: 'paragraph', children: [{ object: 'text', value: id }] }
    ]
  },
  like_count: 0,
  created_at: created,
  edited_at: null,
  viewer: null
})

describe('threadComments', () => {
  it('treats a comment whose parent is absent as top-level', () => {
    const orphan = comment('2', 'missing', '2026-01-01T00:00:02.000Z')
    const root = comment('1', null, '2026-01-01T00:00:01.000Z')
    const child = comment('3', '1', '2026-01-01T00:00:03.000Z')
    expect(threadComments([orphan, root, child])).toEqual([
      { comment: root, depth: 0 },
      { comment: child, depth: 1 },
      { comment: orphan, depth: 0 }
    ])
  })
})
