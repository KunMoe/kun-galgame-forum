import { Centrifuge, UnauthorizedError } from 'centrifuge'
import type { ChatPush, ChatRealtimeToken } from '#shared/types/chat'

export default defineNuxtPlugin(() => {
  if (!useRuntimeConfig().public.chatEnabled) {
    return
  }
  const user = usePersistUserStore()
  const chat = useChatStore()
  const api = useChatApi()
  let client: Centrifuge | null = null

  const token = async () => {
    const r = await api.post<ChatRealtimeToken>('/realtime-token')
    if (r.ok) {
      return r.data
    }
    chat.failed(r.problem)
    if (r.problem.status === 401 || r.problem.status === 403) {
      throw new UnauthorizedError(r.problem.code ?? 'unauthorized')
    }
    throw new Error(r.problem.code ?? r.problem.kind)
  }

  const connect = async (uid: number) => {
    await chat.start(String(uid))
    if (chat.status !== 'ready') {
      return
    }
    const first = await token().catch(() => null)
    if (!first || user.id !== uid) {
      return
    }
    client = new Centrifuge(first.url, {
      token: first.token,
      getToken: async () => (await token()).token
    })
    client.on('publication', (ctx) => chat.push(ctx.data as ChatPush))
    client.on('connected', () => void chat.sync())
    client.connect()
  }

  const disconnect = () => {
    client?.disconnect()
    client = null
    chat.stop()
  }

  onNuxtReady(() => {
    watch(
      () => user.id,
      (id) => {
        disconnect()
        if (id) {
          void connect(id)
        }
      },
      { immediate: true }
    )
  })

  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState !== 'visible' || !user.id) {
      return
    }
    if (chat.status === 'ready') {
      void chat.sync()
    } else if (chat.status === 'down') {
      disconnect()
      void connect(user.id)
    }
  })
})
