// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'

interface TagPathInit {
  params: { path: { tag_id: string } }
  body?: { mode: string }
}

const { calls } = vi.hoisted(() => ({ calls: [] as string[] }))

const ok = (data?: unknown) => ({
  data,
  response: new Response(null, { status: 200 })
})

mockNuxtImport('useApiClient', () => () => ({
  GET: async (path: string) => {
    if (path !== '/me/blocked-tags') {
      return { response: new Response(null, { status: 503 }) }
    }
    calls.push(`GET ${path}`)
    return ok({
      object: 'list',
      items: [
        { object: 'tag_block', id: '6', mode: 'tree', created_at: '' },
        { object: 'tag_block', id: '9', mode: 'exact', created_at: '' }
      ]
    })
  },
  PUT: async (_path: string, init: TagPathInit) => {
    calls.push(`PUT ${init.params.path.tag_id} ${init.body?.mode}`)
    return ok({
      object: 'tag_block',
      id: init.params.path.tag_id,
      mode: init.body?.mode,
      created_at: ''
    })
  },
  DELETE: async (_path: string, init: TagPathInit) => {
    calls.push(`DELETE ${init.params.path.tag_id}`)
    return { response: new Response(null, { status: 204 }) }
  }
}))

const signIn = () =>
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
    nsfwDisplay: 'show'
  })

afterEach(() => {
  calls.length = 0
  usePersistUserStore().resetUser()
  const store = usePersistTagBlockStore()
  store.anonymous = { tree: [], exact: [] }
  store.account = { tree: [], exact: [] }
  store.accountUid = 0
})

describe('useTagBlocks', () => {
  it('keeps a signed-out reader on the cookie and sends ids sorted', async () => {
    const tb = useTagBlocks()
    await tb.block(9)
    await tb.block(3, 'exact')
    await tb.block(5)
    expect(calls).toEqual([])
    expect(tb.excludeQuery.value).toEqual({
      exclude_tag_trees: ['5', '9'],
      exclude_tag_ids: ['3']
    })

    await tb.block(9, 'exact')
    await tb.unblock(3)
    expect(tb.modeOf(9)).toBe('exact')
    expect(tb.excludeQuery.value).toEqual({
      exclude_tag_trees: ['5'],
      exclude_tag_ids: ['9']
    })
  })

  it('reads a signed-in reader from the account, never the cookie', async () => {
    usePersistTagBlockStore().anonymous = { tree: [1], exact: [] }
    signIn()
    const tb = useTagBlocks()
    expect(tb.excludeQuery.value).toEqual({})

    await tb.refresh()
    expect(tb.excludeQuery.value).toEqual({
      exclude_tag_trees: ['6'],
      exclude_tag_ids: ['9']
    })

    await tb.unblock(6)
    expect(calls).toContain('DELETE 6')
    expect(tb.excludeQuery.value).toEqual({ exclude_tag_ids: ['9'] })
  })

  it('loads the account list before its first write instead of overwriting it', async () => {
    signIn()
    const tb = useTagBlocks()
    await tb.block(12)
    expect(calls).toEqual(['GET /me/blocked-tags', 'PUT 12 tree'])
    expect(tb.excludeQuery.value).toEqual({
      exclude_tag_trees: ['6', '12'],
      exclude_tag_ids: ['9']
    })
  })
})
