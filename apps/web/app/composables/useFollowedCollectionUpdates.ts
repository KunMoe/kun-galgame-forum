import { settle } from '#shared/utils/api/problem'

export const useFollowedCollectionUpdates = () => {
  const count = useState('followed-collection-updates', () => 0)
  const api = useApiClient()

  const refresh = async () => {
    if (!usePersistUserStore().id || import.meta.server) {
      return
    }
    const result = await settle(api.GET('/me/followed-collections/updates'))
    if (result.ok) {
      count.value = result.data.updated_count
    }
  }

  return { count, refresh }
}
