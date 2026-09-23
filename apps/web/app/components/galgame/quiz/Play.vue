<script setup lang="ts">
import {
  KUN_QUIZ_TYPE_MAP,
  KUN_QUIZ_TYPE_ICON_MAP,
  KUN_QUIZ_TYPE_COLOR_MAP,
  KUN_QUIZ_CATEGORY_MAP,
  KUN_QUIZ_SPOILER_MAP,
  KUN_QUIZ_SPOILER_COLOR_MAP,
  kunQuizDifficultyLabel,
  kunQuizDifficultyColor
} from '~/constants/galgame-quiz'
import { settle } from '#shared/utils/api/problem'
import type {
  Quiz,
  QuizQuality,
  QuizSource,
  QuizSubmission
} from '#shared/utils/api/schemas'
import { contentPlainText } from '~/utils/contentPlainText'
import { toKunUser } from '~/utils/userRef'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'

const props = defineProps<{ quiz: Quiz }>()

const router = useRouter()
const api = useApiClient()
const answerKey = useIdempotencyKey()
const workName = useWorkName()

const returnToLibrary = () => {
  const back = window.history.state?.back
  if (typeof back === 'string' && /^\/galgame-quiz(\?|#|$)/.test(back)) {
    router.back()
  } else {
    navigateTo('/galgame-quiz')
  }
}

const state = ref<Quiz>({ ...props.quiz })
watch(
  () => props.quiz,
  (next) => {
    state.value = { ...next }
  }
)

const answerRef = ref<{
  getSubmitted: () => QuizSubmission
  validate: () => string | null
} | null>(null)
const isSubmitting = ref(false)

const applyQuality = (r: QuizQuality) => {
  state.value = {
    ...state.value,
    quality_average: r.quality_average,
    quality_count: r.quality_count,
    viewer: state.value.viewer
      ? {
          ...state.value.viewer,
          quality_rating: r.viewer?.quality_rating ?? null
        }
      : state.value.viewer
  }
}

const reloadQuiz = async () => {
  const fresh = await settle(
    api.GET('/quizzes/{quiz_id}', {
      params: { path: { quiz_id: state.value.id } }
    })
  )
  if (!fresh.ok) {
    reportProblem(fresh.problem)
    return
  }
  state.value = fresh.data
}

const submitAnswer = async () => {
  if (!requireLogin()) return
  const err = answerRef.value?.validate()
  if (err) {
    useMessage(err, 'warn')
    return
  }
  const body = answerRef.value?.getSubmitted()
  if (!body) return

  isSubmitting.value = true
  const res = await settle(
    api.POST('/quizzes/{quiz_id}/answers', {
      params: {
        path: { quiz_id: state.value.id },
        header: {
          'Idempotency-Key': answerKey.take(
            `/quizzes/${state.value.id}/answers`,
            body
          )
        }
      },
      body
    })
  )
  isSubmitting.value = false
  if (!res.ok) {
    reportProblem(res.problem)
    if (res.problem.code === 'ALREADY_EXISTS') {
      await reloadQuiz()
    }
    return
  }
  answerKey.clear()
  const posted = res.data
  if (posted.answer && posted.solution) {
    state.value = {
      ...state.value,
      solution: posted.solution,
      answer_count: state.value.answer_count + 1,
      correct_count:
        state.value.correct_count + (posted.answer.is_correct ? 1 : 0),
      viewer: state.value.viewer
        ? {
            ...state.value.viewer,
            has_answered: true,
            answer: posted.answer
          }
        : state.value.viewer
    }
  }
  await reloadQuiz()
}

const isDeleting = ref(false)
const canEdit = computed(() => state.value.viewer?.can_edit ?? false)
const canDelete = computed(() => state.value.viewer?.can_delete ?? false)
const canManage = computed(() => canEdit.value || canDelete.value)
const author = computed(() => toKunUser(state.value.author))
const hasDescription = computed(
  () => contentPlainText(state.value.content).trim().length > 0
)
const showAnswerInput = computed(
  () => !state.value.viewer?.has_answered && !state.value.solution
)
const showResult = computed(() => !!state.value.solution)

const showEdit = ref(false)
const editSource = ref<QuizSource | null>(null)
const openEdit = async () => {
  const data = await settle(
    api.GET('/quizzes/{quiz_id}/source', {
      params: { path: { quiz_id: state.value.id } }
    })
  )
  if (!data.ok) {
    reportProblem(data.problem)
    return
  }
  editSource.value = data.data
  showEdit.value = true
}

const remove = async () => {
  const ok = await useComponentMessageStore().alert(
    '确认删除',
    '删除后本题及所有作答记录将被移除, 出题获得的萌萌点会被扣除'
  )
  if (!ok) return
  isDeleting.value = true
  const res = await settle(
    api.DELETE('/quizzes/{quiz_id}', {
      params: { path: { quiz_id: state.value.id } }
    })
  )
  isDeleting.value = false
  if (!res.ok) {
    reportProblem(res.problem)
    return
  }
  useMessage('已删除', 'success')
  returnToLibrary()
}

const favoriteQuiz = async (next: boolean) => {
  const options = { params: { path: { quiz_id: state.value.id } } }
  const result = await settle(
    next
      ? api.PUT('/quizzes/{quiz_id}/favorite', options)
      : api.DELETE('/quizzes/{quiz_id}/favorite', options)
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  state.value = {
    ...state.value,
    favorite_count: result.data.favorite_count,
    viewer: state.value.viewer
      ? {
          ...state.value.viewer,
          has_favorited: result.data.viewer?.has_favorited ?? next
        }
      : state.value.viewer
  }
  return true
}

const correctRate = computed(() =>
  state.value.answer_count > 0
    ? Math.round((state.value.correct_count / state.value.answer_count) * 100)
    : null
)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2">
      <KunButton variant="light" size="sm" @click="returnToLibrary">
        <span class="flex items-center gap-1">
          <KunIcon name="lucide:arrow-left" />返回题库
        </span>
      </KunButton>

      <KunPopover v-if="canManage" position="bottom-end">
        <template #trigger>
          <KunButton :is-icon-only="true" variant="light" size="sm">
            <KunIcon name="lucide:ellipsis" />
          </KunButton>
        </template>
        <div class="flex w-32 flex-col gap-1 p-2">
          <KunButton
            v-if="canEdit"
            variant="light"
            color="default"
            size="sm"
            class-name="w-full justify-start gap-2"
            @click="openEdit"
          >
            <KunIcon name="lucide:pencil" />编辑
          </KunButton>
          <KunButton
            v-if="canDelete"
            variant="light"
            color="danger"
            size="sm"
            class-name="w-full justify-start gap-2"
            :loading="isDeleting"
            @click="remove"
          >
            <KunIcon name="lucide:trash-2" />删除
          </KunButton>
        </div>
      </KunPopover>
    </div>

    <KunCard :is-transparent="false">
      <div class="space-y-4">
        <div class="flex flex-wrap items-center gap-2">
          <KunChip
            :color="KUN_QUIZ_TYPE_COLOR_MAP[state.quiz_type]"
            variant="flat"
          >
            <span class="flex items-center gap-1">
              <KunIcon :name="KUN_QUIZ_TYPE_ICON_MAP[state.quiz_type]" />
              {{ KUN_QUIZ_TYPE_MAP[state.quiz_type] }}
            </span>
          </KunChip>
          <KunChip
            :color="kunQuizDifficultyColor(state.difficulty)"
            variant="flat"
          >
            {{ kunQuizDifficultyLabel(state.difficulty) }} ·
            {{ state.difficulty }}
          </KunChip>
          <KunChip variant="light">
            {{ KUN_QUIZ_CATEGORY_MAP[state.quiz_category] }}
          </KunChip>
          <KunChip
            v-if="state.spoiler_level !== 'none'"
            :color="KUN_QUIZ_SPOILER_COLOR_MAP[state.spoiler_level]"
            variant="flat"
          >
            {{ KUN_QUIZ_SPOILER_MAP[state.spoiler_level] }}
          </KunChip>
          <KunLink
            v-for="work in state.works"
            :key="work.id"
            :to="`/galgame/${work.id}`"
            class="text-sm"
          >
            {{ workName(work) }}
          </KunLink>
          <KunChip
            v-if="!state.works.length && state.is_work_hidden"
            variant="flat"
            size="sm"
          >
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:lock" />关联作品作答后揭晓
            </span>
          </KunChip>
        </div>

        <div role="heading" aria-level="1">
          <ContentDocument
            :document="state.prompt"
            compact
            class-name="text-xl font-bold break-words"
          />
        </div>

        <ContentDocument v-if="hasDescription" :document="state.content" />

        <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
          <div
            class="text-default-500 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm"
          >
            <span class="text-default-700 flex items-center gap-1">
              <KunAvatar
                :disable-floating="true"
                :user="author"
                size="xs"
                :is-navigation="false"
              />
              {{ author.name }}
            </span>
            <KunTime :time="state.created_at" />
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:users" />{{ state.answer_count }} 人作答
            </span>
            <span v-if="correctRate !== null" class="flex items-center gap-1">
              <KunIcon name="lucide:target" />正确率 {{ correctRate }}%
            </span>
          </div>

          <div class="text-default-500 ml-auto flex items-center gap-1">
            <span class="inline-flex items-center gap-1.5 px-2 py-1 text-sm">
              <KunIcon name="lucide:eye" class="text-[1.15rem]" />{{
                state.view_count
              }}
            </span>
            <FavoriteToggle
              :favorited="state.viewer?.has_favorited ?? false"
              :count="state.favorite_count"
              :action="favoriteQuiz"
              :messages="['已收藏', '已取消收藏']"
            />
          </div>
        </div>

        <KunDivider />

        <div v-if="showAnswerInput" class="space-y-4">
          <GalgameQuizPlayAnswerInput
            ref="answerRef"
            :type="state.quiz_type"
            :choices="state.choices"
          />
          <div class="flex justify-end">
            <KunButton :loading="isSubmitting" @click="submitAnswer">
              提交答案
            </KunButton>
          </div>
        </div>

        <GalgameQuizPlayResult
          v-else-if="showResult && state.solution"
          :quiz="state"
          :answer="state.viewer?.answer ?? null"
          :solution="state.solution"
          @rated="applyQuality"
        />

        <KunDivider />

        <GalgameQuizDetailPanel :quiz="state" />
      </div>
    </KunCard>

    <GalgameQuizPublish
      v-model="showEdit"
      :source="editSource"
      :works="state.works"
      @on-updated="reloadQuiz"
    />
  </div>
</template>
