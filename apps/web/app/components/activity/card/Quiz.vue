<script setup lang="ts">
import { kunQuizDifficultyLabel } from '~/constants/galgame-quiz'
import { settle } from '#shared/utils/api/problem'

import type { Activity, ActivityQuiz } from '#shared/utils/api/schemas'

const props = defineProps<{ activity: Activity; quiz: ActivityQuiz }>()
const api = useApiClient()

const quizId = computed(() => String(props.quiz.quiz_id))

const summary = computed(
  () =>
    `出了一道题目，${kunQuizDifficultyLabel(props.quiz.difficulty)}·难度${props.quiz.difficulty}，已经有 ${props.quiz.answer_count} 人作答。`
)

const descriptionText = computed(() =>
  markdownToText(props.quiz.description_excerpt)
)

const { isFavorited, setFavorited, ensureLoaded } = useMyQuizInteractions()
onMounted(() => ensureLoaded(quizId.value ? [quizId.value] : []))

const toggleFavorite = async (next: boolean) => {
  if (!quizId.value) return false
  const options = { params: { path: { quiz_id: quizId.value } } }
  const result = await settle(
    next
      ? api.PUT('/quizzes/{quiz_id}/favorite', options)
      : api.DELETE('/quizzes/{quiz_id}/favorite', options)
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  setFavorited(quizId.value, result.data.viewer?.has_favorited ?? next)
  return true
}
</script>

<template>
  <ActivityCardShell
    :performer="activity.performer"
    :occurred-at="activity.occurred_at"
  >
    <div class="space-y-2">
      <p class="text-default-600 text-sm">{{ summary }}</p>

      <KunLink
        underline="none"
        color="default"
        :to="activity.path"
        class-name="group block"
      >
        <p
          class="group-hover:text-primary line-clamp-3 text-base break-words transition-colors"
        >
          {{ maskSpoilers(activity.excerpt_markdown) }}
        </p>
      </KunLink>

      <p
        v-if="descriptionText"
        class="text-default-500 text-sm break-words whitespace-pre-line"
      >
        {{ descriptionText }}
      </p>

      <div class="flex items-center gap-2">
        <FavoriteToggle
          :favorited="isFavorited(quizId)"
          :count="quiz.favorite_count"
          :action="toggleFavorite"
          size="sm"
        />
        <KunLink
          underline="none"
          color="default"
          :to="activity.path"
          class-name="text-default-500 hover:text-primary ml-auto flex items-center gap-0.5 text-sm"
        >
          查看详情
          <KunIcon name="lucide:chevron-right" class="size-4" />
        </KunLink>
      </div>
    </div>
  </ActivityCardShell>
</template>
