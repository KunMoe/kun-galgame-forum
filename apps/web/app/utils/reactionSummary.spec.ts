import { describe, expect, it } from 'vitest'
import type { ReactionSummary, UserRef } from '#shared/utils/api/schemas'
import { toKunReactions } from './reactionSummary'

const user = (id: string, name: string | null = 'Ada'): UserRef => ({
  object: 'user',
  id,
  name,
  avatar: {
    hash: 'ab',
    url: `https://cdn.example/${id}.webp`,
    width: 64,
    height: 64,
    thumbhash: null,
    sexual: null
  },
  avatar_frame: null
})

const summary = (over: Partial<ReactionSummary> = {}): ReactionSummary => ({
  reaction: 'heart',
  count: 4,
  reactors: [user('2', 'Bea'), user('3', null)],
  viewer: { has_reacted: true },
  ...over
})

describe('toKunReactions', () => {
  it('maps count, mine from viewer.has_reacted, and reactors through toKunUser', () => {
    expect(toKunReactions([summary()])).toEqual([
      {
        reaction: 'heart',
        count: 4,
        mine: true,
        reactors: [
          {
            id: 2,
            name: 'Bea',
            avatar: 'https://cdn.example/2.webp',
            avatarDecoration: null
          },
          {
            id: 3,
            name: '已注销用户',
            avatar: 'https://cdn.example/3.webp',
            avatarDecoration: null
          }
        ]
      }
    ])
  })

  it('sets mine to false when viewer is null', () => {
    expect(toKunReactions([summary({ viewer: null })])[0]!.mine).toBe(false)
  })

  it('sets mine to false when viewer.has_reacted is false', () => {
    expect(
      toKunReactions([summary({ viewer: { has_reacted: false } })])[0]!.mine
    ).toBe(false)
  })
})
