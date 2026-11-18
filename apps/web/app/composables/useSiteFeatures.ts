import type { SiteFeatures } from '#shared/utils/api/schemas'

// Filled once per page load by plugins/site-features; a failed read leaves
// every feature off.
export const useSiteFeatures = () => {
  const features = useState<SiteFeatures | null>(
    'kun-site-features',
    () => null
  )
  return {
    agentEnabled: computed(() => features.value?.is_agent_enabled ?? false)
  }
}
