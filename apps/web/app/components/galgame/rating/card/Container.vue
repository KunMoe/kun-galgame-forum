<script setup lang="ts">
import type { RatingsQuery } from '#shared/utils/api/schemas'
import { KUN_GALGAME_PLAY_STATE_CONST } from '~/constants/galgame-playtime'
import {
  KUN_GALGAME_RATING_GAME_TYPE_CONST,
  KUN_GALGAME_RATING_SPOILER_CONST
} from '~/constants/galgame-rating'
import { ratingToCard } from '~/utils/galgame/ratingCard'
import {
  spoilerOptions,
  playStatusOptions,
  typeOptions,
  sortFieldOptions
} from './_sort'

// The filters go in the URL with the page: a page restored on its own would
// point into a list the reader never saw, since every filter would have reset
// to its default on the way back.
const filters = useQueryState(
  {
    page: queryPage(),
    sort_field: queryString('time'),
    sort_order: queryEnum(['desc', 'asc'] as const, 'desc'),
    spoiler_level: queryString('all'),
    play_status: queryString('all'),
    galgame_type: queryString('all')
  },
  { pageKey: 'page' }
)
const params = reactive({
  page: filters.page,
  limit: 24,
  sort_field: filters.sort_field,
  sort_order: filters.sort_order,
  spoiler_level: filters.spoiler_level,
  play_status: filters.play_status,
  galgame_type: filters.galgame_type
})

const known = <T extends string>(value: string, vocab: readonly T[]) =>
  vocab.includes(value as T) ? (value as T) : undefined

const { allowsNsfw } = useContentStance()
const query = computed<RatingsQuery>(() => {
  const column = params.sort_field === 'time' ? 'created' : params.sort_field
  const order = params.sort_order === 'asc' ? 'asc' : 'desc'
  const sort = known(`${column}_${order}`, [
    'created_desc',
    'created_asc',
    'view_desc',
    'view_asc',
    'overall_desc',
    'overall_asc'
  ] as const)
  return {
    page: params.page,
    limit: params.limit,
    sort,
    spoiler_level: known(params.spoiler_level, KUN_GALGAME_RATING_SPOILER_CONST),
    play_status: known(params.play_status, KUN_GALGAME_PLAY_STATE_CONST),
    game_type: known(params.galgame_type, KUN_GALGAME_RATING_GAME_TYPE_CONST),
    include_nsfw: allowsNsfw.value
  }
})

const nameOf = useCatalogName()
const { data, status } = await useApi(
  () => `ratings:${JSON.stringify(query.value)}`,
  (api, { signal }) =>
    api.GET('/ratings', { params: { query: query.value }, signal })
)
const ratings = computed(() =>
  (data.value?.items ?? []).map((r) => ratingToCard(r, nameOf))
)
</script>

<template>
  <div class="space-y-3">
    <div class="space-y-2">
      <KunHeader name="Galgame 评分列表">
        <template #description>
          <p class="text-default-500">
            浏览所有用户对所有 Galgame 的评分与短评，支持按 Galgame 剧透等级,
            Galgame 游玩状态, Galgame 游戏类型筛选, 并按时间, Galgame 热度或
            Galgame 评分进行排序。
          </p>
        </template>
      </KunHeader>

      <div
        class="flex w-full shrink-0 flex-wrap items-center justify-between gap-3 rounded-lg border border-transparent sm:flex-nowrap"
      >
        <div class="grid w-full grid-cols-2 gap-3 lg:grid-cols-4">
          <KunSelect v-model="params.spoiler_level" :options="spoilerOptions" />
          <KunSelect
            v-model="params.play_status"
            :options="playStatusOptions"
          />
          <KunSelect v-model="params.galgame_type" :options="typeOptions" />
          <KunSelect v-model="params.sort_field" :options="sortFieldOptions" />
        </div>

        <div class="flex items-center gap-2">
          <KunButton
            :is-icon-only="true"
            :variant="params.sort_order === 'desc' ? 'flat' : 'light'"
            size="lg"
            @click="params.sort_order = 'desc'"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-down" />
          </KunButton>

          <KunButton
            :is-icon-only="true"
            :variant="params.sort_order === 'asc' ? 'flat' : 'light'"
            size="lg"
            @click="params.sort_order = 'asc'"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-up" />
          </KunButton>
        </div>
      </div>
    </div>

    <GalgameRatingCard v-if="data" :ratings="ratings" />

    <KunPagination
      v-if="(data?.total || 0) > params.limit"
      v-model:current-page="params.page"
      :total-page="Math.ceil((data?.total || 0) / params.limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
