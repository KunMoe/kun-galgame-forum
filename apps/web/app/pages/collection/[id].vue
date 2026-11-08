<script setup lang="ts">
import type { BreadcrumbList, CollectionPage, WithContext } from 'schema-dts'
import {
  settle,
  isMissingProblem,
  type ApiResult
} from '#shared/utils/api/problem'
import { deletedUserName } from '#shared/utils/deletedUser'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'
import type {
  Collection,
  CollectionVisibility,
  BatchListCollectionProgress,
  ListCollectionSummary,
  BatchListMyWork,
  ListWallComment,
  PageListCollectionEntry
} from '#shared/utils/api/schemas'

const route = useRoute()
const collectionId = computed(() => String((route.params as { id: string }).id))
const { allowsNsfw, stanceKey } = useContentStance()
const api = useApiClient()
const limit = 24

const SORT_OPTIONS = [
  { value: 'position_asc', label: '自定义顺序' },
  { value: 'favorited_desc', label: '最近收藏' },
  { value: 'favorited_asc', label: '最早收藏' },
  { value: 'released_desc', label: '发售日期 (新到旧)' },
  { value: 'released_asc', label: '发售日期 (旧到新)' }
] as const
type EntrySort = (typeof SORT_OPTIONS)[number]['value']

const { sort, page } = useQueryState(
  {
    sort: queryString(),
    page: queryPage()
  },
  { pageKey: 'page' }
)

const collectionRead = useKunApi<Collection>(
  () => `collection:${collectionId.value}:${stanceKey.value}`,
  (client, { signal }) =>
    client.GET('/collections/{collection_id}', {
      params: {
        path: { collection_id: collectionId.value },
        query: { include_nsfw: allowsNsfw.value }
      },
      signal
    }),
  { critical: true }
)

const { data: detail, problem, pending, refresh } = await collectionRead
useKunNotFound(() => isMissingProblem(problem.value), '未找到这个合集')

const effectiveSort = computed<EntrySort>(() => {
  const picked = SORT_OPTIONS.find((o) => o.value === sort.value)
  if (picked) {
    return picked.value
  }
  return detail.value?.is_default ? 'favorited_desc' : 'position_asc'
})

const {
  data: entries,
  status: entriesStatus,
  pending: entriesPending,
  refresh: refreshEntries
} = useKunApi<PageListCollectionEntry>(
  () =>
    `collection-entries:${collectionId.value}:${effectiveSort.value}:${page.value}:${limit}:${stanceKey.value}`,
  (client, { signal }) =>
    client.GET('/collections/{collection_id}/entries', {
      params: {
        path: { collection_id: collectionId.value },
        query: {
          page: page.value,
          limit,
          include_nsfw: allowsNsfw.value,
          sort: effectiveSort.value
        }
      },
      signal
    })
)

const { data: related } = useKunApi<ListCollectionSummary>(
  () => `collection-related:${collectionId.value}:${stanceKey.value}`,
  (client, { signal }) =>
    client.GET('/collections/{collection_id}/related', {
      params: {
        path: { collection_id: collectionId.value },
        query: { include_nsfw: allowsNsfw.value }
      },
      signal
    })
)

const ownerName = computed(() => detail.value?.owner.name ?? deletedUserName)
const displayName = computed(() =>
  detail.value ? collectionDisplayName(detail.value, ownerName.value) : '收藏夹'
)
const displayDescription = computed(() =>
  detail.value
    ? collectionDisplayDescription(detail.value, ownerName.value)
    : ''
)

const copySource = computed(() => detail.value?.copied_from ?? null)
const bodyDescription = computed(() => {
  const text = displayDescription.value
  const prefix = copySource.value ? `源自「${copySource.value.title}」` : ''
  return prefix && text.startsWith(prefix)
    ? text.slice(prefix.length).trimStart()
    : text
})

const hiddenCount = computed(() =>
  detail.value && entries.value
    ? Math.max(detail.value.item_count - entries.value.total, 0)
    : 0
)
const hiddenNote = computed(() =>
  allowsNsfw.value
    ? `另有 ${hiddenCount.value} 部作品已下架，未显示`
    : `另有 ${hiddenCount.value} 部作品未显示：受你的内容显示设置影响，或已下架`
)

const isOwner = computed(() => !!detail.value?.viewer?.is_owner)
const canEdit = computed(() => !!detail.value?.viewer?.can_edit)
const canDelete = computed(() => !!detail.value?.viewer?.can_delete)
const isPublic = computed(() => detail.value?.visibility === 'public')
const isShared = computed(
  () => !!detail.value && detail.value.visibility !== 'private'
)
const isSignedIn = computed(() => !!usePersistUserStore().id)

const canReorder = computed(
  () => isOwner.value && effectiveSort.value === 'position_asc'
)

const following = ref(false)
const followerCount = ref(0)
watch(
  detail,
  (value) => {
    following.value = !!value?.viewer?.is_following
    followerCount.value = value?.follower_count ?? 0
  },
  { immediate: true, flush: 'sync' }
)
const onFollowChanged = (value: boolean, count: number) => {
  following.value = value
  followerCount.value = count
}

const reportOpen = ref(false)
const menu = useTemplateRef<{ close: () => void }>('menu')
const hasMenu = computed(
  () => canEdit.value || canDelete.value || isShared.value
)
const runMenu = (action: () => void) => {
  menu.value?.close()
  action()
}

const openReport = () => {
  if (requireLogin()) {
    reportOpen.value = true
  }
}

const visibilityMeta = collectionVisibilityMeta

const findOwnCopy = async () => {
  for (let page = 1; page <= 2; page++) {
    const result = await settle(
      api.GET('/me/collections', { params: { query: { page, limit: 100 } } })
    )
    if (!result.ok) {
      return null
    }
    const hit = result.data.items.find(
      (c) => c.copied_from_collection_id === collectionId.value
    )
    if (hit || result.data.items.length < 100) {
      return hit ?? null
    }
  }
  return null
}

const copyKey = useIdempotencyKey()
const copying = ref(false)
const copyCollection = async () => {
  if (!requireLogin() || copying.value || !detail.value) {
    return
  }
  copying.value = true
  const existing = await findOwnCopy()
  copying.value = false
  const ok = await useComponentMessageStore().alert(
    existing ? '再复制一份？' : '复制到我的收藏夹？',
    (existing
      ? `你已经复制过这个收藏夹，副本是「${existing.title || '未命名收藏夹'}」，再复制会另外新建一个。`
      : '') +
      `会在你的账号下新建一个仅自己可见的收藏夹，放入这里现在的 ${detail.value.item_count} 部作品，保留原来的顺序、备注、标签和封面设置，介绍开头会注明来源。复制之后两边互不影响：原收藏夹之后的更新不会同步过来，你的修改也不会改动原收藏夹。`
  )
  if (!ok) {
    return
  }
  copying.value = true
  const body = { collection_id: collectionId.value }
  const result = await settle(
    api.POST('/collection-copies', {
      params: {
        header: {
          'Idempotency-Key': copyKey.take('/collection-copies', body)
        }
      },
      body
    })
  )
  copying.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  copyKey.clear()
  useMessage('已复制到你的收藏夹，可以改个名字或删掉来源说明', 'success')
  await navigateTo(`/collection/${result.data.id}?edit=1`)
}

const setCover = async (workId: string | null) => {
  const result = await settle(
    api.PATCH('/collections/{collection_id}', {
      params: { path: { collection_id: collectionId.value } },
      body: { cover_work_id: workId ?? '' }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(workId ? '已设为封面' : '已改回自动封面', 'success')
  refresh()
}

const { refresh: refreshFollowedUpdates } = useFollowedCollectionUpdates()
const markSeen = async () => {
  if (!following.value || !detail.value) {
    return
  }
  const result = await settle(
    api.PUT('/collections/{collection_id}/read-marker', {
      params: { path: { collection_id: detail.value.id } }
    })
  )
  if (result.ok) {
    void refreshFollowedUpdates()
  }
}

const { data: progressList, refresh: refreshProgress } =
  useKunApi<BatchListCollectionProgress>(
    () =>
      isSignedIn.value
        ? `collection-progress:${collectionId.value}`
        : 'collection-progress:none',
    (client, { signal }) =>
      client.GET('/me/collection-progress', {
        params: { query: { collection_ids: [collectionId.value] } },
        signal
      }),
    { immediate: isSignedIn.value }
  )
const progress = computed(() => progressList.value?.items[0] ?? null)

const pageWorkIds = computed(
  () => entries.value?.items.map((e) => e.work_summary.id) ?? []
)
const { data: myWorks } = useKunApi<BatchListMyWork>(
  () =>
    isSignedIn.value && pageWorkIds.value.length
      ? `collection-my-works:${collectionId.value}:${pageWorkIds.value.join(',')}`
      : 'collection-my-works:none',
  (client, { signal }) =>
    client.GET('/me/works', {
      params: { query: { work_ids: pageWorkIds.value } },
      signal
    }),
  { server: false, immediate: isSignedIn.value && pageWorkIds.value.length > 0 }
)
const markedStates = ref<Record<string, string | null>>({})
const marks = computed(() => {
  const out: Record<string, { collected: boolean; playState: string | null }> =
    {}
  for (const w of myWorks.value?.items ?? []) {
    out[w.work_id] = {
      collected: (w.library?.collection_ids ?? []).some(
        (id) => id !== collectionId.value
      ),
      playState:
        w.work_id in markedStates.value
          ? (markedStates.value[w.work_id] ?? null)
          : (w.library?.playtime?.play_state ?? null)
    }
  }
  return out
})
const onMarked = (workId: string, playState: string | null) => {
  markedStates.value = { ...markedStates.value, [workId]: playState }
  void refreshProgress()
}

onMounted(() => {
  void markSeen()
  if (route.query.edit === '1' && canEdit.value) {
    editOpen.value = true
    navigateTo(
      { query: { ...route.query, edit: undefined } },
      { replace: true }
    )
  }
})

const editOpen = ref(false)
const editInitial = computed(() => ({
  title: detail.value?.title ?? '',
  description: detail.value?.description ?? '',
  visibility: detail.value?.visibility ?? ('public' as CollectionVisibility),
  tags: detail.value?.tag_names ?? []
}))

const onEdited = () => {
  editOpen.value = false
  refresh()
}

const remove = async () => {
  const ok = await useComponentMessageStore().alert(
    '确定删除该收藏夹吗?',
    '删除后收藏夹内的收藏关系将一并移除，不可撤销。'
  )
  if (!ok) {
    return
  }
  const result = await settle(
    api.DELETE('/collections/{collection_id}', {
      params: { path: { collection_id: collectionId.value } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(10568, 'success')
  navigateTo(`/user/${detail.value?.owner.id}/collection/galgame`)
}

const nameOfWork = useWorkName()

const wallRead = useNuxtData<ApiResult<ListWallComment>>(
  `api:wall:collection:${collectionId.value}`
)

const seo = computed(() => {
  const c = detail.value
  if (!c || c.visibility === 'private') {
    return null
  }
  const indexable = c.visibility === 'public' && c.content_rating !== 'r18'
  const coverWork = c.cover_work ?? c.recent_works[0]
  const shareImage =
    (c.content_rating === 'r18' || coverWork?.is_nsfw
      ? coverWork?.cover_sfw?.url
      : coverWork?.cover?.url) || undefined
  const pageUrl = `${kungal.domain.main}/collection/${c.id}`
  const ownerUrl = `${kungal.domain.main}/user/${c.owner.id}`
  const works = entries.value?.items ?? []
  const workNames = works.map((e) => nameOfWork(e.work_summary))
  const title = `${displayName.value} - ${ownerName.value} 的 Galgame 收藏夹`
  const description = truncateRunes(
    displayDescription.value
      ? `${displayDescription.value}（收录 ${c.item_count} 部作品）`
      : `${ownerName.value} 整理的 Galgame 收藏夹「${displayName.value}」，收录 ${c.item_count} 部作品` +
          (workNames.length ? `：${workNames.slice(0, 6).join('、')}` : '') +
          '。',
    175
  )
  const comments =
    wallRead.data.value?.ok === true
      ? wallRead.data.value.data.items.filter(
          (w) => w.state === 'visible' && !w.root_comment_id
        )
      : []

  const jsonLd: WithContext<CollectionPage> = {
    '@context': 'https://schema.org',
    '@type': 'CollectionPage',
    name: displayName.value,
    headline: displayName.value,
    description,
    url: pageUrl,
    ...(shareImage && { image: shareImage }),
    inLanguage: 'zh-CN',
    datePublished: c.created_at,
    dateModified: c.updated_at,
    author: {
      '@type': 'Person',
      name: ownerName.value,
      url: ownerUrl,
      ...(c.owner.avatar?.url && { image: c.owner.avatar.url })
    },
    keywords: ['Galgame 收藏夹', 'Galgame 推荐', ...workNames.slice(0, 8)].join(
      ', '
    ),
    interactionStatistic: [
      {
        '@type': 'InteractionCounter',
        interactionType: { '@type': 'WatchAction' },
        userInteractionCount: c.view_count
      },
      {
        '@type': 'InteractionCounter',
        interactionType: { '@type': 'FollowAction' },
        userInteractionCount: followerCount.value
      },
      ...(comments.length
        ? [
            {
              '@type': 'InteractionCounter' as const,
              interactionType: { '@type': 'CommentAction' as const },
              userInteractionCount: comments.length
            }
          ]
        : [])
    ],
    mainEntity: {
      '@type': 'ItemList',
      name: displayName.value,
      numberOfItems: c.item_count,
      itemListOrder:
        effectiveSort.value === 'position_asc'
          ? 'https://schema.org/ItemListUnordered'
          : 'https://schema.org/ItemListOrderDescending',
      itemListElement: works.map((e, i) => ({
        '@type': 'ListItem',
        position: (page.value - 1) * limit + i + 1,
        url: `${kungal.domain.main}/galgame/${e.work_summary.id}`,
        name: nameOfWork(e.work_summary),
        ...(e.note && { description: truncateRunes(e.note, 300) }),
        ...(e.work_summary.cover?.url &&
          !e.work_summary.is_nsfw && { image: e.work_summary.cover.url })
      }))
    },
    ...(comments.length && {
      comment: comments.slice(0, 10).map((w) => ({
        '@type': 'Comment',
        text: truncateRunes(contentPlainText(w.content).trim(), 1000),
        datePublished: w.created_at,
        url: `${pageUrl}#collection-comment-${w.id}`,
        author: {
          '@type': 'Person',
          name: w.author.name ?? deletedUserName,
          url: `${kungal.domain.main}/user/${w.author.id}`
        }
      }))
    })
  }

  const breadcrumbLd: WithContext<BreadcrumbList> = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: '首页',
        item: kungal.domain.main
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Galgame 收藏夹',
        item: `${kungal.domain.main}/collection`
      },
      {
        '@type': 'ListItem',
        position: 3,
        name: displayName.value,
        item: pageUrl
      }
    ]
  }

  return {
    c,
    title,
    description,
    jsonLd,
    breadcrumbLd,
    workNames,
    indexable,
    shareImage
  }
})

useHead(() => ({
  script: seo.value?.indexable
    ? [
        {
          id: 'schema-org-collection',
          type: 'application/ld+json',
          innerHTML: seo.value.jsonLd
        },
        {
          id: 'schema-org-breadcrumb',
          type: 'application/ld+json',
          innerHTML: seo.value.breadcrumbLd
        }
      ]
    : []
}))

useKunSeoMeta(() => {
  if (seo.value) {
    const { c, title, description, workNames, indexable, shareImage } =
      seo.value
    return {
      title,
      description,
      indexable,
      keywords: [
        'Galgame 收藏夹',
        displayName.value,
        ...workNames.slice(0, 5)
      ].join(','),
      ogType: 'article',
      ...(shareImage && { ogImage: shareImage }),
      articleAuthor: [`${kungal.domain.main}/user/${c.owner.id}`],
      articlePublishedTime: c.created_at,
      articleModifiedTime: c.updated_at
    }
  }
  if (detail.value) {
    return { noindex: true, title: displayName.value }
  }
  return problem.value ? { noindex: true, title: '未找到该收藏夹' } : {}
})
</script>

<template>
  <div>
    <div
      v-if="detail"
      class="grid grid-cols-1 items-start gap-6 lg:grid-cols-[minmax(0,1fr)_22rem] xl:grid-cols-[15rem_minmax(0,1fr)_22rem]"
    >
      <aside
        class="sticky top-20 hidden max-h-[calc(100dvh-6rem)] space-y-3 overflow-y-auto xl:block"
      >
        <KunHeader name="相关收藏夹" scale="h3" />
        <div class="space-y-1">
          <GalgameCollectionMiniCard
            v-for="c in related?.items ?? []"
            :key="c.id"
            :collection="c"
          />
        </div>
      </aside>

      <div class="min-w-0 space-y-6">
        <div class="space-y-4">
          <div class="space-y-2">
            <h1 class="text-3xl font-bold break-words">{{ displayName }}</h1>
            <p
              v-if="copySource"
              class="text-default-500 flex flex-wrap items-center gap-1 text-sm"
            >
              <KunIcon name="lucide:copy" class="size-4" />
              复制自
              <NuxtLink
                :to="`/collection/${copySource.id}`"
                class="text-primary hover:underline"
              >
                「{{ copySource.title || '原收藏夹' }}」
              </NuxtLink>
            </p>
            <p
              v-if="bodyDescription"
              class="text-default-600 break-words whitespace-pre-line"
            >
              {{ bodyDescription }}
            </p>
            <div v-if="detail.tag_names.length" class="flex flex-wrap gap-2">
              <NuxtLink
                v-for="tag in detail.tag_names"
                :key="tag"
                :to="{ path: '/collection', query: { tag } }"
                class="bg-default-100 text-default-600 hover:bg-primary/15 hover:text-primary rounded-md px-2 py-0.5 text-sm transition-colors"
              >
                #{{ tag }}
              </NuxtLink>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-x-5 gap-y-2">
            <KunLink
              :to="`/user/${detail.owner.id}`"
              underline="none"
              color="default"
              class-name="flex items-center gap-2 font-medium"
            >
              <KunAvatar
                :user="toKunUser(detail.owner)"
                size="sm"
                :is-navigation="false"
              />
              {{ ownerName }}
            </KunLink>
            <span
              class="text-default-500 inline-flex items-center gap-1 leading-none"
            >
              <KunIcon
                :name="visibilityMeta(detail.visibility).icon"
                class="size-4"
              />
              {{ visibilityMeta(detail.visibility).label }}
            </span>
            <GalgameCollectionStats
              :collection="{ ...detail, follower_count: followerCount }"
              size="lg"
            />
            <div v-if="hasMenu" class="ml-auto">
              <KunPopover ref="menu" position="bottom-end">
                <template #trigger>
                  <KunButton
                    is-icon-only
                    variant="light"
                    size="sm"
                    aria-label="更多操作"
                  >
                    <KunIcon name="lucide:ellipsis" />
                  </KunButton>
                </template>
                <div class="flex w-44 flex-col gap-1 p-2">
                  <KunButton
                    v-if="isShared || isOwner"
                    variant="light"
                    size="sm"
                    class-name="w-full justify-start gap-2"
                    :loading="copying"
                    @click="runMenu(copyCollection)"
                  >
                    <KunIcon name="lucide:copy" />
                    复制到我的收藏夹
                  </KunButton>
                  <KunButton
                    v-if="canEdit"
                    variant="light"
                    size="sm"
                    class-name="w-full justify-start gap-2"
                    @click="runMenu(() => (editOpen = true))"
                  >
                    <KunIcon name="lucide:pencil" />
                    编辑
                  </KunButton>
                  <KunButton
                    v-if="isShared && !isOwner"
                    variant="light"
                    size="sm"
                    class-name="w-full justify-start gap-2"
                    @click="runMenu(openReport)"
                  >
                    <KunIcon name="lucide:flag" />
                    举报
                  </KunButton>
                  <KunButton
                    v-if="canDelete"
                    variant="light"
                    color="danger"
                    size="sm"
                    class-name="w-full justify-start gap-2"
                    @click="runMenu(remove)"
                  >
                    <KunIcon name="lucide:trash-2" />
                    删除
                  </KunButton>
                </div>
              </KunPopover>
            </div>
          </div>

          <GalgameCollectionProgress v-if="progress" :progress="progress" />
        </div>

        <p
          v-if="hiddenCount > 0 && entries?.items.length"
          class="text-default-500 text-sm"
        >
          {{ hiddenNote }}
        </p>

        <div class="flex flex-wrap items-center gap-3">
          <GalgameCollectionFollowButton
            v-if="isShared && !isOwner"
            :collection-id="detail.id"
            :following="following"
            :follower-count="followerCount"
            @changed="onFollowChanged"
          />
          <p
            v-else-if="isOwner && detail.item_count > 1"
            class="text-default-500 text-sm"
          >
            {{
              canReorder
                ? '拖动左侧的把手调整顺序，点击备注按钮为作品写下推荐理由'
                : '切换到「自定义顺序」后可以拖动调整顺序'
            }}
          </p>
          <KunSelect
            v-if="detail.item_count > 1"
            :model-value="effectiveSort"
            :options="[...SORT_OPTIONS]"
            class-name="ml-auto w-48"
            aria-label="排序"
            @update:model-value="(value) => (sort = String(value))"
          />
        </div>

        <GalgameCollectionEntryListSkeleton v-if="entriesPending && !entries" />

        <div
          v-else-if="entries && entries.items.length"
          class="flex flex-col space-y-3"
        >
          <GalgameCollectionEntryList
            :collection-id="detail.id"
            :entries="entries.items"
            :can-edit-notes="canEdit"
            :can-reorder="canReorder"
            :can-report="isShared && !isOwner"
            :can-set-cover="isOwner"
            :cover-work-id="detail.cover_work?.id ?? null"
            :marks="marks"
            :can-mark="isSignedIn"
            @changed="refreshEntries"
            @marked="onMarked"
            @set-cover="setCover"
          />
          <KunPagination
            v-if="entries.total > limit"
            v-model:current-page="page"
            :total-page="Math.ceil(entries.total / limit)"
            :is-loading="entriesStatus === 'pending'"
          />
        </div>
        <KunNull
          v-else-if="entries"
          :description="
            hiddenCount > 0 ? hiddenNote : '这个收藏夹还没有收藏任何 Galgame'
          "
        />
      </div>

      <div
        class="min-w-0 lg:sticky lg:top-20 lg:max-h-[calc(100dvh-6rem)] lg:overflow-y-auto lg:pr-1"
      >
        <GalgameCollectionComment v-if="isShared" :collection-id="detail.id" />
      </div>

      <GalgameCollectionEditModal
        v-if="canEdit"
        v-model="editOpen"
        mode="edit"
        :collection-id="detail.id"
        :initial="editInitial"
        @saved="onEdited"
      />
      <GalgameCollectionReportModal
        v-if="isShared && !isOwner"
        v-model="reportOpen"
        :collection-id="detail.id"
      />
    </div>

    <GalgameCollectionDetailSkeleton v-else-if="pending" />

    <KunNull v-else description="收藏夹不存在或你没有权限查看" />
  </div>
</template>
