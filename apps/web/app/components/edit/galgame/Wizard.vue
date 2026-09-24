<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type {
  WorkSubmissionCandidate,
  WorkSubmissionSummary
} from '#shared/utils/api/schemas'

const SEARCH_LIMIT = 12

const q = ref('')
const canReviewClaims = useCan('galgame.claim.review')
const { allowsNsfw } = useContentStance()
const api = useApiClient()
const nameOf = useCatalogName()

const hasSearched = ref(false)
const isSearching = ref(false)
const isLoadingMore = ref(false)
const hits = ref<WorkSubmissionCandidate[]>([])
const nextCursor = ref<string | undefined>()
const pending = ref<WorkSubmissionSummary[]>([])
const searchedQuery = ref('')

const searchPage = (query: string, cursor?: string) =>
  settle(
    api.GET('/work-submission-candidates', {
      params: {
        query: {
          q: query,
          limit: SEARCH_LIMIT,
          include_nsfw: allowsNsfw.value,
          cursor
        }
      }
    })
  )

const handleSearch = async () => {
  const query = q.value.trim()
  if (!query) {
    useMessage('请先输入关键词', 'warn')
    return
  }
  isSearching.value = true
  const [found, mine] = await Promise.all([
    searchPage(query),
    settle(
      api.GET('/me/work-submissions', {
        params: {
          query: { state: ['pending', 'declined'], limit: SEARCH_LIMIT }
        }
      })
    )
  ])
  isSearching.value = false
  hasSearched.value = true
  if (!found.ok) {
    reportProblem(found.problem)
    return
  }
  searchedQuery.value = query
  hits.value = found.data.items
  nextCursor.value = found.data.next_cursor
  pending.value = mine.ok ? mine.data.items : []
}

const loadMore = async () => {
  if (isLoadingMore.value || !nextCursor.value) {
    return
  }
  isLoadingMore.value = true
  const next = await searchPage(searchedQuery.value, nextCursor.value)
  isLoadingMore.value = false
  if (!next.ok) {
    reportProblem(next.problem)
    return
  }
  hits.value.push(...next.data.items)
  nextCursor.value = next.data.next_cursor
}

const imageOf = (hit: WorkSubmissionCandidate) =>
  hit.work_summary.banner?.url ?? hit.work_summary.cover?.url

const handleCreateNew = async () => {
  const store = usePersistEditGalgameStore()
  if (q.value.trim() && !store.name['zh-cn']) {
    store.name['zh-cn'] = q.value.trim()
  }
  await navigateTo('/edit/galgame/create')
}

const noMatches = computed(
  () => hasSearched.value && !hits.value.length && !pending.value.length
)

const route = useRoute()
onMounted(() => {
  const pre = route.query.q
  if (typeof pre === 'string' && pre.trim()) {
    q.value = pre.trim()
    handleSearch()
  }
})
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="发布 Galgame"
      description="先搜索您想发布资源的游戏。资料库里已有的作品直接打开详情页发布资源；确实没有的再新建申请。"
    >
      <template #endContent>
        <div class="flex items-center gap-2">
          <KunLink to="/edit/galgame/mine">
            <KunButton size="sm" variant="flat">我的提交</KunButton>
          </KunLink>
          <KunLink v-if="canReviewClaims" to="/edit/galgame/audited">
            <KunButton size="sm" variant="flat">我的审核</KunButton>
          </KunLink>
        </div>
      </template>
    </KunHeader>

    <KunDivider>
      <span class="mx-2">① 搜索是否已存在</span>
    </KunDivider>

    <div class="space-y-2">
      <div class="flex items-center gap-2">
        <KunInput
          v-model="q"
          placeholder="输入游戏名 (任意语言)"
          @keydown.enter="handleSearch"
        />
        <KunButton
          class-name="whitespace-nowrap"
          :loading="isSearching"
          @click="handleSearch"
        >
          搜索
        </KunButton>
      </div>
      <p class="text-default-500 text-sm">
        搜索覆盖资料库中的全部游戏。打开详情页即可发布资源；catalog 没有的原创 /
        同人作品走下方新建申请。您自己的待审核 / 已拒绝投稿也会列在这里。
      </p>
    </div>

    <div v-if="hasSearched" class="space-y-4">
      <div v-if="pending.length" class="space-y-2">
        <h3 class="text-default-700 text-sm font-bold">您的待审 / 已拒草稿</h3>
        <div
          v-for="item in pending"
          :key="`pending-${item.id}`"
          class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
        >
          <div class="min-w-0 flex-1 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <h4 class="truncate font-medium">
                {{ item.display_name || `#${item.id}` }}
              </h4>
              <KunChip
                size="xs"
                variant="flat"
                :color="galgameClaimStateBadge(item.state).color"
              >
                {{ galgameClaimStateBadge(item.state).label }}
              </KunChip>
            </div>
            <p v-if="item.last_event?.note" class="text-default-500 text-sm">
              {{ item.last_event.note }}
            </p>
          </div>
          <KunLink :to="`/galgame/${item.id}/edit`">
            <KunButton size="sm" variant="flat">继续编辑</KunButton>
          </KunLink>
        </div>
      </div>

      <div v-if="hits.length" class="space-y-2">
        <h3 class="text-default-700 text-sm font-bold">匹配的 Galgame</h3>
        <div
          v-for="hit in hits"
          :key="`item-${hit.work_summary.id}`"
          class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
        >
          <KunImage
            v-if="imageOf(hit)"
            :src="imageOf(hit) ?? ''"
            loading="lazy"
            placeholder="/placeholder.webp"
            class="h-16 w-28 shrink-0 rounded object-cover"
            :style="{ aspectRatio: '16/9' }"
          />
          <div class="min-w-0 flex-1 space-y-1">
            <h4 class="truncate font-medium">
              {{ nameOf(hit.work_summary).name || `#${hit.work_summary.id}` }}
            </h4>
            <p
              v-if="hit.work_summary.release_date"
              class="text-default-500 text-sm"
            >
              {{ hit.work_summary.release_date }}
            </p>
          </div>
          <span
            v-if="hit.state === CLAIM_STATE_PENDING"
            class="text-default-400 shrink-0 text-sm"
          >
            他人投稿审核中
          </span>
          <KunLink v-else :to="`/galgame/${hit.work_summary.id}`">
            <KunButton size="sm" variant="flat">查看 / 发布资源</KunButton>
          </KunLink>
        </div>
        <KunButton
          v-if="nextCursor"
          variant="flat"
          :loading="isLoadingMore"
          @click="loadMore"
        >
          加载更多
        </KunButton>
      </div>

      <KunInfo
        v-if="noMatches"
        color="info"
        title="没有找到匹配的 Galgame"
        description="确认确实没有后, 用下方「新建 Galgame 申请」提交。"
      />
    </div>

    <KunDivider>
      <span class="mx-2">② 都没有？新建申请</span>
    </KunDivider>

    <KunInfo
      color="info"
      title="提交一份新的 Galgame 申请"
      description="仅用于 VNDB 未收录的原创 / 同人 / 独立作品。提交后进入审核队列, 审核通过才会公开。"
    />
    <div class="flex justify-end">
      <KunButton size="lg" @click="handleCreateNew"
        >新建 Galgame 申请</KunButton
      >
    </div>
  </div>
</template>
