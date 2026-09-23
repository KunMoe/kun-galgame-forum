// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { createApiClient, sessionCookie } from './client'
import { settle, type ClientProblem, type FieldError } from './problem'
import { fieldMessage, problemMessage } from './message'

const origin = 'https://forum.test'

const jsonResponse = (
  status: number,
  body: unknown,
  headers: Record<string, string>
) =>
  new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers
  })

describe('createApiClient', () => {
  it('builds GET /topics with sort, limit and include_nsfw=true', async () => {
    let captured: Request | undefined
    const api = createApiClient({
      origin,
      fetch: async (input) => {
        captured = input
        return jsonResponse(
          200,
          { object: 'list', items: [] },
          { 'content-type': 'application/json' }
        )
      }
    })
    await api.GET('/topics', {
      params: {
        query: { sort: 'bumped_desc', limit: 10, include_nsfw: true }
      }
    })
    expect(captured).toBeDefined()
    const url = new URL(captured!.url)
    expect(url.origin).toBe(origin)
    expect(url.pathname.split('/')).toEqual(['', 'api', 'v1', 'topics'])
    expect(url.searchParams.get('sort')).toBe('bumped_desc')
    expect(url.searchParams.get('limit')).toBe('10')
    expect(url.searchParams.get('include_nsfw')).toBe('true')
  })

  it('sends an array query parameter comma-separated, the way the spec declares it', async () => {
    let captured: Request | undefined
    const api = createApiClient({
      origin,
      fetch: async (input) => {
        captured = input
        return jsonResponse(
          200,
          { object: 'list', items: [], missing: [] },
          { 'content-type': 'application/json' }
        )
      }
    })
    await api.GET('/me/topic-states', {
      params: { query: { topic_ids: ['12', '34', '56'] } }
    })
    const url = new URL(captured!.url)
    expect(url.searchParams.getAll('topic_ids')).toEqual(['12,34,56'])
  })

  it('sets the cookie header only when given, and credentials include', async () => {
    let withCookie: Request | undefined
    const apiCookie = createApiClient({
      origin,
      cookie: 'kungal_session=abc',
      fetch: async (input) => {
        withCookie = input
        return jsonResponse(200, { object: 'list', items: [] }, {})
      }
    })
    await apiCookie.GET('/topics', {})
    expect(withCookie!.credentials).toBe('include')
    expect(withCookie!.headers.get('cookie')).toBe('kungal_session=abc')

    let withoutCookie: Request | undefined
    const apiBare = createApiClient({
      origin,
      fetch: async (input) => {
        withoutCookie = input
        return jsonResponse(200, { object: 'list', items: [] }, {})
      }
    })
    await apiBare.GET('/topics', {})
    expect(withoutCookie!.credentials).toBe('include')
    expect(withoutCookie!.headers.get('cookie')).toBeNull()
  })

  it('settles a hanging fetch with timeoutMs 20 as timeout', async () => {
    const hanging = (input: Request) =>
      new Promise<Response>((_resolve, reject) => {
        const fail = () => {
          const reason = input.signal.reason
          reject(
            reason instanceof DOMException
              ? reason
              : new DOMException('The operation was aborted.', 'TimeoutError')
          )
        }
        if (input.signal.aborted) {
          fail()
          return
        }
        input.signal.addEventListener('abort', fail, { once: true })
      })
    const api = createApiClient({
      origin,
      timeoutMs: 20,
      fetch: hanging
    })
    const result = await settle(api.GET('/topics', {}))
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.problem.kind).toBe('timeout')
      expect(result.problem.status).toBe(0)
      expect(result.problem.code).toBeNull()
    }
  })

  it('still honours the caller signal when a timeout is set', async () => {
    const api = createApiClient({
      origin,
      timeoutMs: 60_000,
      fetch: (input) =>
        new Promise<Response>((_resolve, reject) => {
          input.signal.addEventListener(
            'abort',
            () => reject(new DOMException('aborted', 'AbortError')),
            { once: true }
          )
        })
    })
    const controller = new AbortController()
    const pending = settle(api.GET('/topics', { signal: controller.signal }))
    controller.abort()
    const result = await pending
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.problem.kind).toBe('network')
    }
  })
})

describe('sessionCookie', () => {
  it('keeps only kungal_session among several cookies', () => {
    expect(sessionCookie('theme=dark; kungal_session=tok; other=1')).toBe(
      'kungal_session=tok'
    )
  })

  it('is undefined without kungal_session or for undefined', () => {
    expect(sessionCookie('theme=dark; other=1')).toBeUndefined()
    expect(sessionCookie(undefined)).toBeUndefined()
  })
})

describe('settle', () => {
  it('normalizes a problem response', async () => {
    const body = {
      type: 'https://developer.nextmoe.dev/problems/platform/validation-failed',
      title: 'Validation failed',
      status: 422,
      detail: 'The request is syntactically valid but semantically not.',
      instance: '/topics',
      code: 'VALIDATION_FAILED',
      request_id: 'req_01ARZ3NDEKTSV4RRFFQ69G5FAV',
      errors: [
        {
          pointer: '/title',
          reason: 'TOO_LONG',
          detail: 'too long',
          params: { max_length: 8 }
        }
      ]
    }
    const result = await settle(
      Promise.resolve({
        error: body,
        response: jsonResponse(422, body, {
          'content-type': 'application/problem+json',
          'X-Request-ID': 'req_header'
        })
      })
    )
    expect(result).toEqual({
      ok: false,
      problem: {
        kind: 'problem',
        status: 422,
        code: 'VALIDATION_FAILED',
        errors: body.errors,
        requestId: 'req_01ARZ3NDEKTSV4RRFFQ69G5FAV'
      }
    })
  })

  it('treats a 502 HTML page as http with the request id header', async () => {
    const result = await settle(
      Promise.resolve({
        error: '<html>bad gateway</html>',
        response: new Response('<html>bad gateway</html>', {
          status: 502,
          headers: {
            'content-type': 'text/html; charset=utf-8',
            'X-Request-ID': 'req_html502'
          }
        })
      })
    )
    expect(result).toEqual({
      ok: false,
      problem: {
        kind: 'http',
        status: 502,
        code: null,
        errors: [],
        requestId: 'req_html502'
      }
    })
  })

  it('treats a 429 with an empty body as http', async () => {
    const result = await settle(
      Promise.resolve({
        error: undefined,
        response: new Response(null, { status: 429 })
      })
    )
    expect(result).toEqual({
      ok: false,
      problem: {
        kind: 'http',
        status: 429,
        code: null,
        errors: [],
        requestId: null
      }
    })
  })

  it('treats a problem-typed body without code as http', async () => {
    const body = { status: 400, title: 'Bad Request' }
    const result = await settle(
      Promise.resolve({
        error: body,
        response: jsonResponse(400, body, {
          'content-type': 'application/problem+json'
        })
      })
    )
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.problem.kind).toBe('http')
      expect(result.problem.code).toBeNull()
      expect(result.problem.status).toBe(400)
    }
  })

  it('treats a problem-shaped body served as application/json as http', async () => {
    const body = { code: 'NOT_FOUND', status: 404 }
    const result = await settle(
      Promise.resolve({
        error: body,
        response: jsonResponse(404, body, {
          'content-type': 'application/json'
        })
      })
    )
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.problem.kind).toBe('http')
      expect(result.problem.code).toBeNull()
      expect(result.problem.status).toBe(404)
    }
  })

  it('settles TypeError as network', async () => {
    const result = await settle(
      Promise.reject(new TypeError('Failed to fetch'))
    )
    expect(result).toEqual({
      ok: false,
      problem: {
        kind: 'network',
        status: 0,
        code: null,
        errors: [],
        requestId: null
      }
    })
  })

  it('settles AbortError as network', async () => {
    const result = await settle(
      Promise.reject(new DOMException('x', 'AbortError'))
    )
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.problem.kind).toBe('network')
    }
  })

  it('rethrows an Error', async () => {
    await expect(settle(Promise.reject(new Error('bug')))).rejects.toThrow(
      'bug'
    )
  })
})

const problem = (
  partial: Partial<ClientProblem> & Pick<ClientProblem, 'kind' | 'status'>
): ClientProblem => ({
  code: null,
  errors: [],
  requestId: null,
  ...partial
})

describe('problemMessage', () => {
  it('uses the catalogue code text', () => {
    expect(
      problemMessage(
        problem({ kind: 'problem', status: 401, code: 'MISSING_CREDENTIAL' })
      )
    ).toBe('请先登录')
  })

  it('falls back to the 409 status text for an unknown code', () => {
    expect(
      problemMessage(problem({ kind: 'problem', status: 409, code: 'NOPE' }))
    ).toBe('操作冲突，请刷新页面后重试')
  })

  it('falls back to client.default for an unknown code and status', () => {
    expect(
      problemMessage(problem({ kind: 'problem', status: 418, code: 'NOPE' }))
    ).toBe('出错了，请稍后重试')
  })

  it('uses client.network', () => {
    expect(problemMessage(problem({ kind: 'network', status: 0 }))).toBe(
      '网络请求失败，请检查网络后重试'
    )
  })

  it('uses client.timeout', () => {
    expect(problemMessage(problem({ kind: 'timeout', status: 0 }))).toBe(
      '请求超时，请稍后重试'
    )
  })

  it('uses the 502 status text for http 502', () => {
    expect(problemMessage(problem({ kind: 'http', status: 502 }))).toBe(
      '服务器暂时无法访问，请稍后重试'
    )
  })
})

const field = (reason: string, params?: FieldError['params']): FieldError => ({
  reason,
  detail: 'diagnostic',
  ...(params ? { params } : {})
})

describe('fieldMessage', () => {
  it('formats TOO_LONG with max_length', () => {
    expect(fieldMessage(field('TOO_LONG', { max_length: 233 }))).toBe(
      '最多 233 个字符'
    )
  })

  it('formats OUT_OF_RANGE variants', () => {
    expect(fieldMessage(field('OUT_OF_RANGE', { minimum: 1 }))).toBe(
      '不能小于 1'
    )
    expect(fieldMessage(field('OUT_OF_RANGE', { maximum: 10000 }))).toBe(
      '不能大于 10,000'
    )
    expect(
      fieldMessage(field('OUT_OF_RANGE', { minimum: 1, maximum: 10000 }))
    ).toBe('需要在 1 到 10,000 之间')
    expect(fieldMessage(field('OUT_OF_RANGE'))).toBe('超出了允许的范围')
  })

  it('formats UNKNOWN_VALUE allowed as a Chinese list', () => {
    expect(fieldMessage(field('UNKNOWN_VALUE', { allowed: ['a', 'b'] }))).toBe(
      '只能是以下之一：a、b'
    )
  })

  it('uses REQUIRED default', () => {
    expect(fieldMessage(field('REQUIRED'))).toBe('必填')
  })

  it('uses client.field for an unknown reason', () => {
    expect(fieldMessage(field('NOT_A_REASON'))).toBe('填写的内容有误')
  })
})
