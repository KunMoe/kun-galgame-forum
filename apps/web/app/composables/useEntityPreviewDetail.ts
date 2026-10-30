import type {
  ContentDocument,
  EntityTarget,
  GalgameResource,
  Rating,
  WorkRef
} from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { catalogNameText } from '~/utils/catalogName'

const RELATED = 6

interface Intro {
  locale: string
  value: string
}

export interface EntityPreviewDetail {
  intro: string
  document: ContentDocument | null
  relatedTitle: string
  works: WorkRef[]
  topics: { id: string; title: string; created_at: string }[]
  resources: GalgameResource[]
  resource: GalgameResource | null
  rating: Rating | null
}

const pickIntro = (intros: Intro[] | undefined): string => {
  const list = (intros ?? []).filter((i) => i.value.trim())
  const zh = list.find((i) => i.locale.toLowerCase().startsWith('zh'))
  return (zh ?? list[0])?.value.trim() ?? ''
}

const empty = (): EntityPreviewDetail => ({
  intro: '',
  document: null,
  relatedTitle: '',
  works: [],
  topics: [],
  resources: [],
  resource: null,
  rating: null
})

export const useEntityPreviewDetail = (target: () => EntityTarget | null) => {
  const api = useApiClient()
  const { allowsNsfw } = useContentStance()
  const { showKUNGalgamePreferOriginalName } = storeToRefs(
    usePersistSettingsStore()
  )
  const detail = ref<EntityPreviewDetail>(empty())
  const status = ref<'idle' | 'loading' | 'done'>('idle')

  const load = async (t: EntityTarget, signal: AbortSignal) => {
    const out = empty()
    const include_nsfw = allowsNsfw.value
    const opts = { signal }
    switch (t.object) {
      case 'work': {
        const [w, related] = await Promise.all([
          settle(
            api.GET('/works/{work_id}', {
              params: { path: { work_id: t.id } },
              ...opts
            })
          ),
          t.maker
            ? settle(
                api.GET('/companies/{company_id}/works', {
                  params: {
                    path: { company_id: t.maker.id },
                    query: { limit: RELATED + 1, include_nsfw }
                  },
                  ...opts
                })
              )
            : null
        ])
        out.intro = w.ok ? pickIntro(w.data.intros) : ''
        if (t.maker && related?.ok) {
          out.relatedTitle = `${catalogNameText(t.maker, showKUNGalgamePreferOriginalName.value)}的其他作品`
          out.works = related.data.items
            .map((i) => i.work_summary)
            .filter((w) => w.id !== t.id)
            .slice(0, RELATED)
        }
        break
      }
      case 'character': {
        const [c, related] = await Promise.all([
          settle(
            api.GET('/characters/{character_id}', {
              params: { path: { character_id: t.id }, query: { include_nsfw } },
              ...opts
            })
          ),
          settle(
            api.GET('/characters/{character_id}/appearances', {
              params: {
                path: { character_id: t.id },
                query: { limit: RELATED, include_nsfw }
              },
              ...opts
            })
          )
        ])
        out.intro = c.ok ? pickIntro(c.data.intros) : ''
        out.relatedTitle = '登场作品'
        out.works = related.ok
          ? related.data.items.map((i) => i.work_summary)
          : []
        break
      }
      case 'credit_name': {
        const [n, related] = await Promise.all([
          settle(
            api.GET('/credit-names/{credit_name_id}', {
              params: { path: { credit_name_id: t.id } },
              ...opts
            })
          ),
          settle(
            api.GET('/credit-names/{credit_name_id}/credits', {
              params: {
                path: { credit_name_id: t.id },
                query: { limit: RELATED, include_nsfw }
              },
              ...opts
            })
          )
        ])
        out.intro = n.ok ? pickIntro(n.data.intros) : ''
        out.relatedTitle = '参与作品'
        out.works = related.ok
          ? related.data.items.map((i) => i.work_summary)
          : []
        break
      }
      case 'company': {
        const [c, related] = await Promise.all([
          settle(
            api.GET('/companies/{company_id}', {
              params: { path: { company_id: t.id } },
              ...opts
            })
          ),
          settle(
            api.GET('/companies/{company_id}/works', {
              params: {
                path: { company_id: t.id },
                query: { limit: RELATED, include_nsfw }
              },
              ...opts
            })
          )
        ])
        out.intro = c.ok ? pickIntro(c.data.intros) : ''
        out.relatedTitle = '代表作品'
        out.works = related.ok
          ? related.data.items.map((i) => i.work_summary)
          : []
        break
      }
      case 'tag': {
        const [g, related] = await Promise.all([
          settle(
            api.GET('/tags/{tag_id}', {
              params: { path: { tag_id: t.id }, query: { include_nsfw } },
              ...opts
            })
          ),
          settle(
            api.GET('/tags/{tag_id}/works', {
              params: {
                path: { tag_id: t.id },
                query: { limit: RELATED, include_nsfw }
              },
              ...opts
            })
          )
        ])
        out.intro = g.ok ? pickIntro(g.data.intros) : ''
        out.relatedTitle = '带有这个标签的作品'
        out.works = related.ok ? related.data.items : []
        break
      }
      case 'series': {
        const [s, related] = await Promise.all([
          settle(
            api.GET('/series/{series_id}', {
              params: { path: { series_id: t.id } },
              ...opts
            })
          ),
          settle(
            api.GET('/series/{series_id}/works', {
              params: {
                path: { series_id: t.id },
                query: { limit: RELATED, include_nsfw }
              },
              ...opts
            })
          )
        ])
        out.intro = s.ok ? pickIntro(s.data.intros) : ''
        out.relatedTitle = '系列作品'
        out.works = related.ok ? related.data.items : []
        break
      }
      case 'rating': {
        const r = await settle(
          api.GET('/ratings/{rating_id}', {
            params: { path: { rating_id: t.id } },
            ...opts
          })
        )
        out.rating = r.ok ? r.data : null
        out.relatedTitle = '评分的作品'
        out.works = t.work ? [t.work] : []
        break
      }
      case 'topic': {
        const [topic, related] = await Promise.all([
          settle(
            api.GET('/topics/{topic_id}', {
              params: { path: { topic_id: t.id } },
              ...opts
            })
          ),
          settle(
            api.GET('/users/{user_id}/topics', {
              params: {
                path: { user_id: t.author.id },
                query: {
                  relation: 'authored',
                  page: 1,
                  limit: RELATED + 1,
                  include_nsfw
                }
              },
              ...opts
            })
          )
        ])
        out.document = topic.ok ? topic.data.content : null
        out.relatedTitle = '作者的其它话题'
        out.topics = related.ok
          ? related.data.items.filter((i) => i.id !== t.id).slice(0, RELATED)
          : []
        break
      }
      case 'galgame_resource': {
        const [r, related] = await Promise.all([
          settle(
            api.GET('/galgame-resources/{resource_id}', {
              params: { path: { resource_id: t.id } },
              ...opts
            })
          ),
          t.work
            ? settle(
                api.GET('/works/{work_id}/resources', {
                  params: {
                    path: { work_id: t.work.id },
                    query: { limit: RELATED + 1 }
                  },
                  ...opts
                })
              )
            : null
        ])
        out.resource = r.ok ? r.data : null
        out.relatedTitle = '同一作品的其它资源'
        out.resources = related?.ok
          ? related.data.items.filter((i) => i.id !== t.id).slice(0, RELATED)
          : []
        break
      }
      default:
        break
    }
    return out
  }

  let controller: AbortController | null = null
  watch(
    target,
    async (t) => {
      controller?.abort()
      detail.value = empty()
      if (!t || t.object === 'unavailable_entity' || t.object === 'user') {
        status.value = 'done'
        return
      }
      controller = new AbortController()
      const { signal } = controller
      status.value = 'loading'
      const next = await load(t, signal)
      if (!signal.aborted) {
        detail.value = next
        status.value = 'done'
      }
    },
    { immediate: true }
  )
  onBeforeUnmount(() => controller?.abort())

  return { detail, status }
}
