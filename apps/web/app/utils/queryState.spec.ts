import { describe, expect, it } from 'vitest'
import { nextTick, watch } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import {
  createQueryState,
  queryEnum,
  queryIds,
  queryInt,
  queryPage,
  queryString,
  type QueryRoute
} from './queryState'

const stub = { template: '<div />' }

const schema = {
  page: queryPage(),
  platform: queryString(),
  sort: queryEnum(['bumped', 'created'] as const, 'bumped', {
    update_time: 'bumped'
  }),
  ids: queryIds(),
  minRating: queryInt(0, 0)
}

const liveRoute = (router: Router): QueryRoute => ({
  get path() {
    return router.currentRoute.value.path
  },
  get hash() {
    return router.currentRoute.value.hash
  },
  get query() {
    return router.currentRoute.value.query
  }
})

const setup = async (url: string) => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/list', component: stub },
      { path: '/detail', component: stub }
    ]
  })
  await router.push(url)
  const navigations: string[] = []
  const replaces: unknown[] = []
  const replace = router.replace.bind(router)
  router.replace = (to) => {
    replaces.push(to)
    return replace(to)
  }
  router.afterEach((to, _from, failure) => {
    if (!failure) {
      navigations.push(to.fullPath)
    }
  })
  const route = liveRoute(router)
  const q = createQueryState(schema, { route, router, pageKey: 'page' })
  return { router, route, navigations, replaces, q }
}

const settle = async () => {
  for (let i = 0; i < 5; i++) {
    await nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))
  }
}

const queryOf = (router: Router) => router.currentRoute.value.query

describe('createQueryState', () => {
  it('parses the whole query, with defaults, aliases and bad values dropped', async () => {
    const { q } = await setup(
      '/list?page=3&sort=update_time&ids=3,x,3,-1,5&minRating=abc'
    )
    expect(q.state.value).toEqual({
      page: 3,
      platform: '',
      sort: 'bumped',
      ids: [3, 5],
      minRating: 0
    })
  })

  it('folds several writes in one tick into one navigation and one state change', async () => {
    const { router, navigations, replaces, q } = await setup('/list?page=4')
    let changes = 0
    watch(q.state, () => changes++, { flush: 'sync' })

    q.platform.value = 'win'
    q.sort.value = 'created'
    q.set({ minRating: 7 })
    await settle()

    expect(replaces).toHaveLength(1)
    expect(navigations).toHaveLength(1)
    expect(queryOf(router)).toEqual({
      platform: 'win',
      sort: 'created',
      minRating: '7'
    })
    expect(changes).toBe(1)
  })

  it('sends a filter change back to page 1, but keeps an explicit page and a no-op', async () => {
    const { router, q } = await setup('/list?page=3&platform=win')

    q.set({ platform: 'win' })
    await settle()
    expect(queryOf(router).page).toBe('3')

    q.set({ platform: 'mac', page: 2 })
    await settle()
    expect(queryOf(router)).toEqual({ page: '2', platform: 'mac' })

    q.platform.value = 'and'
    await settle()
    expect(queryOf(router)).toEqual({ platform: 'and' })
  })

  it('keeps the page a link or back/forward arrives with', async () => {
    const { router, q } = await setup('/list?page=3')
    await router.replace('/list?platform=win&page=2')
    await settle()
    expect(q.state.value.page).toBe(2)
    expect(queryOf(router).page).toBe('2')
  })

  it('keeps unchanged values, and the whole state, identical across navigations', async () => {
    const { router, q } = await setup('/list?ids=1,2')
    const ids = q.state.value.ids
    q.page.value = 2
    await settle()
    expect(q.state.value.ids).toBe(ids)

    const whole = q.state.value
    await router.replace({
      path: '/list',
      query: { ...queryOf(router), x: '1' }
    })
    await settle()
    expect(q.state.value).toBe(whole)
  })

  it('shares one navigation between two states on the same page', async () => {
    const { router, route, replaces, q } = await setup('/list')
    const tabs = createQueryState(
      { tab: queryString('all') },
      { route, router }
    )
    q.platform.value = 'win'
    tabs.tab.value = 'mine'
    await settle()
    expect(replaces).toHaveLength(1)
    expect(queryOf(router)).toEqual({ platform: 'win', tab: 'mine' })
  })

  it('does not let a write in the next tick undo one still in flight', async () => {
    const { router, q } = await setup('/list')
    q.platform.value = 'win'
    await nextTick()
    q.sort.value = 'created'
    await settle()
    expect(queryOf(router)).toEqual({ platform: 'win', sort: 'created' })
  })

  it('drops defaults and keeps keys it does not own', async () => {
    const { router, q } = await setup('/list?keep=1&platform=win&sort=created')
    q.set({ platform: '', sort: 'bumped' })
    await settle()
    expect(queryOf(router)).toEqual({ keep: '1' })
  })

  it('does not write from a page that is already leaving', async () => {
    const { router } = await setup('/list?page=3')
    const leaving: QueryRoute = {
      path: '/list',
      hash: '',
      query: { page: '3' }
    }
    const q = createQueryState(schema, {
      route: leaving,
      router,
      pageKey: 'page'
    })
    await router.push('/detail')
    q.platform.value = 'win'
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/detail')
  })
})
