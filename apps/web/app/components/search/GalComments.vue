<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { WallCommentSearchHit } from '#shared/utils/api/schemas'

const props = defineProps<{
  keywords: string
}>()

const PAGE_SIZE = 24

const api = useApiClient()
const results = ref<WallCommentSearchHit[]>([])
const nextCursor = ref('')
const pending = ref(!!props.keywords)
const loadingMore = ref(false)
const failed = ref(false)

const hasMore = computed(() => nextCursor.value !== '')

let latest = 0

const tooShort = computed(() => props.keywords.trim().length < 2)

const fetchPage = async (cursor: string) => {
  const result = await settle(
    api.GET('/search/wall-comments', {
      params: {
        query: {
          q: props.keywords,
          limit: PAGE_SIZE,
          ...(cursor ? { cursor } : {})
        }
      }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return null
  }
  return result.data
}

// The community service answers a keyset cursor and no total, so this lane
// cannot use the shared paginator: there is no last page to bound it with, and
// a page yields fewer rows than it asked for whenever a wall this forum cannot
// link to is dropped.
const load = async () => {
  const current = ++latest
  if (!props.keywords || tooShort.value) {
    results.value = []
    nextCursor.value = ''
    pending.value = false
    return
  }
  pending.value = true
  const data = await fetchPage('')
  if (current !== latest) {
    return
  }
  failed.value = !data
  results.value = data?.items ?? []
  nextCursor.value = data?.next_cursor ?? ''
  pending.value = false
}

const loadMore = async () => {
  if (!hasMore.value || loadingMore.value) {
    return
  }
  const current = latest
  loadingMore.value = true
  const data = await fetchPage(nextCursor.value)
  loadingMore.value = false
  if (current !== latest || !data) {
    return
  }
  const seen = new Set(results.value.map((item) => item.id))
  results.value = [
    ...results.value,
    ...data.items.filter((item) => !seen.has(item.id))
  ]
  nextCursor.value = data.next_cursor ?? ''
}

watch(() => props.keywords, load, { immediate: true })
</script>

<template>
  <div class="space-y-6">
    <p class="text-default-500 text-sm">
      <template v-if="pending && !results.length">正在搜索…</template>
      <template v-else-if="results.length">
        Galgame 与资源页评论区的搜索结果, 按时间从新到旧
      </template>
    </p>

    <SearchSkeleton v-if="pending && !results.length" shape="row" />

    <div v-else-if="results.length" class="space-y-2">
      <KunCard v-for="comment in results" :key="comment.id" padding="sm">
        <SearchGalCommentCard :comment="comment" :keywords="keywords" />
      </KunCard>
    </div>

    <KunNull v-else-if="failed" description="搜索没能完成, 请稍后重试" />

    <KunNull v-else-if="tooShort" description="评论搜索至少需要 2 个字" />

    <KunNull v-else-if="keywords" description="杂鱼杂鱼杂鱼~什么也没有搜索到" />

    <KunButton
      v-if="hasMore"
      variant="light"
      color="primary"
      full-width
      :loading="loadingMore"
      @click="loadMore"
    >
      <KunIcon name="lucide:chevron-down" />
      加载更多评论
    </KunButton>
  </div>
</template>
