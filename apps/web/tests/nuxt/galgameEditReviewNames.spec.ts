// @vitest-environment nuxt
import { describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { computed, ref } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { createApiClient } from '#shared/utils/api/client'
import ReviewDetail from '~/components/galgame-edit/review/Detail.vue'

const requests = vi.hoisted(() => [] as string[])
const toast = vi.hoisted(() => ({ calls: [] as unknown[][] }))

const proposal = {
  object: 'edit_proposal',
  id: '41',
  work_id: '31',
  state: 'open',
  note: null,
  patch: {},
  effective_patch: {
    'catalog.work.engine_ids': [7, 8],
    'catalog.work.series_ids': [9]
  },
  amendments: [],
  proposer: { object: 'user', id: '3', name: 'kun', avatar: null },
  decider: null,
  created_at: '2026-09-24T00:00:00Z',
  decided_at: null,
  base_revision_seq: 1,
  viewer: { can_decide: false }
}

const listField = (key: string) => ({
  key,
  field_type: 'list',
  diff_hint: 'items',
  element: { element_type: 'ref', members: [] },
  encoding: null,
  base: 0,
  is_deprecated: false,
  is_nullable: false,
  max_elements: 50,
  max_suppressed: 0,
  vocabulary: ''
})

const form = {
  fields: [
    listField('catalog.work.engine_ids'),
    listField('catalog.work.series_ids')
  ],
  field_values: {
    'catalog.work.engine_ids': [7],
    'catalog.work.series_ids': []
  },
  vocabularies: []
}

mockNuxtImport('useApi', () => (key: () => string) => {
  const k = key()
  const data = k.startsWith('edit-proposal:')
    ? { proposal, etag: null }
    : k.startsWith('work-edit-form:')
      ? form
      : undefined
  const result = {
    data: computed(() => data),
    problem: computed(() => null),
    status: ref('success'),
    refresh: vi.fn()
  }
  return Object.assign(Promise.resolve(result), result)
})

mockNuxtImport('useCan', () => () => computed(() => false))

mockNuxtImport('useMessage', () => (...args: unknown[]) => {
  toast.calls.push(args)
})

mockNuxtImport(
  'useApiClient',
  () => () =>
    createApiClient({
      origin: 'https://forum.test',
      fetch: async (input) => {
        const url = new URL((input as Request).url)
        const [, , version, ...rest] = url.pathname.split('/')
        const path =
          version === 'v1' ? rest.join('/') : `legacy:${url.pathname}`
        requests.push(path)
        const json = (
          status: number,
          body: unknown,
          type = 'application/json'
        ) =>
          new Response(JSON.stringify(body), {
            status,
            headers: { 'content-type': type }
          })
        if (path === 'works/31') {
          return json(200, {
            id: '31',
            tags: [],
            companies: [],
            engines: [
              {
                id: '7',
                display_name: 'Known Engine',
                latin: null,
                localized: {}
              }
            ],
            series: [],
            roster: [],
            credits: []
          })
        }
        if (path === 'engines/8') {
          return json(200, {
            object: 'engine',
            id: '8',
            display_name: 'Siglus',
            latin: null,
            localized: {}
          })
        }
        return json(
          404,
          {
            type: 'about:blank',
            title: 'Not found',
            status: 404,
            code: 'NOT_FOUND',
            request_id: 'req_01M00000000000000000000000',
            errors: []
          },
          'application/problem+json'
        )
      }
    })
)

describe('galgame edit review detail', () => {
  it('names the relations a proposal adds through the v1 entity reads, and shows a raw id for a missing one', async () => {
    const wrapper = await mountSuspended(ReviewDetail, {
      route: '/galgame-edit/review/41'
    })
    await flushPromises()
    await flushPromises()

    const text = wrapper.text().replace(/\s+/g, ' ')
    expect(text).toContain('引擎 + Siglus')
    expect(text).toContain('所属系列 + #9')
    expect(requests).toContain('engines/8')
    expect(requests).toContain('series/9')
    expect(requests).not.toContain('engines/7')
    expect(requests.some((r) => r.startsWith('legacy:'))).toBe(false)
    expect(toast.calls).toEqual([])
  })
})
