import { settle, type ApiResult, type ClientProblem } from './problem'
import { problemMessage } from './message'

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

const parse = (text: string): unknown => {
  try {
    return JSON.parse(text)
  } catch {
    return undefined
  }
}

export const createChatApi = (
  origin: string,
  fetcher: typeof fetch = (input, init) => globalThis.fetch(input, init)
) => {
  const request = <T>(
    method: Method,
    path: string,
    body?: unknown
  ): Promise<ApiResult<T>> =>
    settle<T>(
      (async () => {
        const headers: Record<string, string> = { Accept: 'application/json' }
        let payload: BodyInit | undefined
        if (body instanceof FormData) {
          payload = body
        } else if (body !== undefined) {
          headers['Content-Type'] = 'application/json'
          payload = JSON.stringify(body)
        }
        const response = await fetcher(`${origin}/api/v1/chat${path}`, {
          method,
          headers,
          body: payload,
          credentials: 'include'
        })
        const text = await response.text()
        const json = text ? parse(text) : undefined
        return response.ok
          ? { data: json as T, response }
          : { error: json, response }
      })()
    )

  return {
    get: <T>(path: string) => request<T>('GET', path),
    post: <T = undefined>(path: string, body?: unknown) =>
      request<T>('POST', path, body),
    put: <T = undefined>(path: string, body?: unknown) =>
      request<T>('PUT', path, body),
    patch: <T = undefined>(path: string, body?: unknown) =>
      request<T>('PATCH', path, body),
    del: <T = undefined>(path: string) => request<T>('DELETE', path)
  }
}

export type ChatApi = ReturnType<typeof createChatApi>

const chatMessages: Record<string, string> = {
  CHAT_BLOCKED: '你们之间有拉黑关系，无法互发私信',
  CHAT_NOT_ACCEPTING: '对方暂不接受你的私信，或你的账号注册未满 72 小时',
  CHAT_REQUEST_LIMIT:
    '对方接受私信请求前，你只能发送最多 3 条不含链接和图片的文字消息',
  CHAT_EDIT_WINDOW_CLOSED: '消息发出超过 48 小时后不能再编辑',
  CHAT_NOT_PERMITTED: '你不能对这条消息这样做',
  SCOPE_REQUIRED: '这次登录还没有授权私信，请重新登录'
}

export const chatProblemMessage = (problem: ClientProblem): string =>
  (problem.code && chatMessages[problem.code]) || problemMessage(problem)
