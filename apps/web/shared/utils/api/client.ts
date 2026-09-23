import createClient from 'openapi-fetch'
import type { paths } from '../../types/api/v1'

export type CreateApiClientOptions = {
  origin: string
  cookie?: string
  timeoutMs?: number
  fetch?: (input: Request) => Promise<Response>
}

export const sessionCookie = (
  cookieHeader: string | undefined
): string | undefined => {
  if (!cookieHeader) {
    return undefined
  }
  for (const part of cookieHeader.split(';')) {
    const trimmed = part.trim()
    if (trimmed.startsWith('kungal_session=')) {
      return trimmed
    }
  }
  return undefined
}

export const createApiClient = (options: CreateApiClientOptions) => {
  const baseFetch =
    options.fetch ?? ((input: Request) => globalThis.fetch(input))
  const fetchWithTimeout = (input: Request): Promise<Response> => {
    if (options.timeoutMs === undefined) {
      return baseFetch(input)
    }
    const timeout = AbortSignal.timeout(options.timeoutMs)
    const signal = input.signal
      ? AbortSignal.any([input.signal, timeout])
      : timeout
    return baseFetch(new Request(input, { signal }))
  }

  return createClient<paths>({
    baseUrl: `${options.origin}/api/v1`,
    credentials: 'include',
    fetch: fetchWithTimeout,
    // huma declares every array query parameter explode:false and reads only
    // the first of repeated keys, so openapi-fetch's default
    // topic_ids=1&topic_ids=2 answered for topic 1 alone.
    querySerializer: { array: { style: 'form', explode: false } },
    ...(options.cookie ? { headers: { cookie: options.cookie } } : {})
  })
}

export type ApiClient = ReturnType<typeof createApiClient>
