<script setup lang="ts">
import type {
  CollectionEntry,
  PageListCollectionEntry
} from '#shared/utils/api/schemas'
import { useCursorList } from '~/composables/useCursorList'

const props = defineProps<{
  collectionId: string
}>()

const sort = defineModel<CollectionEntrySortValue>('sort', { required: true })

const LIMIT = 48
const MAX_DEPTH = 10000

type QuickItem = CollectionEntry & { id: string }
type QuickPage = Omit<PageListCollectionEntry, 'items'> & {
  items: QuickItem[]
  next_cursor?: string
}

const { allowsNsfw, stanceKey } = useContentStance()
const listKey = computed(() => `${props.collectionId}:${sort.value}`)

const { items, total, hasMore, loadingMore, loadMore, problem, pending } =
  useCursorList<QuickItem, QuickPage>(
    () => `collection-quick:${listKey.value}:${LIMIT}:${stanceKey.value}`,
    async (client, cursor, { signal }) => {
      const page = cursor ? Number(cursor) : 1
      const res = await client.GET('/collections/{collection_id}/entries', {
        params: {
          path: { collection_id: props.collectionId },
          query: {
            page,
            limit: LIMIT,
            include_nsfw: allowsNsfw.value,
            sort: sort.value
          }
        },
        signal
      })
      if (!res.data) {
        return res
      }
      const { items: entries, ...rest } = res.data
      const more =
        entries.length === LIMIT &&
        (page + 1) * LIMIT <= MAX_DEPTH &&
        (rest.total_relation === 'gte' || page * LIMIT < rest.total)
      return {
        ...res,
        data: {
          ...rest,
          items: entries.map((entry) => ({
            ...entry,
            id: entry.work_summary.id
          })),
          ...(more ? { next_cursor: String(page + 1) } : {})
        }
      }
    }
  )

const sentinel = useTemplateRef('sentinel')
const { loadNext } = useAutoLoadMore(sentinel, {
  hasMore,
  loading: loadingMore,
  loadMore,
  resetOn: listKey
})

const galgames = useWorkCards(() =>
  items.value.map((item) => item.work_summary)
)
const noteOf = computed(
  () => new Map(items.value.map((item) => [Number(item.id), item.note]))
)
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center gap-3">
      <span v-if="total !== undefined" class="text-default-500 text-sm">
        共 {{ total }} 部作品
      </span>
      <KunSelect
        v-model="sort"
        :options="[...COLLECTION_ENTRY_SORTS]"
        size="sm"
        class-name="ml-auto w-44"
        aria-label="排序"
      />
    </div>

    <KunInfo
      v-if="problem"
      color="danger"
      title="读取失败"
      description="暂时读不到这个收藏夹的作品, 请稍后再试。"
    />

    <GalgameCard
      v-else-if="galgames.length"
      :is-transparent="false"
      :galgames="galgames"
    >
      <template #meta="{ galgame }">
        <p
          v-if="noteOf.get(galgame.id)"
          class="text-default-500 mt-1 line-clamp-2 text-xs break-words"
          :title="noteOf.get(galgame.id) ?? undefined"
        >
          <KunIcon
            name="lucide:message-square-text"
            class="mr-0.5 inline size-3 align-[-0.125em]"
          />
          {{ noteOf.get(galgame.id) }}
        </p>
      </template>
    </GalgameCard>

    <GalgameCardGridSkeleton v-else-if="pending" />

    <KunNull v-else description="这个收藏夹还没有收藏任何 Galgame" />

    <div v-if="items.length" ref="sentinel" class="flex justify-center">
      <KunButton
        v-if="hasMore"
        variant="flat"
        :loading="loadingMore"
        @click="loadNext"
      >
        加载更多
      </KunButton>
    </div>
  </div>
</template>
