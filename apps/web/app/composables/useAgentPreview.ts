import { settle } from '#shared/utils/api/problem'
import { agentProblemMessage } from '#shared/utils/api/agent'
import { entityPreviewQuery } from '~/composables/useEntitySearch'
import { agentCardEntity } from '~/utils/agentCards'

export const useAgentPreview = () => {
  const api = useApiClient()
  const panel = useEntityPreviewPanel()

  const open = async (card: AgentEntityCard) => {
    const ref = agentCardEntity(card)
    if (!ref) return
    const r = await settle(
      api.GET('/entity-previews', {
        params: { query: entityPreviewQuery([ref]) }
      })
    )
    if (!r.ok) {
      useMessage(agentProblemMessage(r.problem), 'warn')
      return
    }
    const target = r.data.items[0]
    if (target) panel.open(target)
  }

  return { open }
}
