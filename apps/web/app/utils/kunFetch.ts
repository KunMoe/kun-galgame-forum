export interface KunApiFieldError {
  pointer?: string
  parameter?: string
  header?: string
  reason?: string
  detail?: string
}

interface KunApiResponse<T> {
  code: number
  message: string
  data: T
  errors?: KunApiFieldError[]
}

// Returning true claims the rejection: the default toast is skipped because the
// caller is rendering the message itself. The edit form needs this — the engine
// names the field it refused, and one toast for "第 1 行 标题：必填" is strictly
// worse than the message sitting under that control.
type KunApiErrorHook = (envelope: KunApiResponse<unknown>) => boolean

const SSR_API_TIMEOUT_MS = 10000

const CODE_AUTH_EXPIRED = 205
const CODE_BANNED = 234
const CODE_REAUTH_REQUIRED = 235
const STATUS_RATE_LIMITED = 429

const LOGIN_REQUIRED_CODES = new Set([
  10115, 10146, 10216, 10220, 10228, 10232, 10235, 10237, 10240, 10249, 10529,
  10532, 10546
])

const SSR_FORWARDED_COOKIES = ['kungal_session', 'KUNGalgameSettings']

const extractForwardedCookies = (cookieHeader?: string): string | undefined => {
  if (!cookieHeader) return undefined
  const kept: string[] = []
  for (const part of cookieHeader.split(';')) {
    const trimmed = part.trim()
    if (SSR_FORWARDED_COOKIES.some((name) => trimmed.startsWith(`${name}=`))) {
      kept.push(trimmed)
    }
  }
  return kept.length > 0 ? kept.join('; ') : undefined
}

let authExpiryTimer: ReturnType<typeof setTimeout> | null = null

// A 429 carries no forum envelope — it is refused before a handler runs — so it
// fell through to 「网络请求失败，请稍后重试」 on the kunFetch path and to
// nothing at all on the useKunFetch one, whose onResponseError only speaks when
// there is an envelope to read. A reader who hit the limit posting a reply saw
// their reply not appear and no reason why; the 429 was in the console.
// A non-JSON error body (a proxy's 429, an nginx 502 page) parses into a
// STRING, not an object — and a string is truthy with `code === undefined`, so
// `resp && resp.code !== 0` was true and the envelope path ran with
// `code=undefined, message=undefined`, ending in `useMessage(undefined)`: a
// toast with no text. That is what 「回复报错但没有提示，控制台查看才有」 was.
const asEnvelope = (data: unknown): KunApiResponse<unknown> | undefined =>
  data !== null && typeof data === 'object' && typeof (data as { code?: unknown }).code === 'number'
    ? (data as KunApiResponse<unknown>)
    : undefined

const handleRateLimited = (status: number | undefined) => {
  if (import.meta.server || status !== STATUS_RATE_LIMITED) {
    return false
  }
  useMessage('操作太频繁，请稍等一会儿再试', 'error', 5000)
  return true
}

const handleApiError = async (code: number, message: string) => {
  if (import.meta.server) return

  if (code === CODE_BANNED) {
    const userStore = usePersistUserStore()
    if (userStore.id) {
      userStore.resetUser()
    }
    useMessage(message || '您的账号已被封禁', 'error', 10000)
    return
  }

  if (code === CODE_REAUTH_REQUIRED) {
    useMessage(message || '请退出登录后重新登录以授予该权限', 'error', 10000)
    return
  }

  if (code === CODE_AUTH_EXPIRED) {
    const userStore = usePersistUserStore()
    if (!userStore.id || authExpiryTimer) {
      return
    }
    const nuxtApp = useNuxtApp()
    const config = useRuntimeConfig()
    authExpiryTimer = setTimeout(async () => {
      authExpiryTimer = null
      const store = usePersistUserStore()
      if (!store.id) {
        return
      }
      let dead = false
      try {
        const resp = await $fetch<KunApiResponse<unknown>>(
          `${config.public.apiBaseUrl}/api/user/status`,
          { credentials: 'include' }
        )
        dead = !resp || resp.code !== 0
      } catch (e) {
        const status =
          (e as { status?: number; response?: { status?: number } })?.status ??
          (e as { response?: { status?: number } })?.response?.status
        dead = status === 401 || status === 403
      }
      if (!dead || !store.id) {
        return
      }
      store.resetUser()
      useMessage(message || '登录已失效，请重新登录', 'error', 7777)
      nuxtApp.runWithContext(() => navigateTo('/'))
    }, 1500)
    return
  }

  if (LOGIN_REQUIRED_CODES.has(code) && !usePersistUserStore().id) {
    useAuthModal().open()
    return
  }

  if (code !== 0) {
    useMessage(message, 'error')
  }
}

export const useKunFetch = createUseFetch({
  timeout: import.meta.server ? SSR_API_TIMEOUT_MS : undefined,
  credentials: 'include',
  onRequest({ options }) {
    const config = useRuntimeConfig()
    options.baseURL = `${
      import.meta.server ? config.apiBaseUrl : config.public.apiBaseUrl
    }/api`
    if (import.meta.server) {
      const forwarded = extractForwardedCookies(
        useRequestHeaders(['cookie']).cookie
      )
      if (forwarded) {
        const merged = new Headers(options.headers as HeadersInit | undefined)
        merged.set('cookie', forwarded)
        options.headers = merged
      }
    }
  },
  async onResponseError({ response }) {
    const resp = asEnvelope(response._data)
    if (resp && resp.code !== 0) {
      await handleApiError(resp.code, resp.message)
      return
    }
    handleRateLimited(response.status)
  },
  transform(resp: unknown) {
    const envelope = resp as KunApiResponse<unknown> | null | undefined
    if (!envelope || envelope.code !== 0) {
      return null
    }
    return envelope.data !== undefined ? envelope.data : envelope.message
  }
})

export const kunFetch = async <T>(
  url: string,
  options?: Record<string, unknown>
): Promise<T | null> => {
  const config = useRuntimeConfig()
  const apiBase = import.meta.server
    ? `${config.apiBaseUrl}/api`
    : `${config.public.apiBaseUrl}/api`

  const { onApiError, ...fetchOptions } = (options ?? {}) as {
    onApiError?: KunApiErrorHook
  } & Record<string, unknown>
  const reject = async (envelope: KunApiResponse<unknown>) => {
    if (onApiError?.(envelope)) {
      return
    }
    await handleApiError(envelope.code, envelope.message)
  }

  const headers = new Headers(
    (options as { headers?: HeadersInit } | undefined)?.headers
  )
  if (import.meta.server) {
    const forwarded = extractForwardedCookies(
      useRequestHeaders(['cookie']).cookie
    )
    if (forwarded) {
      headers.set('cookie', forwarded)
    }
  }

  try {
    const resp = await $fetch<KunApiResponse<T>>(`${apiBase}${url}`, {
      timeout: import.meta.server ? SSR_API_TIMEOUT_MS : undefined,
      credentials: 'include',
      ...fetchOptions,
      headers
    })

    if (!resp || resp.code !== 0) {
      const envelope = asEnvelope(resp)
      if (envelope) {
        await reject(envelope)
      }
      return null
    }

    return resp.data !== undefined ? resp.data : (resp.message as T)
  } catch (error) {
    if (import.meta.client) {
      const err = error as {
        data?: KunApiResponse<unknown>
        status?: number
        statusCode?: number
        response?: { status?: number; _data?: KunApiResponse<unknown> }
      }
      const envelope = asEnvelope(err.data ?? err.response?._data)
      const status = err.status ?? err.statusCode ?? err.response?.status

      if (envelope && envelope.code !== 0) {
        await reject(envelope)
      } else if (handleRateLimited(status)) {
        // said already
      } else if (status === 401 || status === 403) {
        await handleApiError(CODE_AUTH_EXPIRED, '登录已失效，请重新登录')
      } else {
        useMessage('网络请求失败，请稍后重试', 'error')
      }
    }
    return null
  }
}
