import { createChatApi, type ChatApi } from '#shared/utils/api/chat'

let client: ChatApi | undefined

export const useChatApi = (): ChatApi => {
  client ??= createChatApi(useRuntimeConfig().public.apiBaseUrl as string)
  return client
}
