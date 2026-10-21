// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { computed, ref } from 'vue'
import GalgameDetailPage from '~/pages/galgame/[id]/index.vue'

const deleted = {
  code: 'ENTITY_DELETED',
  status: 410,
  object: 'work',
  deleted_reason: 'spam_or_test',
  deleted_at: '2026-10-21T08:30:00Z'
}

mockNuxtImport(
  'useApi',
  () =>
    async (
      _key: unknown,
      fetcher: (
        api: unknown,
        opts: { signal: AbortSignal }
      ) => Promise<unknown>
    ) => {
      const api = { GET: async () => ({ error: deleted }) }
      await fetcher(api, { signal: new AbortController().signal })
      return {
        data: computed(() => undefined),
        problem: computed(() => ({ status: 410, code: 'ENTITY_DELETED' })),
        status: ref('error'),
        refresh: () => {}
      }
    }
)

describe('galgame detail of a deleted work', () => {
  it('says the work was deleted and why', async () => {
    const wrapper = await mountSuspended(GalgameDetailPage, {
      route: '/galgame/31'
    })
    expect(wrapper.text()).toContain(
      '这部作品已被删除: 这是垃圾或测试条目 (2026-10-21)'
    )
  })
})
