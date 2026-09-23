import { settle } from '#shared/utils/api/problem'

export default defineNuxtPlugin(async () => {
  const mine = useState<string[] | null>('kun-perm-mine', () => null)

  if (mine.value) return

  const { id } = usePersistUserStore()
  if (!id) return

  const result = await settle(useApiClient().GET('/me/permissions'))
  if (result.ok) {
    mine.value = result.data.permissions
  }
})
