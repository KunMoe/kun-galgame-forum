// @vitest-environment nuxt
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { flushPromises } from '@vue/test-utils'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))

mockNuxtImport('useApiClient', () => () => ({ GET: get }))

const ok = (items: unknown[]) => ({
  data: { object: 'list', items, missing: [] },
  response: new Response(null, { status: 200 })
})
const unavailable = () => ({
  error: { status: 503, code: 'SERVICE_UNAVAILABLE', errors: [] },
  response: new Response(null, {
    status: 503,
    headers: { 'content-type': 'application/problem+json' }
  })
})
const myWork = (id: string, collectionIds: string[] | null) => ({
  object: 'my_work',
  work_id: id,
  has_liked: true,
  library: collectionIds
    ? { collection_ids: collectionIds, playtime: null }
    : null
})

// revalidate-me also listens to visibilitychange and GETs /me/account.
const worksCalls = () =>
  get.mock.calls.filter(([path]) => path === '/me/works').length

const setVisibility = (state: DocumentVisibilityState) => {
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    get: () => state
  })
}

beforeEach(() => {
  clearNuxtState()
  setVisibility('visible')
  usePersistUserStore().id = 7
})

afterEach(() => {
  get.mockReset()
  usePersistUserStore().resetUser()
})

describe('useMyGalgameInteractions', () => {
  it('reads collected from library, and null library as unknown', async () => {
    get.mockResolvedValue(
      ok([myWork('1', ['9']), myWork('2', []), myWork('3', null)])
    )
    const { isFavorited, ensureLoaded } = useMyGalgameInteractions()
    ensureLoaded([1, 2, 3])
    await flushPromises()

    expect(isFavorited(1)).toBe(true)
    expect(isFavorited(2)).toBe(false)
    // catalog unreadable for this reader: unknown, never "not collected"
    expect(isFavorited(3)).toBeNull()
  })

  it('stays unknown while loading and after a failed read', async () => {
    get.mockResolvedValue(unavailable())
    const { isFavorited, ensureLoaded } = useMyGalgameInteractions()
    expect(isFavorited(5)).toBeNull()
    ensureLoaded([5])
    await flushPromises()
    expect(isFavorited(5)).toBeNull()
  })

  it('answers false for a signed-out reader, who can still click', () => {
    usePersistUserStore().resetUser()
    const { isFavorited, ensureLoaded } = useMyGalgameInteractions()
    ensureLoaded([1])
    expect(isFavorited(1)).toBe(false)
    expect(worksCalls()).toBe(0)
  })

  it('holds the saved value until the re-read lands', async () => {
    get.mockResolvedValueOnce(ok([myWork('4', [])]))
    const { isFavorited, setFavorited, ensureLoaded } =
      useMyGalgameInteractions()
    ensureLoaded([4])
    await flushPromises()
    expect(isFavorited(4)).toBe(false)

    get.mockResolvedValueOnce(ok([myWork('4', ['11'])]))
    setFavorited(4, true)
    expect(isFavorited(4)).toBe(true)
    await flushPromises()
    expect(worksCalls()).toBe(2)
    expect(isFavorited(4)).toBe(true)
  })

  // A browser restoring a batch of tabs spent one reader's catalog quota.
  it('waits for a hidden tab to be shown before reading', async () => {
    setVisibility('hidden')
    get.mockResolvedValue(ok([myWork('6', ['1'])]))
    const { isFavorited, ensureLoaded } = useMyGalgameInteractions()
    ensureLoaded([6])
    await flushPromises()
    expect(worksCalls()).toBe(0)
    expect(isFavorited(6)).toBeNull()

    setVisibility('visible')
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()
    expect(worksCalls()).toBe(1)
    expect(isFavorited(6)).toBe(true)
  })
})
