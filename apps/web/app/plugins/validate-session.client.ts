import { settle } from '#shared/utils/api/problem'
import { handleBannedAccount, probeSessionExpiry } from '~/utils/kunFetch'

export default defineNuxtPlugin((nuxtApp) => {
  const userStore = usePersistUserStore()
  if (!userStore.id) return

  nuxtApp.runWithContext(() => {
    const api = useApiClient()
    void settle(api.GET('/me')).then((result) => {
      if (result.ok) return
      if (result.problem.code === 'ACCOUNT_BANNED') {
        handleBannedAccount()
      } else if (result.problem.status === 401) {
        probeSessionExpiry()
      }
    })
    useRefreshMe().refreshMe()
  })
})
