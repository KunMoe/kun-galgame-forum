import { settle } from '#shared/utils/api/problem'
import type { Section } from '#shared/utils/api/schemas'

const KEY = 'api:sections'

export const useTopicSections = () => {
  const api = useApiClient()
  const asyncData = useAsyncData(
    KEY,
    (_app, { signal }) => settle(api.GET('/sections', { signal })),
    {
      getCachedData: (key, nuxtApp) =>
        nuxtApp.payload.data[key] ?? nuxtApp.static.data[key]
    }
  )

  const sections = computed<Section[]>(() =>
    asyncData.data.value?.ok ? asyncData.data.value.data.items : []
  )
  const byKey = computed(
    () => new Map(sections.value.map((section) => [section.section, section]))
  )

  const sectionOf = (key?: string | null) =>
    key ? byKey.value.get(key as Section['section']) : undefined
  const sectionLabel = (key?: string | null) => sectionOf(key)?.label ?? ''

  return Object.assign(asyncData.then(() => undefined), {
    sections,
    sectionOf,
    sectionLabel,
    refresh: asyncData.refresh
  })
}
