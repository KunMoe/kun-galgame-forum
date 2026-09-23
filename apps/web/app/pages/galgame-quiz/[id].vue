<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_CATEGORY_MAP
} from '~/constants/galgame-quiz'

const route = useRoute()
const { data } = await useKunFetch<GalgameQuizPlay>(
  `/galgame-quiz/${route.params.id}`
)

const quiz = data.value
const galgameNames = (quiz?.galgames ?? [])
  .map((g) => g.name)
  .filter(Boolean)
  .join('、')
const banner = quiz?.galgames?.[0]?.banner

const seoTitle = quiz
  ? `${truncateRunes(maskSpoilers(quiz.question, '……'), 42)}${galgameNames ? `｜${galgameNames}` : ''}`
  : 'Galgame 答题'

const seoDescription = (() => {
  if (!quiz) return '在鲲 Galgame 论坛 Galgame 题库中作答这道题目。'
  const bits = [
    `${KUN_QUIZ_TYPE_MAP[quiz.type] ?? ''}·${KUN_QUIZ_CATEGORY_MAP[quiz.category] ?? ''}`,
    `难度 ${quiz.difficulty}/10`
  ]
  if (quiz.answer_count > 0) bits.push(`${quiz.answer_count} 人已作答`)
  const prefix = galgameNames ? `关联作品《${galgameNames}》。` : ''
  return `${prefix}${truncateRunes(maskSpoilers(quiz.question, '……'), 70)} —— ${bits.join('，')}。来鲲 Galgame 论坛 Galgame 题库一起出题答题。`
})()

const seoKeywords = [
  galgameNames,
  KUN_QUIZ_CATEGORY_MAP[quiz?.category ?? 'other'],
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
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-3">
    <template v-if="data">
      <GalgameQuizPlay :quiz="data" />
      <GalgameQuizCommentCommunityContainer
        :quiz-id="data.id"
        :comment-count="data.comment_count"
      />
    </template>
    <KunNull v-else description="题目不存在或已被删除" />
  </div>
</template>
