<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_CATEGORY_MAP
} from '~/constants/galgame-quiz'
import type { Quiz } from '#shared/utils/api/schemas'
import { contentPlainText } from '~/utils/contentPlainText'
import { firstImageUrl } from '#shared/utils/content/plainText'
import { problemMessage } from '#shared/utils/api/message'

const route = useRoute()
const quizId = computed(() => String(route.params.id))
const workName = useWorkName()

const { data, problem } = await useApi<Quiz>(
  () => `quiz:${quizId.value}`,
  (api, { signal }) =>
    api.GET('/quizzes/{quiz_id}', {
      params: { path: { quiz_id: quizId.value } },
      signal
    })
)

const quiz = data.value
if (quiz) {
  const galgameNames = quiz.works.map((work) => workName(work)).filter(Boolean)
  const namesText = galgameNames.join('、')
  const promptText = contentPlainText(quiz.prompt).replace(/\s+/g, ' ').trim()
  const banner = quiz.works[0]?.cover?.url ?? firstImageUrl(quiz.content)

  const seoTitle = `${truncateRunes(promptText, 42)}${namesText ? `｜${namesText}` : ''}`

  const bits = [
    `${KUN_QUIZ_TYPE_MAP[quiz.quiz_type]}·${KUN_QUIZ_CATEGORY_MAP[quiz.quiz_category]}`,
    `难度 ${quiz.difficulty}/10`
  ]
  if (quiz.answer_count > 0) bits.push(`${quiz.answer_count} 人已作答`)
  const prefix = namesText ? `关联作品《${namesText}》。` : ''
  const seoDescription = `${prefix}${truncateRunes(promptText, 70)} —— ${bits.join('，')}。来鲲 Galgame 论坛 Galgame 题库一起出题答题。`

  const seoKeywords = [
    namesText,
    KUN_QUIZ_CATEGORY_MAP[quiz.quiz_category],
    'Galgame 题库',
    'galgame 答题',
    '视觉小说 题库',
    'gal 测验'
  ]
    .filter(Boolean)
    .join(',')

  useKunSeoMeta({
    title: seoTitle,
    description: seoDescription,
    keywords: seoKeywords,
    ogType: 'article',
    ...(banner ? { ogImage: banner } : {})
  })
} else {
  useKunDisableSeo('题目不存在或已被删除')
}
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-3">
    <template v-if="data">
      <GalgameQuizPlay :quiz="data" />
      <GalgameQuizCommentCommunityContainer
        :quiz-id="Number(data.id)"
        :comment-count="data.comment_count"
      />
    </template>
    <KunNull
      v-else-if="problem"
      :description="problemMessage(problem)"
    />
    <KunNull v-else description="题目不存在或已被删除" />
  </div>
</template>
