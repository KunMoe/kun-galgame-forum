import { settle } from '#shared/utils/api/problem'
import type { SiteFeatures } from '#shared/utils/api/schemas'

export default defineNuxtPlugin(async () => {
  const features = useState<SiteFeatures | null>(
    'kun-site-features',
    () => null
  )
  if (features.value) return

  const result = await settle(useApiClient().GET('/site/features'))
  if (result.ok) {
    features.value = result.data
  }
})
