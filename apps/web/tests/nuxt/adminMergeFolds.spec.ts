// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import MergeFolds from '~/pages/admin/merge-folds.vue'

const page = {
  object: 'list',
  total: 2,
  total_relation: 'eq',
  items: [
    {
      object: 'work_merge_fold',
      id: '2',
      old_work_id: '5904',
      new_work_id: '61101',
      folded_at: '2026-10-21T08:00:00Z',
      undone_at: '2026-10-22T08:00:00Z',
      moved_count: 3,
      dropped_count: 1,
      summed_count: 0,
      repointed_count: 0,
      problems: [
        {
          table_name: 'galgame_resource',
          row_id: '77',
          cause: 'elsewhere',
          current_work_id: '70000'
        }
      ]
    },
    {
      object: 'work_merge_fold',
      id: '1',
      old_work_id: '12',
      new_work_id: '13',
      folded_at: '2026-10-21T07:00:00Z',
      undone_at: null,
      moved_count: 1,
      dropped_count: 0,
      summed_count: 0,
      repointed_count: 0,
      problems: []
    }
  ]
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('admin merge folds page', () => {
  it('lists folds and says which rows an undo could not put back', async () => {
    const fetchSpy = vi.fn(
      async (_request: Request) =>
        new Response(JSON.stringify(page), {
          headers: { 'content-type': 'application/json' }
        })
    )
    vi.stubGlobal('fetch', fetchSpy)
    const wrapper = await mountSuspended(MergeFolds)
    await flushPromises()

    const url = new URL(fetchSpy.mock.calls[0]![0].url)
    expect(url.pathname).toMatch(/\/admin\/work-merge-folds$/)
    expect(url.searchParams.get('page')).toBe('1')
    expect(url.searchParams.get('limit')).toBe('20')
    const text = wrapper.text()
    expect(text).toContain('已撤销')
    expect(text).toContain('生效中')
    expect(text).toContain('迁移 3')
    expect(text).toContain('有 1 条没能搬回')
    expect(text).toContain('资源 #77: 现在挂在另一部作品上')
    expect(text).toContain('#70000')
  })
})
