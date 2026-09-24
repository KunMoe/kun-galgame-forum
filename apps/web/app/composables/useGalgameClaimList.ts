import type { WorkSubmissionSummary } from '#shared/utils/api/schemas'

const CLAIM_PAGE_LIMIT = 20

export const useGalgameClaimList = (lane: 'mine' | 'reviews') =>
  useCursorList<WorkSubmissionSummary>(
    `work-submissions:${lane}`,
    (api, cursor, { signal }) =>
      lane === 'mine'
        ? api.GET('/me/work-submissions', {
            params: {
              query: {
                state: ['pending', 'declined', 'draft'],
                limit: CLAIM_PAGE_LIMIT,
                cursor
              }
            },
            signal
          })
        : api.GET('/me/work-submission-reviews', {
            params: { query: { limit: CLAIM_PAGE_LIMIT, cursor } },
            signal
          })
  )
