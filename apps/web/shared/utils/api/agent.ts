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

export const agentPath = (origin: string, path: string) =>
  `${origin}/api/v1/agent${path}`

export const createAgentApi = (
  origin: string,
  fetcher: typeof fetch = (input, init) => globalThis.fetch(input, init)
) => {
  const request = <T>(
    method: Method,
    path: string,
    body?: unknown,
    extra?: Record<string, string>
  ): Promise<ApiResult<T>> =>
    settle<T>(
      (async () => {
        const headers: Record<string, string> = {
          Accept: 'application/json',
          ...extra
        }
        let payload: BodyInit | undefined
        if (body instanceof FormData) {
          payload = body
        } else if (body !== undefined) {
          headers['Content-Type'] = 'application/json'
          payload = JSON.stringify(body)
        }
        const response = await fetcher(agentPath(origin, path), {
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
    post: <T = undefined>(
      path: string,
      body?: unknown,
      headers?: Record<string, string>
    ) => request<T>('POST', path, body, headers),
    put: <T = undefined>(path: string, body?: unknown) =>
      request<T>('PUT', path, body),
    patch: <T = undefined>(path: string, body?: unknown) =>
      request<T>('PATCH', path, body),
    del: <T = undefined>(path: string) => request<T>('DELETE', path)
  }
}

export type AgentApi = ReturnType<typeof createAgentApi>

const agentMessages: Record<string, string> = {
  ASSISTANT_DISABLED: '本站的 AI 助手暂未开放',
  CLIENT_NOT_BOUND: '本站的 AI 助手暂未开放',
  FREE_UNITS_EXHAUSTED: '今天的免费次数已用完，明天再来吧',
  INSUFFICIENT_MOEMOEPOINTS: '今天的免费次数已用完，NextMoe 萌萌点也不足',
  RUN_ACTIVE: '上一条回复还在生成，请稍候',
  IDEMPOTENCY_KEY_REUSED: '这条消息已经发出过了，刷新后再试',
  BAD_PARENT: '要接着回复的消息不在这个对话里',
  NOT_REGENERABLE: '这条消息不能重新生成',
  EMPTY_MESSAGE: '消息不能为空',
  UNSUPPORTED_ATTACHMENT: '这种文件不能上传',
  FILE_TOO_LARGE: '文件超过本站上限',
  UNREADABLE_FILE: '文件读不出来：文本需要是 UTF-8，扫描件文档没有可读的文字',
  BAD_UPLOAD: '附件已失效，请重新上传',
  TOO_MANY_FILES: '文件太多了，删掉几个再发',
  INPUT_TOO_LONG: '内容太长了，删减一些再发吧',
  RATE_LIMITED: '操作太频繁，请稍后再试',
  CATALOG_UNAVAILABLE: '鲲 Galgame 论坛数据暂时检索不到，请稍后再试',
  MEMORY_FULL: '记忆已满，先删掉几条再添加',
  NOT_FOUND: '对话不存在',
  INVALID_REQUEST: '请求内容不符合要求',
  SERVICE_UNAVAILABLE: 'AI 助手暂不可用，请稍后再试'
}

export const agentProblemMessage = (problem: ClientProblem): string =>
  (problem.code && agentMessages[problem.code]) || problemMessage(problem)

export const isAgentDisabled = (problem: ClientProblem) =>
  problem.code === 'ASSISTANT_DISABLED' || problem.code === 'CLIENT_NOT_BOUND'

export const isAgentQuotaProblem = (problem: ClientProblem) =>
  problem.status === 402
