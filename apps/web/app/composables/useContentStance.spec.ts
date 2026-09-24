// @vitest-environment nuxt
import { afterEach, describe, expect, it } from 'vitest'

afterEach(() => {
  usePersistUserStore().resetUser()
  usePersistSettingsStore().showKUNGalgameContentLimit = 'sfw'
})

describe('useContentStance, logged out', () => {
  it('is the same binary cookie switch it has always been', () => {
    const { stance, allowsNsfw, isBlurred, setAnonymousNsfw } =
      useContentStance()
    const settings = usePersistSettingsStore()

    expect(stance.value).toBe('hide')
    expect(allowsNsfw.value).toBe(false)

    setAnonymousNsfw(true)
    expect(settings.showKUNGalgameContentLimit).toBe('nsfw')
    expect(stance.value).toBe('show')
    expect(allowsNsfw.value).toBe(true)
    // Nothing logged out can ever reach 模糊, so no surface has to mask.
    expect(isBlurred.value).toBe(false)

    setAnonymousNsfw(false)
    expect(settings.showKUNGalgameContentLimit).toBe('sfw')
    expect(stance.value).toBe('hide')
  })

  it('still honours the legacy "all" value', () => {
    usePersistSettingsStore().showKUNGalgameContentLimit = 'all'
    expect(useContentStance().allowsNsfw.value).toBe(true)
  })
})

describe('useContentStance, signed in', () => {
  const signIn = (adultConfirmed: boolean, nsfwDisplay: string) => {
    usePersistUserStore().setUserInfo({
      id: 7,
      sub: 'u-7',
      name: 'kun',
      avatar: '',
      avatarMin: '',
      moemoepoint: 0,
      roles: ['user'],
      isCheckIn: false,
      dailyToolsetUploadBytes: 0,
      adultConfirmed,
      nsfwDisplay
    })
  }

  it('ignores an nsfw cookie left over from before this wave', () => {
    usePersistSettingsStore().showKUNGalgameContentLimit = 'nsfw'
    signIn(true, 'hide')

    const { stance, allowsNsfw } = useContentStance()
    expect(stance.value).toBe('hide')
    expect(allowsNsfw.value).toBe(false)
  })

  // adult_confirmed is constant true upstream since 2026-09-23, but a session
  // written before the claim existed still folds to hide rather than to the
  // stored display value.
  it('a missing adult_confirmed claim still folds to hide', () => {
    signIn(false, 'show')
    expect(useContentStance().stance.value).toBe('hide')
  })

  it('blur lets content through masked, show lets it through plain', () => {
    signIn(true, 'blur')
    const blurred = useContentStance()
    expect(blurred.stance.value).toBe('blur')
    expect(blurred.allowsNsfw.value).toBe(true)
    expect(blurred.isBlurred.value).toBe(true)

    signIn(true, 'show')
    const shown = useContentStance()
    expect(shown.stance.value).toBe('show')
    expect(shown.isBlurred.value).toBe(false)
  })
})

describe('useContentStance, stanceKey', () => {
  const signIn = (nsfwDisplay: string) => {
    usePersistUserStore().setUserInfo({
      id: 7,
      sub: 'u-7',
      name: 'kun',
      avatar: '',
      avatarMin: '',
      moemoepoint: 0,
      roles: ['user'],
      isCheckIn: false,
      dailyToolsetUploadBytes: 0,
      adultConfirmed: true,
      nsfwDisplay
    })
  }

  // A key that moves after hydration misses the SSR payload: setup then ran
  // with no data and an NSFW work rendered past its gate.
  it('does not move when a signed-in stance flips', () => {
    signIn('show')
    const { stanceKey } = useContentStance()
    expect(stanceKey.value).toBe('me:7')

    usePersistUserStore().setContentStance(true, 'hide')
    expect(stanceKey.value).toBe('me:7')
  })

  it('follows the cookie for an anonymous reader', () => {
    const { stanceKey, setAnonymousNsfw } = useContentStance()
    expect(stanceKey.value).toBe('sfw')
    setAnonymousNsfw(true)
    expect(stanceKey.value).toBe('nsfw')
  })
})
