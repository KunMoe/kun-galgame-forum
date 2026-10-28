import type { ShallowRef } from 'vue'
import type { EntityTarget } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import type { EntityLinkRef, EntityRefType } from '~/utils/entityUrl'
import {
  entityPreviewQuery,
  orderForPreview
} from '~/composables/useEntitySearch'

type PreviewState = EntityTarget | 'loading' | 'failed'
type Key = `${EntityRefType}:${string}`

const BATCH = 50

const cache = new Map<Key, ShallowRef<PreviewState>>()
let queued: { type: EntityRefType; id: string }[] = []
let scheduled = false

const flush = async (api: ReturnType<typeof useApiClient>) => {
  scheduled = false
  const batch = queued
  queued = []
  for (let i = 0; i < batch.length; i += BATCH) {
    const chunk = batch.slice(i, i + BATCH)
    const ordered = orderForPreview(chunk)
    const query = entityPreviewQuery(ordered)
    const result = await settle(
      api.GET('/entity-previews', { params: { query } })
    )
    ordered.forEach((r, index) => {
      const state = cache.get(`${r.type}:${r.id}`)
      if (state) {
        state.value = result.ok
          ? (result.data.items[index] ?? 'failed')
          : 'failed'
      }
    })
  }
}

export const useEntityPreview = (ref: () => EntityLinkRef) => {
  const api = useApiClient()
  const key = computed<Key>(() => `${ref().type}:${ref().id}`)

  watch(
    key,
    (k) => {
      if (cache.has(k)) {
        return
      }
      cache.set(k, shallowRef<PreviewState>('loading'))
      const { type, id } = ref()
      queued.push({ type, id })
      if (!scheduled) {
        scheduled = true
        queueMicrotask(() => void flush(api))
      }
    },
    { immediate: true }
  )

  return computed<PreviewState>(() => cache.get(key.value)?.value ?? 'loading')
}
