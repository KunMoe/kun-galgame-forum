import type { Section } from '#shared/utils/api/schemas'

const KEY = 'sections'

export const useTopicSections = () => {
  const list = useKunApi(
    KEY,
    (api, { signal }) => api.GET('/sections', { signal }),
    {
      getCachedData: (key, nuxtApp) =>
        nuxtApp.payload.data[key] ?? nuxtApp.static.data[key]
    }
  )

  const sections = computed<Section[]>(() => list.data.value?.items ?? [])
  const byKey = computed(
    () =>
      new Map<string, Section>(
        sections.value.flatMap((section) => [
          [section.section, section],
          ...section.retired_sections.map((old) => [old, section] as const)
        ])
      )
  )

  const sectionOf = (key?: string | null) =>
    key ? byKey.value.get(key) : undefined
  const sectionLabel = (key?: string | null) => sectionOf(key)?.label ?? ''

  return Object.assign(list.then(() => undefined), {
    sections,
    sectionOf,
    sectionLabel,
    refresh: list.refresh
  })
}
