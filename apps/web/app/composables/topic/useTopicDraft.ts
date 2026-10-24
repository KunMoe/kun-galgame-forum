import { useTopicEditorStore } from './useTopicEditorStore'
import { coverHashFromToken, coverTokenFromHash } from './applyTopicSource'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import {
  lotteryFormFromDraft,
  lotteryFormToDraft,
  lotteryHasCodes,
  pollFormFromDraft,
  pollFormToDraft
} from './pendingMiniApps'
import { settle } from '#shared/utils/api/problem'
import type {
  TopicDraftCreate,
  TopicDraftSummary
} from '#shared/utils/api/schemas'

export const useTopicDraft = () => {
  const {
    title,
    content,
    section,
    isNSFW,
    coverImages,
    polls,
    lotteries
  } = useTopicEditorStore()
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
      cover_image_hashes: coverImages.value.map(coverHashFromToken),
      polls: polls.value.map(pollFormToDraft),
      lotteries: lotteries.value.map(lotteryFormToDraft)
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
    section.value = d.sections.slice(0, 1)
    isNSFW.value = d.is_nsfw
    coverImages.value = d.cover_image_hashes.map(coverTokenFromHash)
    polls.value = d.polls.map(pollFormFromDraft)
    lotteries.value = d.lotteries.map(lotteryFormFromDraft)
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

  const hasMiniApps = computed(
    () => polls.value.length + lotteries.value.length > 0
  )

  const hasUnsavableCodes = computed(() =>
    lotteries.value.some(lotteryHasCodes)
  )

  return {
    listDrafts,
    saveCurrentAsDraft,
    loadDraft,
    deleteDraft,
    hasCurrentContent,
    hasMiniApps,
    hasUnsavableCodes
  }
}
