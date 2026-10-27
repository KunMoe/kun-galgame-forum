import type { AsyncDataOptions } from '#app'
import {
  createApiClient,
  sessionCookie,
  type ApiClient
} from '#shared/utils/api/client'
import {
  isTransientProblem,
  settle,
  type ApiResult,
  type ClientProblem
} from '#shared/utils/api/problem'

let browserClient: ApiClient | undefined

export const useApiClient = (): ApiClient => {
  if (import.meta.server) {
    return createApiClient({
      origin: useRuntimeConfig().apiBaseUrl,
      cookie: sessionCookie(useRequestHeaders(['cookie']).cookie),
      timeoutMs: 10000
    })
  }
  browserClient ??= createApiClient({
    origin: useRuntimeConfig().public.apiBaseUrl
  })
  return browserClient
}

type FetchOutcome<T> = { data?: T; error?: unknown; response: Response }

type ApiCall<R> = (api: ApiClient, ctx: { signal: AbortSignal }) => Promise<R>

export type UseKunApiOptions<T> = Pick<
  AsyncDataOptions<ApiResult<T>>,
  'lazy' | 'server' | 'immediate' | 'getCachedData'
> & {
  // The page's main read. If it fails for a moment during SSR the response is
  // 503: the 200 it used to be, with an error body and noindex, told crawlers
  // to drop the page whenever the catalog was slow.
  critical?: boolean
}

export type KunApi<T> = {
  data: ComputedRef<T | undefined>
  problem: ComputedRef<ClientProblem | null>
  // Nothing to show for this key yet; a refresh keeps the old data on screen.
  pending: ComputedRef<boolean>
  status: ReturnType<typeof useAsyncData>['status']
  refresh: ReturnType<typeof useAsyncData>['refresh']
}

const toResult = <T>(
  pending: Promise<FetchOutcome<T> | ApiResult<T>>
): Promise<ApiResult<T>> =>
  pending.then(
    (out) => ('ok' in out ? out : settle(Promise.resolve(out))),
    (error: unknown) => settle<T>(Promise.reject(error))
  )

// Lazy by default: an awaited fetch holds client navigation until it lands.
// The server renders after every fetch either way, so `await` is only needed
// where setup reads the data on the server, e.g. to throw a 404.
export const useKunApi = <T>(
  key: MaybeRefOrGetter<string>,
  call: ApiCall<FetchOutcome<T>> | ApiCall<ApiResult<T>>,
  options?: UseKunApiOptions<T>
): KunApi<T> & Promise<KunApi<T>> => {
  const { critical, ...asyncOptions } = options ?? {}
  const api = useApiClient()
  const asyncData = useAsyncData(
    () => `api:${toValue(key)}`,
    (_app, { signal }) => toResult(call(api, { signal })),
    { lazy: true, ...asyncOptions }
  )
  const result = asyncData.data

  const kunApi: KunApi<T> = {
    data: computed(() => (result.value?.ok ? result.value.data : undefined)),
    problem: computed(() =>
      result.value && !result.value.ok ? result.value.problem : null
    ),
    pending: computed(
      () => asyncData.status.value === 'pending' && result.value === undefined
    ),
    status: asyncData.status,
    refresh: asyncData.refresh
  }

  if (import.meta.server && critical) {
    const event = useRequestEvent()
    const retryAfter = useResponseHeader('Retry-After')
    watch(
      kunApi.problem,
      (problem) => {
        if (event && problem && isTransientProblem(problem)) {
          setResponseStatus(event, 503)
          retryAfter.value = '60'
        }
      },
      { flush: 'sync' }
    )
  }

  return Object.assign(
    Promise.resolve(asyncData).then(() => kunApi),
    kunApi
  )
}
