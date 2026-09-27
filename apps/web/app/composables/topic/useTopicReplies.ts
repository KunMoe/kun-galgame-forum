import type { ApiClient } from '#shared/utils/api/client'
import { settle, type ClientProblem } from '#shared/utils/api/problem'
import type { Reply } from '#shared/utils/api/schemas'
import { useApiClient } from '~/composables/useApi'

const PAGE_SIZE = 30

export const replyPageStart = (floor: number) =>
  Math.floor((floor - 1) / PAGE_SIZE) * PAGE_SIZE + 1

type SortOrder = 'asc' | 'desc'

type FailedLoad =
  | { type: 'initial'; fromFloor?: number }
  | { type: 'more' }
  | { type: 'earlier' }
  | { type: 'sort'; order: SortOrder }
  | { type: 'refresh'; id: string }

const sortParam = (order: SortOrder) =>
  order === 'desc' ? ('floor_desc' as const) : ('floor_asc' as const)

const dedupe = (items: Reply[]) => {
  const seen = new Set<string>()
  return items.filter((item) => {
    if (seen.has(item.id)) {
      return false
    }
    seen.add(item.id)
    return true
  })
}

export const useTopicReplies = (
  topicId: string,
  api?: Pick<ApiClient, 'GET'>
) => {
  const client = api ?? useApiClient()

  const replies = useState<Reply[]>(`kun-topic-replies-${topicId}`, () => [])
  const isComplete = useState(
    `kun-topic-replies-complete-${topicId}`,
    () => false
  )
  const hasEarlier = useState(
    `kun-topic-replies-earlier-${topicId}`,
    () => false
  )
  const status = useState<'idle' | 'pending' | 'success' | 'error'>(
    `kun-topic-replies-status-${topicId}`,
    () => 'idle'
  )
  const sortOrder = useState<SortOrder>(
    `kun-topic-replies-sort-${topicId}`,
    () => 'asc'
  )
  const problem = useState<ClientProblem | null>(
    `kun-topic-replies-problem-${topicId}`,
    () => null
  )
  const forwardCursor = useState<string | undefined>(
    `kun-topic-replies-fwd-${topicId}`,
    () => undefined
  )
  const backwardCursor = useState<string | undefined>(
    `kun-topic-replies-bwd-${topicId}`,
    () => undefined
  )
  const backwardStarted = useState(
    `kun-topic-replies-bwd-started-${topicId}`,
    () => false
  )
  const failedLoad = useState<FailedLoad | null>(
    `kun-topic-replies-failed-${topicId}`,
    () => null
  )

  const loadedIds = () => new Set(replies.value.map((reply) => reply.id))

  const notLoaded = (items: Reply[]) => {
    const seen = loadedIds()
    return items.filter((item) => {
      if (seen.has(item.id)) {
        return false
      }
      seen.add(item.id)
      return true
    })
  }

  const lowestFloor = () => {
    if (!replies.value.length) {
      return 1
    }
    return Math.min(...replies.value.map((reply) => reply.floor))
  }

  const fetchPage = async (query: {
    cursor?: string
    sort: 'floor_asc' | 'floor_desc'
    from_floor?: number
  }) =>
    settle(
      client.GET('/topics/{topic_id}/replies', {
        params: {
          path: { topic_id: topicId },
          query: {
            limit: PAGE_SIZE,
            sort: query.sort,
            ...(query.cursor ? { cursor: query.cursor } : {}),
            ...(query.from_floor !== undefined
              ? { from_floor: query.from_floor }
              : {})
          }
        }
      })
    )

  const fail = (load: FailedLoad, next: ClientProblem) => {
    status.value = 'error'
    problem.value = next
    failedLoad.value = load
  }

  const succeed = () => {
    status.value = 'success'
    problem.value = null
    failedLoad.value = null
  }

  const loadInitialReplies = async (opts: { fromFloor?: number } = {}) => {
    if (status.value === 'pending' || replies.value.length > 0) {
      return
    }
    status.value = 'pending'
    sortOrder.value = 'asc'
    backwardStarted.value = false
    backwardCursor.value = undefined
    const fromFloor = opts.fromFloor
    const result = await fetchPage({
      sort: 'floor_asc',
      ...(fromFloor !== undefined ? { from_floor: fromFloor } : {})
    })
    if (!result.ok) {
      fail({ type: 'initial', fromFloor }, result.problem)
      return
    }
    replies.value = dedupe(result.data.items)
    forwardCursor.value = result.data.next_cursor
    isComplete.value = !result.data.next_cursor
    hasEarlier.value = fromFloor !== undefined && fromFloor > 1
    succeed()
  }

  const loadMore = async () => {
    if (status.value === 'pending' || isComplete.value) {
      return
    }
    const cursor = forwardCursor.value
    if (!cursor) {
      isComplete.value = true
      return
    }
    status.value = 'pending'
    const result = await fetchPage({
      sort: sortParam(sortOrder.value),
      cursor
    })
    if (!result.ok) {
      fail({ type: 'more' }, result.problem)
      return
    }
    replies.value.push(...notLoaded(result.data.items))
    forwardCursor.value = result.data.next_cursor
    isComplete.value = !result.data.next_cursor
    succeed()
  }

  const loadEarlier = async () => {
    if (status.value === 'pending' || !hasEarlier.value) {
      return
    }
    status.value = 'pending'
    const result = await fetchPage(
      backwardStarted.value
        ? {
            sort: 'floor_desc',
            ...(backwardCursor.value ? { cursor: backwardCursor.value } : {})
          }
        : { sort: 'floor_desc', from_floor: lowestFloor() - 1 }
    )
    if (!result.ok) {
      fail({ type: 'earlier' }, result.problem)
      return
    }
    backwardStarted.value = true
    backwardCursor.value = result.data.next_cursor
    hasEarlier.value = Boolean(result.data.next_cursor)
    const incoming = notLoaded(result.data.items).sort(
      (left, right) => left.floor - right.floor
    )
    replies.value.unshift(...incoming)
    succeed()
  }

  const setSort = async (order: SortOrder) => {
    if (status.value === 'pending' || sortOrder.value === order) {
      return
    }
    status.value = 'pending'
    const result = await fetchPage({ sort: sortParam(order) })
    if (!result.ok) {
      fail({ type: 'sort', order }, result.problem)
      return
    }
    sortOrder.value = order
    backwardStarted.value = false
    backwardCursor.value = undefined
    hasEarlier.value = false
    replies.value = dedupe(result.data.items)
    forwardCursor.value = result.data.next_cursor
    isComplete.value = !result.data.next_cursor
    succeed()
  }

  const addNewReply = (newReply: Reply) => {
    if (replies.value.some((reply) => reply.id === newReply.id)) {
      return
    }
    if (sortOrder.value === 'desc' && !hasEarlier.value) {
      replies.value.unshift(newReply)
    } else {
      replies.value.push(newReply)
    }
  }

  const updateReply = (updated: Reply) => {
    const index = replies.value.findIndex((reply) => reply.id === updated.id)
    if (index !== -1) {
      replies.value[index] = updated
    }
  }

  const removeReply = (id: string) => {
    const index = replies.value.findIndex((reply) => reply.id === id)
    if (index !== -1) {
      replies.value.splice(index, 1)
    }
  }

  const refreshReply = async (id: string) => {
    const result = await settle(
      client.GET('/replies/{reply_id}', {
        params: { path: { reply_id: id } }
      })
    )
    if (!result.ok) {
      if (result.problem.status === 404) {
        removeReply(id)
        return
      }
      fail({ type: 'refresh', id }, result.problem)
      return
    }
    const index = replies.value.findIndex(
      (reply) => reply.id === result.data.id
    )
    if (index !== -1) {
      replies.value[index] = result.data
    } else {
      addNewReply(result.data)
    }
    if (problem.value) {
      problem.value = null
      failedLoad.value = null
      if (status.value === 'error') {
        status.value = 'success'
      }
    }
  }

  const retry = async () => {
    const failed = failedLoad.value
    if (!failed || status.value === 'pending') {
      return
    }
    problem.value = null
    failedLoad.value = null
    if (failed.type === 'initial') {
      await loadInitialReplies({ fromFloor: failed.fromFloor })
      return
    }
    if (failed.type === 'more') {
      await loadMore()
      return
    }
    if (failed.type === 'earlier') {
      await loadEarlier()
      return
    }
    if (failed.type === 'sort') {
      await setSort(failed.order)
      return
    }
    await refreshReply(failed.id)
  }

  return {
    replies,
    status,
    isComplete,
    hasEarlier,
    sortOrder,
    problem,
    loadInitialReplies,
    loadMore,
    loadEarlier,
    setSort,
    addNewReply,
    updateReply,
    removeReply,
    refreshReply,
    retry
  }
}
