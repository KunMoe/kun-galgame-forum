<script setup lang="ts">
import {
  kunUserGalgameNavItem,
  type KUN_USER_PAGE_GALGAME_TYPE
} from '~/constants/user'
import { settle } from '#shared/utils/api/problem'
import type { WallComment, WorkPage } from '#shared/utils/api/schemas'
import { workSummaryToCard } from '~/utils/galgame/workCard'
import ContentDocument from '~/components/content/Document.vue'

const COMMENT_TYPES = ['galgame_comment', 'galgame_comment_like'] as const

const props = defineProps<{
  userId: number
  type: (typeof KUN_USER_PAGE_GALGAME_TYPE)[number]
}>()

const isCommentMode = computed(() =>
  (COMMENT_TYPES as readonly string[]).includes(props.type)
)

const { allowsNsfw: includeNsfw } = useContentStance()
const nameOf = useCatalogName()
const api = useApiClient()

const activeTab = computed(() => props.type)
const pageData = reactive({
  page: usePageQuery(),
  limit: 24
})

const workRelation = computed(() => {
  if (props.type === 'galgame_contributed') {
    return 'contributed' as const
  }
  if (props.type === 'galgame_like') {
    return 'liked' as const
  }
  return 'published' as const
})

const commentRelation = computed(() =>
  props.type === 'galgame_comment_like'
    ? ('liked' as const)
    : ('authored' as const)
)

const { data: galgameData, status: galgameStatus } = await useApi<WorkPage>(
  () =>
    `user-works:${props.userId}:${workRelation.value}:${pageData.page}:${pageData.limit}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (client, { signal }) =>
    client.GET('/users/{user_id}/works', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          relation: workRelation.value,
          page: pageData.page,
          limit: pageData.limit,
          include_nsfw: includeNsfw.value
        }
      },
      signal
    }),
  { immediate: !isCommentMode.value, server: !isCommentMode.value }
)

const galgameCards = computed(() =>
  (galgameData.value?.items ?? []).map((w) => workSummaryToCard(w, nameOf))
)

const { data: commentData } = await useApi(
  () =>
    `user-wall-comments:${props.userId}:${commentRelation.value}:galgame:${pageData.limit}`,
  (client, { signal }) =>
    client.GET('/users/{user_id}/wall-comments', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          relation: commentRelation.value,
          subject_type: 'galgame',
          limit: pageData.limit
        }
      },
      signal
    }),
  { immediate: isCommentMode.value, server: isCommentMode.value }
)

const comments = ref<WallComment[]>([])
const commentCursor = ref<string | undefined>()
const commentHasMore = computed(() => Boolean(commentCursor.value))
const isLoadingMoreComments = ref(false)

watch(
  commentData,
  (page) => {
    if (!page) {
      return
    }
    comments.value = page.items
    commentCursor.value = page.next_cursor
  },
  { immediate: true }
)

const loadMoreComments = async () => {
  if (
    isLoadingMoreComments.value ||
    !commentHasMore.value ||
    !commentCursor.value
  ) {
    return
  }
  isLoadingMoreComments.value = true
  const result = await settle(
    api.GET('/users/{user_id}/wall-comments', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          relation: commentRelation.value,
          subject_type: 'galgame',
          cursor: commentCursor.value,
          limit: pageData.limit
        }
      }
    })
  )
  isLoadingMoreComments.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  const seen = new Set(comments.value.map((c) => c.id))
  comments.value.push(...result.data.items.filter((c) => !seen.has(c.id)))
  commentCursor.value = result.data.next_cursor
}
</script>

<template>
  <div class="space-y-3">
    <KunTab
      :items="kunUserGalgameNavItem(userId)"
      :model-value="activeTab"
      variant="light"
      size="sm"
      scrollable
    />

    <template v-if="!isCommentMode">
      <div
        v-if="galgameData && galgameData.items.length"
        class="flex flex-col space-y-3"
      >
        <GalgameCard :is-transparent="false" :galgames="galgameCards" />

        <KunPagination
          v-if="galgameData.total > pageData.limit"
          v-model:current-page="pageData.page"
          :total-page="Math.ceil(galgameData.total / pageData.limit)"
          :is-loading="galgameStatus === 'pending'"
        />
      </div>

      <KunNull
        v-if="galgameData && !galgameData.items.length"
        description="这只笨蛋萝莉没有相关的 Galgame"
      />
    </template>

    <template v-else>
      <div
        v-if="comments.length || commentHasMore"
        class="flex flex-col space-y-3"
      >
        <KunCard
          v-for="c in comments"
          :key="c.id"
          :href="`/galgame/${c.subject_id}?comment=${c.id}`"
          content-class="space-y-2"
        >
          <p
            v-if="c.state === 'deleted'"
            class="text-default-400 text-sm italic"
          >
            [已删除]
          </p>
          <ContentDocument v-else compact :document="c.content" />

          <div
            class="text-default-500 flex items-center justify-between text-sm"
          >
            <span>{{ `评论于 Galgame #${c.subject_id}` }}</span>
            <KunTime :time="c.created_at" type="date" show-year />
          </div>
        </KunCard>

        <div class="flex justify-center pt-1">
          <KunButton
            v-if="commentHasMore"
            variant="light"
            :loading="isLoadingMoreComments"
            @click="loadMoreComments"
          >
            加载更多
          </KunButton>
          <span v-else class="text-default-400 text-sm">没有更多评论了</span>
        </div>
      </div>

      <KunNull
        v-else
        description="这只笨蛋萝莉在 Galgame 下没有相关的评论"
      />
    </template>
  </div>
</template>
