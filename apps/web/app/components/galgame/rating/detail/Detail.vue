<script setup lang="ts">
import {
  KUN_GALGAME_PLAY_STATE_MAP,
  type KunGalgamePlayState
} from '~/constants/galgame-playtime'
import {
  KUN_GALGAME_RATING_RECOMMEND_MAP,
  KUN_GALGAME_RATING_RECOMMEND_COLOR_MAP,
  KUN_GALGAME_RATING_SPOILER_MAP,
  KUN_GALGAME_RATING_SPOILER_COLOR_MAP,
  KUN_GALGAME_DIMENSIONS,
  KUN_GALGAME_DIM_LABELS,
  KUN_GALGAME_DIM_DESCRIPTIONS
} from '~/constants/galgame-rating'
import { calcGalgameRating } from '~~/algorithms/GalgameRatingAlg'

const props = defineProps<{
  ratingId: number
  data: GalgameRatingDetails
  refresh: () => void
}>()

const { id: userId } = usePersistUserStore()
const canDeleteAnyRating = useCan('rating.delete_any')

const spoilerRevealed = ref(false)
const isSummaryMasked = computed(
  () => props.data.spoiler_level === 'serious' && !spoilerRevealed.value
)

const canEdit = computed(() => props.data.user.id === userId)
const canDelete = computed(
  () => props.data.user.id === userId || canDeleteAnyRating.value
)
const rating = computed(() =>
  calcGalgameRating(
    { ...props.data },
    props.data.overall,
    props.data.play_status as KunGalgamePlayState,
    props.data.recommend as 'no'
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

  const res = await kunFetch(`/galgame-rating/${props.data.id}`, {
    method: 'DELETE',
    query: { galgame_rating_id: props.data.id }
  })
  if (res) {
    useMessage('删除成功', 'success')
    await navigateTo(`/galgame/${props.data.galgame.id}`)
  }
}
</script>

<template>
  <div class="space-y-3">
    <KunCard
      :is-transparent="false"
      :is-hoverable="false"
      content-class="space-y-3"
    >
      <GalgameRatingDetailGalgame :galgame="data.galgame" />
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
              :user="data.user"
            />

            <div class="flex flex-col gap-1">
              <div class="flex items-center gap-3 text-lg font-bold sm:text-xl">
                <span>
                  {{ data.user.name }}
                </span>
              </div>

              <div class="text-default-500 flex items-center gap-2 text-sm">
                <KunIcon name="lucide:calendar" />
                <KunTime :time="data.created" type="datetime" show-year />
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
                KUN_GALGAME_PLAY_STATE_MAP[data.play_status as KunGalgamePlayState]
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
          :model-value="{
            art: data.art,
            story: data.story,
            music: data.music,
            character: data.character,
            route: data.route,
            system: data.system,
            voice: data.voice,
            replay_value: data.replay_value
          }"
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
            <KunChip color="secondary">{{ data[dim] }}</KunChip>
          </div>
          <p class="text-default-500 text-sm">
            {{ KUN_GALGAME_DIM_DESCRIPTIONS[dim][data[dim]] }}
          </p>
        </div>
      </div>

      <div
        v-if="data.liked_users?.length"
        class="flex flex-wrap items-center gap-2"
      >
        <KunAvatarGroup :users="data.liked_users" :ellipsis="false" />
        <span class="text-default-500 text-sm">点赞了该评分</span>
      </div>

      <div class="flex items-center justify-between gap-2">
        <div class="space-x-1">
          <KunButton size="lg" color="default" variant="light">
            <KunIcon name="lucide:eye" />
            <span class="text-sm">浏览</span>
            <span class="text-sm">{{ data.view }}</span>
          </KunButton>

          <GalgameRatingDetailLike
            :rating-id="data.id"
            :target-user-id="data.user.id"
            :like-count="data.like_count"
            :is-liked="data.is_liked"
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

    <GalgameRatingCommentCommunityContainer
      v-if="data"
      :rating-id="data.id"
      :rating-author="data.user"
    />

    <GalgameRatingPublish
      v-if="data && canEdit"
      v-model="isEditOpen"
      :galgame-id="data.galgame.id"
      :initial-data="{
        galgameRatingId: data.id,
        recommend: data.recommend as 'no',
        overall: data.overall,
        play_status: data.play_status as KunGalgamePlayState,
        spoiler_level: data.spoiler_level as 'none',
        short_summary: data.short_summary,
        art: data.art,
        story: data.story,
        music: data.music,
        character: data.character,
        route: data.route,
        system: data.system,
        voice: data.voice,
        replay_value: data.replay_value,
        galgameType: data.galgame_type
      }"
      @on-updated="refresh"
    />
  </div>
</template>
