// @vitest-environment nuxt
import { describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import GalgameEdit from '~/components/activity/card/GalgameEdit.vue'

const apiClient = vi.hoisted(() => vi.fn())
mockNuxtImport('useApiClient', () => apiClient)

const work = {
  object: 'work',
  id: '31',
  display_name: 'Work 31',
  latin: null,
  localized: {}
}

const activity = (changed: string[]) => ({
  path: '/galgame/31',
  occurred_at: '2026-10-21T08:30:00Z',
  performer: null,
  work_digest: null,
  work_revision: {
    revision_number: 3,
    legacy_revision_id: null,
    changed_fields: changed
  }
})

const mountCard = (changed: string[]) =>
  mountSuspended(GalgameEdit, {
    props: { activity: activity(changed), work } as never,
    global: {
      stubs: {
        ActivityCardShell: { template: '<div><slot /></div>' },
        ActivityCardGalgameInfo: true
      }
    }
  })

describe('galgame edit activity card', () => {
  it('lists the fields the revision touched without asking the API', async () => {
    const wrapper = await mountCard([
      'catalog.work.titles',
      'catalog.work.covers.suppressed',
      'titles'
    ])
    const text = wrapper.text()
    expect(text).toContain('修改了')
    expect(text).toContain('标题与别名')
    expect(text).toContain('隐藏的封面')
    expect(text.match(/标题与别名/g)).toHaveLength(1)
    expect(apiClient).not.toHaveBeenCalled()
  })

  it('shows no field row for an edit recorded before changed_fields', async () => {
    const wrapper = await mountCard([])
    expect(wrapper.text()).not.toContain('修改了')
  })
})
