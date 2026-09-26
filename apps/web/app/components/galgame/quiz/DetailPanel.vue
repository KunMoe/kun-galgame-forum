<script setup lang="ts">
import type { Quiz, QuizAnswer, QuizSubmission } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'
import { problemMessage } from '#shared/utils/api/message'

const props = defineProps<{ quiz: Quiz }>()

const expanded = ref(false)
const bodyRef = ref<HTMLElement | null>(null)
const maxHeight = ref('6rem')
const expand = () => {
  const el = bodyRef.value
  maxHeight.value = el ? `${el.scrollHeight}px` : 'none'
  expanded.value = true
}
const collapse = () => {
  const el = bodyRef.value
  if (el) maxHeight.value = `${el.scrollHeight}px`
  expanded.value = false
  requestAnimationFrame(() =>
    requestAnimationFrame(() => {
      maxHeight.value = '6rem'
    })
  )
}
const onTransitionEnd = (e: TransitionEvent) => {
  if (e.propertyName === 'max-height' && expanded.value)
    maxHeight.value = 'none'
}

const workName = useWorkName()
const {
  items: records,
  hasMore,
  loadingMore,
  loadMore,
  refresh,
  status,
  problem
} = await useCursorList<QuizAnswer>(
  () => `quiz-answers:${props.quiz.id}`,
  (api, cursor, { signal }) =>
    api.GET('/quizzes/{quiz_id}/answers', {
      params: {
        path: { quiz_id: props.quiz.id },
        query: {
          limit: 20,
          ...(cursor ? { cursor } : {})
        }
      },
      signal
    })
)

watch(
  () => props.quiz.answer_count,
  () => {
    refresh()
  }
)

const wrong = computed(() => props.quiz.answer_count - props.quiz.correct_count)

const letter = (i: number) => String.fromCharCode(65 + i)
const formatSubmitted = (s: QuizSubmission): string => {
  if (props.quiz.quiz_type === 'judge') {
    if (s.is_statement_true === true) return '答: 正确'
    if (s.is_statement_true === false) return '答: 错误'
    return ''
  }
  if (!s.choice_indexes.length) return '未选'
  return `选 ${s.choice_indexes.map(letter).join('、')}`
}
</script>

<template>
  <div class="space-y-1">
    <div class="relative">
      <div
        ref="bodyRef"
        class="overflow-hidden transition-[max-height] duration-300"
        :style="{ maxHeight }"
        @transitionend="onTransitionEnd"
      >
        <div class="space-y-4">
          <div
            v-for="work in quiz.works"
            :key="work.id"
            class="border-default-200 flex gap-3 rounded-lg border p-2"
          >
            <div
              class="bg-default-100 h-20 w-14 shrink-0 overflow-hidden rounded-lg"
            >
              <KunImage
                v-if="work.cover"
                :src="work.cover.url"
                :thumbhash="work.cover.thumbhash ?? undefined"
                :width="work.cover.width ?? 56"
                :height="work.cover.height ?? 80"
                object-fit="cover"
                class-name="h-full w-full"
              />
            </div>
            <div class="min-w-0 flex-1">
              <KunLink :to="`/galgame/${work.id}`" class="font-medium">
                {{ workName(work) }}
              </KunLink>
              <div class="mt-1 flex flex-wrap items-center gap-1">
                <KunChip v-if="work.is_nsfw" size="sm" variant="flat" color="danger">
                  NSFW
                </KunChip>
              </div>
            </div>
          </div>
          <div
            v-if="!quiz.works.length && quiz.is_work_hidden"
            class="text-default-500 border-default-200 flex items-center gap-2 rounded-lg border border-dashed p-3 text-sm"
          >
            <KunIcon name="lucide:lock" />
            关联作品已隐藏, 作答后揭晓
          </div>

          <div>
            <p class="text-default-400 mb-2 text-xs">作答统计</p>
            <div v-if="quiz.answer_count > 0" class="flex items-center gap-4">
              <KunProgress
                variant="circle"
                :value="quiz.correct_count"
                :max="quiz.answer_count"
                color="success"
                size="lg"
                show-label
              />
              <div class="space-y-1 text-sm">
                <p class="text-success flex items-center gap-1">
                  <KunIcon name="lucide:check" />正确 {{ quiz.correct_count }}
                </p>
                <p class="text-danger flex items-center gap-1">
                  <KunIcon name="lucide:x" />错误 {{ wrong }}
                </p>
                <p class="text-default-500">
                  共 {{ quiz.answer_count }} 人作答
                </p>
                <p
                  v-if="quiz.quality_count > 0 && quiz.quality_average != null"
                  class="text-default-500"
                >
                  质量 {{ quiz.quality_average }} ({{ quiz.quality_count }} 人)
                </p>
              </div>
            </div>
            <p v-else class="text-default-500 text-sm">暂无作答</p>
          </div>

          <div>
            <p class="text-default-400 mb-1 text-xs">作答记录</p>
            <p v-if="problem" class="text-default-500 text-sm">
              {{ problemMessage(problem) }}
            </p>
            <p v-else-if="status === 'pending'" class="text-default-500 text-sm">
              加载中…
            </p>
            <p v-else-if="!records.length" class="text-default-500 text-sm">
              还没有人作答
            </p>
            <div v-else class="max-h-64 space-y-1 overflow-y-auto pr-1">
              <div
                v-for="rec in records"
                :key="rec.id"
                class="hover:bg-default-100 rounded-md px-1 py-1"
              >
                <div class="flex items-center gap-2 text-sm">
                  <KunAvatar
                    :user="toKunUser(rec.answerer)"
                    size="xs"
                    :is-navigation="false"
                  />
                  <span class="text-default-700 min-w-0 flex-1 truncate">
                    {{ toKunUser(rec.answerer).name }}
                  </span>
                  <KunTime
                    :time="rec.answered_at"
                    class="text-default-400 text-xs"
                  />
                  <KunChip
                    v-if="rec.is_correct === true"
                    color="success"
                    variant="flat"
                    size="sm"
                  >
                    正确
                  </KunChip>
                  <KunChip
                    v-else-if="rec.is_correct === false"
                    color="danger"
                    variant="flat"
                    size="sm"
                  >
                    错误
                  </KunChip>
                </div>
                <p
                  v-if="rec.submission"
                  class="text-default-500 mt-0.5 pl-7 text-xs break-words whitespace-pre-wrap"
                >
                  {{ formatSubmitted(rec.submission) }}
                </p>
              </div>
              <KunButton
                v-if="hasMore"
                variant="light"
                size="sm"
                class-name="w-full"
                :loading="loadingMore"
                @click="loadMore"
              >
                加载更多
              </KunButton>
            </div>
          </div>
        </div>
      </div>

      <!-- SANCTIONED EXCEPTION to 铁律 #1 (no gradients): the collapsed-state
           fade for the "peek then reveal" affordance. Listed in CLAUDE.md; do
           NOT remove it in a no-gradient sweep. -->
      <div
        v-show="!expanded"
        class="pointer-events-none absolute inset-x-0 bottom-0 h-10 bg-gradient-to-t from-[oklch(var(--content1))] to-transparent"
      />
    </div>

    <div class="flex justify-end">
      <KunButton
        variant="light"
        size="sm"
        @click="expanded ? collapse() : expand()"
      >
        <span class="flex items-center gap-1">
          <KunIcon
            :name="expanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
          />
          {{ expanded ? '收起' : '查看详情' }}
        </span>
      </KunButton>
    </div>
  </div>
</template>
