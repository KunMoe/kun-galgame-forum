import { createAgentApi, type AgentApi } from '#shared/utils/api/agent'

let client: AgentApi | undefined

export const useAgentApi = (): AgentApi => {
  client ??= createAgentApi(useRuntimeConfig().public.apiBaseUrl as string)
  return client
}
