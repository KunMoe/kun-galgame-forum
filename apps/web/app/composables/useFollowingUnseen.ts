import { settle } from '#shared/utils/api/problem'
import type { FollowingActivityQuery } from '#shared/utils/api/schemas'

export const useFollowingUnseen = () => {
  const api = useApiClient()
  const hasUnseen = useState('following-activity-unseen', () => false)
  const lastMarked = useState<string>('following-activity-marked', () => '')

  const check = async (query: FollowingActivityQuery) => {
    const result = await settle(
      api.GET('/me/following-activities/summary', { params: { query } })
    )
    hasUnseen.value = result.ok && result.data.unseen_count > 0
  }

  const markSeen = async (seenAt: string) => {
    hasUnseen.value = false
    if (seenAt <= lastMarked.value) {
      return
    }
    lastMarked.value = seenAt
    await settle(
      api.PUT('/me/following-activities/read-marker', {
        body: { seen_at: seenAt }
      })
    )
  }

  return { hasUnseen, check, markSeen }
}
