import type { InjectionKey } from 'vue'
import type {
  ListMoyuPatch,
  ListWalkthrough,
  RatingPage
} from '#shared/utils/api/schemas'

const RATING_PREVIEW_LIMIT = 6

const createGalgamePageRequests = (workId: string) => ({
  ratings: useKunApi<RatingPage>(
    () => `work-ratings:${workId}`,
    (api, { signal }) =>
      api.GET('/ratings', {
        params: {
          query: {
            work_id: workId,
            include_nsfw: true,
            limit: RATING_PREVIEW_LIMIT,
            sort: 'created_desc'
          }
        },
        signal
      })
  ),
  walkthroughs: useKunApi<ListWalkthrough>(
    () => `galgame-walkthroughs:${workId}`,
    (api, { signal }) =>
      api.GET('/walkthroughs', {
        params: { query: { work_id: workId, include_nsfw: true } },
        signal
      })
  ),
  // moyu is another site: the server never waits on it, and the page never
  // holds its reveal for it. A failed lookup is not reported, since the patch
  // tab is only drawn when there is something in it.
  moyuPatches: useKunApi<ListMoyuPatch>(
    () => `galgame-moyu-patches:${workId}`,
    (api, { signal }) =>
      api.GET('/works/{work_id}/moyu-patches', {
        params: { path: { work_id: workId } },
        signal
      }),
    { server: false }
  )
})

type GalgamePageRequests = ReturnType<typeof createGalgamePageRequests>
const GALGAME_PAGE: InjectionKey<GalgamePageRequests> = Symbol('galgame-page')

// The page starts these with the work itself. Fetched from the work view they
// only started once the work had landed, and the rating radar then pushed the
// whole page down.
export const provideGalgamePage = (workId: string) => {
  const requests = createGalgamePageRequests(workId)
  provide(GALGAME_PAGE, requests)
  return requests
}

export const useGalgamePage = () => {
  const requests = inject(GALGAME_PAGE)
  if (!requests) {
    throw new Error('useGalgamePage outside a galgame page')
  }
  return requests
}
