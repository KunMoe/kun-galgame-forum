<script setup lang="ts">
import { KUN_GALGAME_PLAY_STATE_MAP } from '~/constants/galgame-playtime'
import {
  KUN_GALGAME_RATING_RECOMMEND_MAP,
  KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP,
  KUN_GALGAME_RATING_SPOILER_MAP,
  KUN_GALGAME_RATING_SPOILER_COLOR_MAP,
  KUN_GALGAME_DIMENSIONS,
  KUN_GALGAME_DIM_LABELS,
  KUN_GALGAME_DIM_DESCRIPTIONS
} from '~/constants/galgame-rating'
import type { Rating } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { calcGalgameRating } from '~~/algorithms/GalgameRatingAlg'
import { aspectDims } from '~/utils/galgame/ratingCard'

const props = defineProps<{
  data: Rating
  refresh: () => void
}>()

const { id: userId } = usePersistUserStore()
const api = useApiClient()
const author = computed(() => toKunUser(props.data.author))
const dims = computed(() => aspectDims(props.data.aspect_scores))
const likers = computed(() => props.data.likers.map(toKunUser))

const spoilerRevealed = ref(false)
const isSummaryMasked = computed(
  () => props.data.spoiler_level === 'serious' && !spoilerRevealed.value
)

const canEdit = computed(() => props.data.viewer?.can_edit ?? false)
const canDelete = computed(() => props.data.viewer?.can_delete ?? false)
const rating = computed(() =>
  calcGalgameRating(
    dims.value,
    props.data.overall,
    props.data.play_status,
    props.data.recommend
  )
)

const isEditOpen = ref(false)

const handleDeleteRating = async () => {
  if (!userId) {
    useAuthModal().open()
    return
  }

  const ok = await useComponentMessageStore().alert(
    '确认删除评分？',
    '删除操作不可恢复'
  )
  if (!ok) {
    return
  }

  const result = await settle(
    api.DELETE('/ratings/{rating_id}', {
      params: { path: { rating_id: props.data.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('删除成功', 'success')
  await navigateTo(`/galgame/${props.data.work_summary.id}`)
}
</script>

<template>
  <div class="space-y-3">
    <KunCard
      :is-transparent="false"
      :is-hoverable="false"
      content-class="space-y-3"
    >
      <GalgameRatingDetailGalgame :work="data.work_summary" />
    </KunCard>

    <KunCard
      :is-transparent="false"
      :is-hoverable="false"
      content-class="space-y-3"
    >
      <div class="flex flex-wrap gap-6">
        <div class="flex flex-1 flex-col gap-3">
          <div class="flex items-center gap-3">
            <KunAvatar
              class-name="size-15"
              image-class-name="size-15"
              :user="author"
            />

            <div class="flex flex-col gap-1">
              <div class="flex items-center gap-3 text-lg font-bold sm:text-xl">
                <span>
                  {{ author.name }}
                </span>
              </div>

              <div class="text-default-500 flex items-center gap-2 text-sm">
                <KunIcon name="lucide:calendar" />
                <KunTime :time="data.created_at" type="datetime" show-year />
              </div>
            </div>

            <div class="ml-auto flex flex-col items-center">
              <span
                class="text-warning flex items-center gap-1 text-3xl font-bold"
              >
                <KunIcon class-name="text-2xl" name="lucide:lollipop" />
                {{ rating }}
              </span>
              <KunLink
                to="/doc/galgame-rating-guide"
                size="sm"
                class="text-default"
              >
                系统算法评分
              </KunLink>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <div class="text-default-500 text-sm">通关状态</div>
            <KunChip color="primary">
              {{
                KUN_GALGAME_PLAY_STATE_MAP[data.play_status]
              }}
            </KunChip>

            <span class="bg-default-300 h-3 w-px" />

            <div class="text-default-500 text-sm">推荐程度</div>
            <KunChip
              :color="KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP[data.recommend]"
            >
              {{ KUN_GALGAME_RATING_RECOMMEND_MAP[data.recommend] }}
            </KunChip>

            <span class="bg-default-300 h-3 w-px" />

            <div class="text-default-500 text-sm">剧透程度</div>
            <KunChip
              :color="KUN_GALGAME_RATING_SPOILER_COLOR_MAP[data.spoiler_level]"
            >
              {{ KUN_GALGAME_RATING_SPOILER_MAP[data.spoiler_level] }}
            </KunChip>

            <span class="bg-default-300 h-3 w-px" />

            <KunChip color="warning" variant="solid">
              用户总评分
              {{ data.overall }}
            </KunChip>
          </div>

          <div class="relative">
            <KunText
              :content="data.short_summary"
              :class-name="
                cn('leading-7', isSummaryMasked && 'blur-sm select-none')
              "
            />
            <button
              v-if="isSummaryMasked"
              type="button"
              class="bg-background/40 hover:bg-background/20 absolute inset-0 flex flex-col items-center justify-center gap-1 rounded-md backdrop-blur-[2px] transition-colors"
              @click="spoilerRevealed = true"
            >
              <KunIcon name="lucide:eye" class="text-danger size-6" />
              <span class="text-default-700 text-sm font-medium">
                该评分含严重剧透，点击查看
              </span>
            </button>
          </div>
        </div>

        <GalgameRatingRadar
          :model-value="dims"
          :readonly="true"
          :size="300"
        />
      </div>

      <KunHeader
        name="维度说明"
        description="各维度对应分值的文字解释"
        scale="h2"
        class="mt-6"
      />
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <div
          v-for="dim in KUN_GALGAME_DIMENSIONS"
          :key="dim"
          class="rounded-lg border p-3"
        >
          <div class="mb-1 flex items-center justify-between">
            <span class="font-medium">
              {{ KUN_GALGAME_DIM_LABELS[dim] }}
            </span>
            <KunChip color="secondary">{{ dims[dim] }}</KunChip>
          </div>
          <p class="text-default-500 text-sm">
            {{ KUN_GALGAME_DIM_DESCRIPTIONS[dim][dims[dim]] }}
          </p>
        </div>
      </div>

      <div v-if="likers.length" class="flex flex-wrap items-center gap-2">
        <KunAvatarGroup :users="likers" :ellipsis="false" />
        <span class="text-default-500 text-sm">点赞了该评分</span>
      </div>

      <div class="flex items-center justify-between gap-2">
        <div class="space-x-1">
          <KunButton size="lg" color="default" variant="light">
            <KunIcon name="lucide:eye" />
            <span class="text-sm">浏览</span>
            <span class="text-sm">{{ data.view_count }}</span>
          </KunButton>

          <GalgameRatingDetailLike
            :rating-id="Number(data.id)"
            :target-user-id="author.id"
            :like-count="data.like_count"
            :is-liked="data.viewer?.has_liked ?? false"
          />
        </div>

        <div class="space-x-1">
          <KunButton
            v-if="canDelete"
            size="sm"
            color="danger"
            variant="light"
            @click="handleDeleteRating"
          >
            <KunIcon name="lucide:trash-2" /> 删除
          </KunButton>

          <KunButton
            v-if="canEdit"
            size="sm"
            variant="flat"
            @click="isEditOpen = true"
          >
            <KunIcon name="lucide:pencil" /> 编辑
          </KunButton>
        </div>
      </div>
    </KunCard>

    <GalgameRatingCommentCommunityContainer :rating-id="Number(data.id)" />

    <GalgameRatingPublish
      v-if="canEdit"
      v-model="isEditOpen"
      :work-id="Number(data.work_summary.id)"
      :initial-data="{
        galgameRatingId: Number(data.id),
        recommend: data.recommend,
        overall: data.overall,
        play_status: data.play_status,
        spoiler_level: data.spoiler_level,
        short_summary: data.short_summary,
        ...dims,
        galgameType: [...data.game_types]
      }"
      @on-updated="refresh"
    />
  </div>
</template>
