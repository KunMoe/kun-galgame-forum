// @vitest-environment nuxt
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type { Poll, PollViewer, UserRef } from '#shared/utils/api/schemas'
import TopicPollList from './List.vue'
import TopicPollLog from './Log.vue'

const { reportProblem } = vi.hoisted(() => ({ reportProblem: vi.fn() }))
mockNuxtImport('reportProblem', () => reportProblem)

// A signed-in store starts the cloud-preferences sync, and through the typed
// client its GET /me/preferences lands in this spec's fetch stub first.
mockNuxtImport('useCloudPreferences', () => () => ({
  sync: async () => {},
  flush: async () => {}
}))

const user = (id: string): UserRef => ({
  object: 'user',
  id,
  name: `u${id}`,
  avatar: null
})

const pollViewer = (over: Partial<PollViewer> = {}): PollViewer => ({
  has_voted: false,
  chosen_option_ids: [],
  can_vote: true,
  can_change_vote: true,
  can_edit: false,
  can_delete: false,
  can_view_results: true,
  ...over
})

const poll = (over: Partial<Poll> = {}): Poll => ({
  object: 'poll',
  id: '7',
  topic_id: '42',
  title: '今晚吃什么',
  description: '',
  choice_type: 'single',
  min_choice: 1,
  max_choice: 1,
  closes_at: null,
  result_visibility: 'always',
  is_anonymous: false,
  can_change_vote: true,
  author: user('2'),
  options: [
    { object: 'poll_option', id: '11', text: '拉面' },
    { object: 'poll_option', id: '12', text: '咖喱' }
  ],
  results: {
    total_vote_count: 3,
    voter_count: 3,
    options: [
      { option_id: '11', vote_count: 2 },
      { option_id: '12', vote_count: 1 }
    ],
    sample_voters: [user('3')]
  },
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  viewer: pollViewer(),
  ...over
})

type Captured = { method: string; url: string; body: string }

const captureFetch = (payload: unknown, status = 200) => {
  const captured: Captured[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: Request | string | URL) => {
      const request =
        input instanceof Request ? input : new Request(String(input))
      captured.push({
        method: request.method,
        url: request.url,
        body: request.body ? await request.text() : ''
      })
      return new Response(JSON.stringify(payload), {
        status,
        headers: { 'content-type': 'application/json' }
      })
    })
  )
  return captured
}

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'content-type': 'application/json' }
  })

let wrapper: VueWrapper | undefined

const buttonIn = (target: VueWrapper, text: string) =>
  target.findAll('button').find((button) => button.text().trim() === text)

const bodyButton = (text: string) =>
  [...document.body.querySelectorAll('button')].find(
    (button) => (button.textContent ?? '').trim() === text
  )

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.unstubAllGlobals()
  reportProblem.mockClear()
  usePersistUserStore().resetUser()
  clearNuxtData()
})

describe('a poll vote is a slot, not a toggle', () => {
  it('sets the vote with PUT and retracts it with DELETE', async () => {
    usePersistUserStore().id = 1
    const voted = poll({
      viewer: pollViewer({ has_voted: true, chosen_option_ids: ['11'] })
    })

    const setting = captureFetch(voted)
    wrapper = await mountSuspended(TopicPollList, { props: { poll: poll() } })
    await wrapper
      .findAllComponents({ name: 'KunCheckBox' })[0]!
      .vm.$emit('change')
    await nextTick()
    await buttonIn(wrapper, '投票')!.trigger('click')
    await vi.waitFor(() => expect(setting).toHaveLength(1))
    expect(setting[0]!.method).toBe('PUT')
    expect(setting[0]!.url).toMatch(/\/polls\/7\/vote$/)
    expect(JSON.parse(setting[0]!.body)).toEqual({ option_ids: ['11'] })
    expect(reportProblem).not.toHaveBeenCalled()
    wrapper.unmount()

    const clearing = captureFetch(poll())
    wrapper = await mountSuspended(TopicPollList, { props: { poll: voted } })
    await buttonIn(wrapper, '撤回投票')!.trigger('click')
    await vi.waitFor(() => expect(clearing).toHaveLength(1))
    expect(clearing[0]!.method).toBe('DELETE')
    expect(clearing[0]!.url).toMatch(/\/polls\/7\/vote$/)
  })

  it('offers no vote button when viewer.can_vote is false', async () => {
    usePersistUserStore().id = 1
    const captured = captureFetch(poll())
    wrapper = await mountSuspended(TopicPollList, {
      props: {
        poll: poll({
          viewer: pollViewer({
            has_voted: true,
            chosen_option_ids: ['11'],
            can_vote: false,
            can_change_vote: false
          })
        })
      }
    })
    expect(buttonIn(wrapper, '投票')).toBeUndefined()
    expect(buttonIn(wrapper, '修改投票')).toBeUndefined()
    expect(buttonIn(wrapper, '撤回投票')).toBeUndefined()
    expect(captured).toHaveLength(0)
  })
})

describe('a poll with no results block says so', () => {
  it('renders the hidden-results hint instead of zero tallies', async () => {
    usePersistUserStore().id = 1
    captureFetch(poll())
    wrapper = await mountSuspended(TopicPollList, {
      props: {
        poll: poll({
          results: null,
          result_visibility: 'after_vote',
          viewer: pollViewer({ can_view_results: false })
        })
      }
    })
    const text = wrapper.text()
    expect(text).toContain('投票后可以看到结果')
    expect(text).not.toContain('0 票')
    expect(text).not.toContain('0.0%')
    expect(text).not.toContain('共 0 票')
  })

  it('renders tallies when the results block is present', async () => {
    usePersistUserStore().id = 1
    captureFetch(poll())
    wrapper = await mountSuspended(TopicPollList, { props: { poll: poll() } })
    const text = wrapper.text()
    expect(text).toContain('2 票')
    expect(text).toContain('66.7%')
    expect(text).not.toContain('投票后可以看到结果')
  })

  it('does not call an anonymous poll empty while it holds votes', async () => {
    usePersistUserStore().id = 1
    const anonymous = poll({
      is_anonymous: true,
      results: {
        total_vote_count: 3,
        voter_count: 3,
        options: [
          { option_id: '11', vote_count: 2 },
          { option_id: '12', vote_count: 1 }
        ],
        sample_voters: []
      }
    })
    captureFetch(anonymous)
    wrapper = await mountSuspended(TopicPollList, {
      props: { poll: anonymous }
    })
    const text = wrapper.text()
    expect(text).toContain('共 3 票')
    expect(text).not.toContain('还没有人投票')
  })

  it('says a poll is empty only when nobody has voted', async () => {
    usePersistUserStore().id = 1
    const empty = poll({
      results: {
        total_vote_count: 0,
        voter_count: 0,
        options: [
          { option_id: '11', vote_count: 0 },
          { option_id: '12', vote_count: 0 }
        ],
        sample_voters: []
      }
    })
    captureFetch(empty)
    wrapper = await mountSuspended(TopicPollList, { props: { poll: empty } })
    expect(wrapper.text()).toContain('还没有人投票')
  })
})

describe('the vote log is a cursor page', () => {
  const vote = (id: string, optionId: string) => ({
    object: 'poll_vote',
    id,
    poll_id: '7',
    option_id: optionId,
    voter: user('3'),
    created_at: '2026-01-01T00:00:00Z'
  })

  it('reads the next page with the cursor and names the option', async () => {
    const fetchSpy = vi.fn(async (input: Request | string | URL) => {
      const url = new URL(input instanceof Request ? input.url : String(input))
      if (!url.pathname.endsWith('/votes')) {
        return jsonResponse({})
      }
      if (!url.searchParams.get('cursor')) {
        return jsonResponse({
          object: 'list',
          items: [vote('1', '11')],
          next_cursor: 'cur_next'
        })
      }
      return jsonResponse({ object: 'list', items: [vote('2', '12')] })
    })
    vi.stubGlobal('fetch', fetchSpy)

    wrapper = await mountSuspended(TopicPollLog, {
      props: {
        pollId: '7',
        options: poll().options,
        modelValue: true
      }
    })

    await vi.waitFor(() =>
      expect(document.body.textContent ?? '').toContain('加载更多')
    )
    expect(document.body.textContent ?? '').toContain('投给了「拉面」')

    bodyButton('加载更多')!.click()
    await vi.waitFor(() =>
      expect(
        fetchSpy.mock.calls.some((call) => {
          const input = call[0] as Request | string | URL
          const url = new URL(
            input instanceof Request ? input.url : String(input)
          )
          return url.searchParams.get('cursor') === 'cur_next'
        })
      ).toBe(true)
    )
    await vi.waitFor(() =>
      expect(document.body.textContent ?? '').toContain('投给了「咖喱」')
    )
  })
})
