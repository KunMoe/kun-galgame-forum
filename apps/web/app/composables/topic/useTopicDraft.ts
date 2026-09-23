import { useTopicEditorStore } from './useTopicEditorStore'
import { coverHashFromToken, coverTokenFromHash } from './applyTopicSource'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import { settle } from '#shared/utils/api/problem'
import type {
  TopicDraftCreate,
  TopicDraftSummary
} from '#shared/utils/api/schemas'

export const useTopicDraft = () => {
  const { title, content, category, section, isNSFW, coverImages } =
    useTopicEditorStore()
  const api = useApiClient()
  const createKey = useIdempotencyKey()

  const listDrafts = async (): Promise<TopicDraftSummary[]> => {
    const drafts: TopicDraftSummary[] = []
    let cursor: string | undefined
    do {
      const page = await settle(
        api.GET('/me/topic-drafts', {
          params: { query: { limit: 30, cursor } }
        })
      )
      if (!page.ok) {
        reportProblem(page.problem)
        return drafts
      }
      drafts.push(...page.data.items)
      cursor = page.data.next_cursor
    } while (cursor)
    return drafts
  }

  const saveCurrentAsDraft = async () => {
    const body: TopicDraftCreate = {
      title: title.value,
      content_markdown: content.value,
      sections: section.value as TopicDraftCreate['sections'],
      is_nsfw: isNSFW.value,
      cover_image_hashes: coverImages.value.map(coverHashFromToken)
    }
    if (category.value) {
      body.category = category.value as TopicDraftCreate['category']
    }
    const saved = await settle(
      api.POST('/me/topic-drafts', {
        params: {
          header: {
            'Idempotency-Key': createKey.take('/me/topic-drafts', body)
          }
        },
        body
      })
    )
    if (!saved.ok) {
      reportProblem(saved.problem)
      return false
    }
    createKey.clear()
    return true
  }

  const loadDraft = async (id: string) => {
    const draft = await settle(
      api.GET('/me/topic-drafts/{draft_id}', {
        params: { path: { draft_id: id } }
      })
    )
    if (!draft.ok) {
      reportProblem(draft.problem)
      return false
    }
    const d = draft.data
    title.value = d.title
    content.value = d.content_markdown
    category.value = d.category ?? ''
    section.value = [...d.sections]
    isNSFW.value = d.is_nsfw
    coverImages.value = d.cover_image_hashes.map(coverTokenFromHash)
    return true
  }

  const deleteDraft = async (id: string) => {
    const deleted = await settle(
      api.DELETE('/me/topic-drafts/{draft_id}', {
        params: { path: { draft_id: id } }
      })
    )
    if (!deleted.ok) {
      reportProblem(deleted.problem)
    }
    return deleted.ok
  }

  const hasCurrentContent = computed(
    () => !!(title.value.trim() || content.value.trim())
  )

  return {
    listDrafts,
    saveCurrentAsDraft,
    loadDraft,
    deleteDraft,
    hasCurrentContent
  }
}
