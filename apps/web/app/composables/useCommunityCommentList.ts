import type { WallComment } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

export interface CommunityCommentGroup {
  root: WallComment
  replies: WallComment[]
}

const PAGE_LIMIT = 30

export const useCommunityCommentList = async (
  target: CommunityCommentTarget,
  initialTotal = 0
) => {
  const surface = communityCommentSurface(target)

  const { id: viewerId } = usePersistUserStore()
  const api = useApiClient()

  const posts = ref<WallComment[]>([])
  const total = ref(initialTotal)
  const following = ref(false)
  const nextCursor = ref<string | undefined>()
  const seeded = ref(false)
  const loadingMore = ref(false)
  const locked = ref(false)
  const loadFailed = ref(false)

  const listQuery = (cursor?: string) => ({
    subject_type: surface.subjectType,
    subject_id: surface.subjectId,
    limit: PAGE_LIMIT,
    ...(cursor ? { cursor } : {})
  })

  const { data, problem, status } = await useApi(
    `wall:${surface.subjectType}:${surface.subjectId}`,
    (client, { signal }) =>
      client.GET('/wall-comments', { params: { query: listQuery() }, signal }),
    { lazy: true }
  )

  // The read receipt is a write the reader makes, never something inferred from
  // the GET above: the community service refuses to treat a read face as a
  // write, and it answers with the viewer's follow state so the toggle has a
  // state to render. It creates nothing for a wall the viewer neither wrote on
  // nor follows. Failures stay silent: nobody asked for this request.
  const wallPath = {
    subject_type: surface.subjectType,
    subject_id: surface.subjectId
  }

  const reportRead = async () => {
    if (!viewerId || import.meta.server || locked.value) {
      return
    }
    const result = await settle(
      api.PUT('/me/walls/{subject_type}/{subject_id}/read-marker', {
        params: { path: wallPath }
      })
    )
    if (result.ok) {
      following.value = result.data.is_following
    }
  }

  const setFollowing = async (next: boolean) => {
    if (!viewerId) {
      useAuthModal().open()
      return
    }
    const call = next
      ? api.PUT('/me/walls/{subject_type}/{subject_id}/follow', {
          params: { path: wallPath }
        })
      : api.DELETE('/me/walls/{subject_type}/{subject_id}/follow', {
          params: { path: wallPath }
        })
    const result = await settle(call)
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    following.value = result.data.is_following
  }

  const seed = () => {
    if (seeded.value) {
      return
    }
    if (problem.value) {
      locked.value = problem.value.code === 'QUIZ_ANSWER_REQUIRED'
      loadFailed.value = !locked.value
      seeded.value = true
      return
    }
    if (!data.value) {
      return
    }
    posts.value = [...data.value.items]
    nextCursor.value = data.value.next_cursor
    seeded.value = true
    reportRead()
  }
  seed()
  watch([data, problem], seed)

  const hasMore = computed(() => nextCursor.value !== undefined)

  const loadMore = async () => {
    if (!hasMore.value || loadingMore.value) {
      return
    }
    loadingMore.value = true
    const result = await settle(
      api.GET('/wall-comments', {
        params: { query: listQuery(nextCursor.value) }
      })
    )
    loadingMore.value = false
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    const seen = new Set(posts.value.map((p) => p.id))
    posts.value = [
      ...posts.value,
      ...result.data.items.filter((p) => !seen.has(p.id))
    ]
    nextCursor.value = result.data.next_cursor
  }

  const groups = computed<CommunityCommentGroup[]>(() => {
    const list: CommunityCommentGroup[] = []
    const byRootId = new Map<string, CommunityCommentGroup>()
    for (const p of posts.value) {
      const owner = p.root_comment_id ? byRootId.get(p.root_comment_id) : null
      if (owner) {
        owner.replies.push(p)
        continue
      }
      const group: CommunityCommentGroup = { root: p, replies: [] }
      byRootId.set(p.id, group)
      list.push(group)
    }
    return list
  })

  const isEmpty = computed(
    () =>
      seeded.value &&
      !locked.value &&
      !loadFailed.value &&
      !hasMore.value &&
      posts.value.length === 0
  )

  const handleNewComment = (post: WallComment) => {
    if (posts.value.some((p) => p.id === post.id)) {
      return
    }
    posts.value = [...posts.value, post]
    total.value += 1
    reportRead()
  }

  const handleUpdated = (updated: WallComment) => {
    posts.value = posts.value.map((p) => (p.id === updated.id ? updated : p))
  }

  const handleTombstoned = (postId: string) => {
    posts.value = posts.value.map((p) =>
      p.id === postId
        ? {
            ...p,
            state: 'deleted',
            content: { ...p.content, children: [] },
            viewer: p.viewer && {
              ...p.viewer,
              can_edit: false,
              can_delete: false,
              can_like: false,
              can_flag: false
            }
          }
        : p
    )
    total.value = Math.max(total.value - 1, 0)
  }

  const scrollToPost = (postId: string) => {
    nextTick(() => {
      setTimeout(() => {
        const el = document.getElementById(`${surface.anchorPrefix}-${postId}`)
        if (!el) {
          return
        }
        el.scrollIntoView({ behavior: 'smooth', block: 'center' })
        const ring = [
          'outline-2',
          'outline-offset-2',
          'outline-primary',
          'rounded-lg'
        ]
        el.classList.add(...ring)
        setTimeout(() => el.classList.remove(...ring), 3000)
      }, 200)
    })
  }

  return {
    surface,
    posts,
    total,
    following,
    setFollowing,
    status,
    seeded,
    locked,
    loadFailed,
    hasMore,
    loadingMore,
    groups,
    isEmpty,
    loadMore,
    handleNewComment,
    handleUpdated,
    handleTombstoned,
    scrollToPost
  }
}
