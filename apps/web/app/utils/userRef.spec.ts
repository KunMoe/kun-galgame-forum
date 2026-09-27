import { describe, expect, it } from 'vitest'
import type { UserRef } from '#shared/utils/api/schemas'
import { toKunUser } from './userRef'

const ref = (over: Partial<UserRef> = {}): UserRef => ({
  object: 'user',
  id: '7',
  name: 'Bob',
  avatar: {
    hash: 'ab',
    url: 'https://cdn.example/ab.webp',
    width: 64,
    height: 64,
    thumbhash: null,
    sexual: null
  },
  avatar_frame: null,
  ...over
})

describe('toKunUser', () => {
  it('maps a living user', () => {
    expect(toKunUser(ref())).toEqual({
      id: 7,
      name: 'Bob',
      avatar: 'https://cdn.example/ab.webp',
      avatarDecoration: null
    })
  })

  it('uses the deleted-user label and an empty avatar when those fields are null', () => {
    expect(toKunUser(ref({ name: null, avatar: null }))).toEqual({
      id: 7,
      name: '已注销用户',
      avatar: '',
      avatarDecoration: null
    })
  })

  it('maps the avatar frame to the KunUI decoration', () => {
    const framed = ref({
      avatar_frame: {
        static_url: 'https://img.example/decorations/f.png',
        animated_url: 'https://img.example/decorations/f.webp'
      }
    })
    expect(toKunUser(framed).avatarDecoration).toStrictEqual({
      src: 'https://img.example/decorations/f.png',
      animatedSrc: 'https://img.example/decorations/f.webp'
    })
    const still = ref({
      avatar_frame: {
        static_url: 'https://img.example/decorations/f.png',
        animated_url: null
      }
    })
    expect(toKunUser(still).avatarDecoration).toEqual({
      src: 'https://img.example/decorations/f.png'
    })
  })
})
