import { createTopicSchema } from '~/validations/topic'
import { useTopicEditorStore } from './useTopicEditorStore'
import { coverHashFromToken } from './applyTopicSource'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import { usePoll } from './usePoll'
import { formToCreate, useLottery } from './useLottery'
import {
  clonePlain,
  lotteryPointBudget,
  pendingLotteryIssue,
  pendingPollIssue,
  pollFormToCreate
} from './pendingMiniApps'
import type { HeldMiniApp } from '~/store/temp/topic/miniAppHandoff'
import { settle } from '#shared/utils/api/problem'
import type {
  TopicClassification,
  TopicCreate,
  TopicPatch
} from '#shared/utils/api/schemas'

export const useTopicSubmitter = () => {
  const {
    section,
    title,
    content,
    isNSFW,
    coverImages,
    accessScope,
    accessRoles,
    accessUserIds,
    polls,
    lotteries
  } = useTopicEditorStore()
  const tempStore = useTempEditStore()
  const persistStore = usePersistEditTopicStore()
  const { moemoepoint } = usePersistUserStore()
  const { sectionOf } = useTopicSections()
  const api = useApiClient()
  const createKey = useIdempotencyKey()
  const handoff = useTempTopicMiniAppHandoffStore()
  const createdTopicId = ref('')
  const { createPoll } = usePoll(createdTopicId)
  const { createLottery } = useLottery(createdTopicId)

  const rules = reactive({
    isReadRule: false,
    isValidTitle: false,
    isKnownConsequence: false
  })
  const isSubmitting = ref(false)
  const isRewriteMode = computed(() => tempStore.isTopicRewriting)

  const classification = ref<TopicClassification | null>(null)
  const isClassifying = ref(false)
  const classifyIssue = ref<'empty' | 'limited' | 'failed' | null>(null)

  const classify = async () => {
    classification.value = null
    classifyIssue.value = null
    if (!title.value.trim() || !content.value.trim()) {
      classifyIssue.value = 'empty'
      return
    }
    isClassifying.value = true
    const result = await settle(
      api.POST('/topics/classify', {
        body: { title: title.value, content_markdown: content.value }
      })
    )
    isClassifying.value = false
    if (result.ok) {
      classification.value = result.data
      section.value = [result.data.section]
    } else {
      classifyIssue.value = result.problem.status === 429 ? 'limited' : 'failed'
    }
  }

  const writePayload = (mode: 'create' | 'rewrite') => {
    const hashes = coverImages.value.map(coverHashFromToken)
    const body: TopicCreate = {
      title: title.value,
      content_markdown: content.value,
      sections: section.value as TopicCreate['sections'],
      is_nsfw: isNSFW.value,
      access_scope: accessScope.value
    }
    if (mode === 'rewrite') {
      body.cover_image_hashes = hashes
    } else if (hashes.length > 0) {
      body.cover_image_hashes = hashes
    }
    if (mode === 'create' && classification.value) {
      body.sections = [classification.value.section]
      body.classification_token = classification.value.classification_token
    }
    if (accessScope.value === 'role') {
      body.access_roles = accessRoles.value
    }
    if (accessScope.value === 'users') {
      body.access_user_ids = accessUserIds.value.map(String)
    }
    return body
  }

  const checkPendingMiniApps = () => {
    const issue = [
      ...polls.value.map(pendingPollIssue),
      ...lotteries.value.map(pendingLotteryIssue)
    ].find(Boolean)
    if (issue) {
      useMessage(issue, 'warn')
      return false
    }
    const budget = lotteries.value.reduce(
      (sum, lottery) => sum + lotteryPointBudget(lottery),
      0
    )
    if (budget > moemoepoint) {
      useMessage(
        `抽奖的萌萌点奖池共需 ${budget} 萌萌点, 超过了您当前的 ${moemoepoint} 萌萌点`,
        'warn'
      )
      return false
    }
    return true
  }

  const createMiniApps = async (topicId: string, items: HeldMiniApp[]) => {
    createdTopicId.value = topicId
    const failed: HeldMiniApp[] = []
    for (const item of items) {
      const result =
        item.key === 'poll'
          ? await createPoll(pollFormToCreate(item.form))
          : await createLottery(formToCreate(item.form))
      if (!result.ok) {
        reportProblem(result.problem)
        failed.push(item)
      }
    }
    return failed
  }

  const submit = async () => {
    if (isSubmitting.value) {
      return
    }

    const isReadAllRules = Object.values(rules).every((value) => value)
    if (moemoepoint < 50 && !isReadAllRules) {
      useMessage('请勾选同意所有发布须知后再发布话题', 'warn')
      return
    }

    const mode = isRewriteMode.value ? 'rewrite' : 'create'
    const payload = writePayload(mode)
    const result = createTopicSchema.safeParse(payload)
    if (!result.success) {
      const error = JSON.parse(result.error.message)[0]
      useMessage(formatKunZodIssue(error), 'warn')
      return
    }

    if (mode === 'create' && !checkPendingMiniApps()) {
      return
    }

    const chosen = sectionOf(payload.sections[0])
    if (!chosen || (chosen.is_archived && mode === 'create')) {
      useMessage('请选择话题的分区', 'warn')
      return
    }
    if (moemoepoint < -chosen.moemoepoint_on_create) {
      useMessage(
        `您没有足够的萌萌点来发布求助或者寻求资源的话题, 您可以通过发布 Galgame, 签到, 接受别人的赞赏, 等等来获取萌萌点`,
        'warn'
      )
      return
    }

    isSubmitting.value = true
    try {
      if (mode === 'rewrite') {
        const topicId = String(tempStore.id)
        const patched = await settle(
          api.PATCH('/topics/{topic_id}', {
            params: { path: { topic_id: topicId } },
            body: payload as TopicPatch
          })
        )
        if (!patched.ok) {
          reportProblem(patched.problem)
          return
        }
        useKunLoliInfo('重新编辑成功', 5)
        tempStore.resetRewriteTopicData()
        await navigateTo(`/topic/${patched.data.id}`)
        return
      }

      const created = await settle(
        api.POST('/topics', {
          params: {
            header: { 'Idempotency-Key': createKey.take('/topics', payload) }
          },
          body: payload
        })
      )
      if (!created.ok) {
        reportProblem(created.problem)
        return
      }
      createKey.clear()
      // Reset before the mini-apps go out: closing the tab halfway must not
      // leave the editor full, one click away from publishing a second topic.
      const miniApps: HeldMiniApp[] = [
        ...polls.value.map((form) => ({ key: 'poll' as const, form })),
        ...lotteries.value.map((form) => ({ key: 'lottery' as const, form }))
      ].map(clonePlain)
      persistStore.resetTopicData()
      const failed = await createMiniApps(created.data.id, miniApps)
      if (failed.length) {
        handoff.hold(created.data.id, failed)
        useMessage(
          `话题已发布, 但有 ${failed.length} 个投票或抽奖没有创建成功, 可以在话题页重新创建`,
          'warn',
          8000
        )
      } else {
        useKunLoliInfo('发布成功', 5)
      }
      await navigateTo(`/topic/${created.data.id}`)
    } finally {
      isSubmitting.value = false
    }
  }

  return {
    rules,
    submit,
    isSubmitting,
    isRewriteMode,
    classification,
    isClassifying,
    classifyIssue,
    classify
  }
}
