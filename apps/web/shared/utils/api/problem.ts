import type { components } from '../../types/api/v1'

export type FieldError = components['schemas']['FieldError']

export interface ClientProblem {
  kind: 'problem' | 'http' | 'network' | 'timeout'
  status: number
  code: string | null
  errors: FieldError[]
  requestId: string | null
}

// Only these say the entity is not there. Any other failure answered 404 too,
// and a crawler drops a 404 page: a slow catalog unlisted the page it served.
export const isMissingProblem = (p: ClientProblem | null) =>
  p !== null && (p.status === 404 || p.status === 400)

export const isTransientProblem = (p: ClientProblem) =>
  p.kind === 'timeout' || p.kind === 'network' || p.status >= 500

export type ApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; problem: ClientProblem }

const mediaTypeOf = (contentType: string | null): string => {
  if (!contentType) {
    return ''
  }
  return (contentType.split(';')[0] ?? '').trim().toLowerCase()
}

const isProblemBody = (
  body: unknown
): body is {
  code: string
  status: number
  errors?: unknown
  request_id?: unknown
} =>
  body !== null &&
  typeof body === 'object' &&
  typeof (body as { code?: unknown }).code === 'string' &&
  typeof (body as { status?: unknown }).status === 'number'

const requestIdOf = (
  body: { request_id?: unknown } | undefined,
  response: Response
): string | null => {
  if (typeof body?.request_id === 'string') {
    return body.request_id
  }
  return response.headers.get('X-Request-ID')
}

const asFailure = (problem: ClientProblem): ApiResult<never> => ({
  ok: false,
  problem
})

export const settle = async <T>(
  call: Promise<{ data?: T; error?: unknown; response: Response }>
): Promise<ApiResult<T>> => {
  try {
    const result = await call
    if (result.response.ok) {
      return { ok: true, data: result.data as T }
    }
    const media = mediaTypeOf(result.response.headers.get('content-type'))
    if (media === 'application/problem+json' && isProblemBody(result.error)) {
      return asFailure({
        kind: 'problem',
        status: result.error.status,
        code: result.error.code,
        errors: Array.isArray(result.error.errors)
          ? (result.error.errors as FieldError[])
          : [],
        requestId: requestIdOf(result.error, result.response)
      })
    }
    return asFailure({
      kind: 'http',
      status: result.response.status,
      code: null,
      errors: [],
      requestId: requestIdOf(undefined, result.response)
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'TimeoutError') {
      return asFailure({
        kind: 'timeout',
        status: 0,
        code: null,
        errors: [],
        requestId: null
      })
    }
    if (
      err instanceof TypeError ||
      (err instanceof DOMException && err.name === 'AbortError')
    ) {
      return asFailure({
        kind: 'network',
        status: 0,
        code: null,
        errors: [],
        requestId: null
      })
    }
    throw err
  }
}
