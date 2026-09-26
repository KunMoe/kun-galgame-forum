<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_TYPE_ICON_MAP,
  KUN_QUIZ_CATEGORY_MAP,
  KUN_QUIZ_SPOILER_MAP,
  KUN_QUIZ_SPOILER_COLOR_MAP,
  kunQuizDifficultyLabel,
  kunQuizDifficultyColor
} from '~/constants/galgame-quiz'
import type { QuizSummary } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

defineProps<{ quizzes: QuizSummary[] }>()

const authorOf = (quiz: QuizSummary) => toKunUser(quiz.author)

const correctRate = (q: QuizSummary) =>
  q.answer_count > 0
    ? Math.round((q.correct_count / q.answer_count) * 100)
    : null
</script>

<template>
  <div class="divide-default-200 divide-y">
    <NuxtLink
      v-for="quiz in quizzes"
      :key="quiz.id"
      :to="`/galgame-quiz/${quiz.id}`"
      class="hover:bg-default-100 flex items-center gap-3 px-2 py-3 transition-colors"
    >
      <div class="flex w-6 shrink-0 justify-center text-lg">
        <KunIcon
          v-if="quiz.viewer?.is_correct === true"
          name="lucide:circle-check"
          class="text-success"
        />
        <KunIcon
          v-else-if="quiz.viewer?.is_correct === false"
          name="lucide:circle-x"
          class="text-danger"
        />
        <KunIcon
          v-else-if="quiz.viewer?.has_answered"
          name="lucide:circle-check"
          class="text-default-400"
        />
        <KunIcon v-else name="lucide:circle-dashed" class="text-default-400" />
      </div>

      <div class="min-w-0 flex-1">
        <ContentDocument
          :document="quiz.prompt"
          compact
          class-name="font-medium break-words"
        />
        <div
          class="text-default-500 mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs"
        >
          <span class="text-default-700 flex items-center gap-1">
            <KunAvatar
              :user="authorOf(quiz)"
              size="xs"
              :is-navigation="false"
            />
            {{ authorOf(quiz).name }}
          </span>
          <KunTime :time="quiz.bumped_at" />
          <span class="flex items-center gap-1">
            <KunIcon :name="KUN_QUIZ_TYPE_ICON_MAP[quiz.quiz_type]" />
            {{ KUN_QUIZ_TYPE_MAP[quiz.quiz_type] }}
          </span>
          <span>{{ KUN_QUIZ_CATEGORY_MAP[quiz.quiz_category] }}</span>
          <KunChip
            v-if="quiz.spoiler_level !== 'none'"
            :color="KUN_QUIZ_SPOILER_COLOR_MAP[quiz.spoiler_level]"
            variant="flat"
            size="sm"
          >
            {{ KUN_QUIZ_SPOILER_MAP[quiz.spoiler_level] }}
          </KunChip>
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:eye" />{{ quiz.view_count }}
          </span>
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:users" />{{ quiz.answer_count }}
          </span>
          <span v-if="quiz.favorite_count" class="flex items-center gap-1">
            <KunIcon name="lucide:heart" />{{ quiz.favorite_count }}
          </span>
          <span v-if="quiz.comment_count" class="flex items-center gap-1">
            <KunIcon name="lucide:message-square" />{{ quiz.comment_count }}
          </span>
        </div>
      </div>

      <div class="flex shrink-0 flex-col items-end gap-1">
        <KunChip
          :color="kunQuizDifficultyColor(quiz.difficulty)"
          variant="flat"
          size="sm"
        >
          {{ kunQuizDifficultyLabel(quiz.difficulty) }} {{ quiz.difficulty }}
        </KunChip>
        <span
          v-if="correctRate(quiz) !== null"
          class="text-default-500 text-xs"
        >
          正确率 {{ correctRate(quiz) }}%
        </span>
      </div>
    </NuxtLink>
  </div>
</template>
