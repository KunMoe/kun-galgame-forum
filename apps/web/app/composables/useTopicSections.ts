import { settle } from '#shared/utils/api/problem'
import type { Section, SectionGroupKey } from '#shared/utils/api/schemas'

export interface TopicSectionGroup {
  key: SectionGroupKey
  label: string
  sections: Section[]
}

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
  const groups = computed(() =>
    sections.value.reduce<TopicSectionGroup[]>((out, section) => {
      const last = out.at(-1)
      if (last?.key === section.group) {
        last.sections.push(section)
      } else {
        out.push({
          key: section.group,
          label: section.group_label,
          sections: [section]
        })
      }
      return out
    }, [])
  )

  const sectionOf = (key?: string | null) =>
    key ? byKey.value.get(key as Section['section']) : undefined
  const sectionLabel = (key?: string | null) => sectionOf(key)?.label ?? ''

  return Object.assign(asyncData.then(() => undefined), {
    sections,
    groups,
    sectionOf,
    sectionLabel,
    refresh: asyncData.refresh
  })
}
