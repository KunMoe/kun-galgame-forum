<script setup lang="ts">
import { settle, isMissingProblem } from '#shared/utils/api/problem'
import { deletedUserName } from '#shared/utils/deletedUser'
import type {
  Collection,
  CollectionVisibility,
  WorkPage
} from '#shared/utils/api/schemas'

const route = useRoute()
const collectionId = computed(() => String((route.params as { id: string }).id))
const { allowsNsfw, stanceKey } = useContentStance()
const api = useApiClient()
const limit = 24

const SORT_OPTIONS = [
  { value: 'favorited_desc', label: '最近收藏' },
  { value: 'favorited_asc', label: '最早收藏' },
  { value: 'released_desc', label: '发售日期 (新到旧)' },
  { value: 'released_asc', label: '发售日期 (旧到新)' }
] as const
const { sort, page } = useQueryState(
  {
    sort: queryEnum(
      SORT_OPTIONS.map((o) => o.value),
      'favorited_desc'
    ),
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

const {
  data: works,
  status: worksStatus,
  pending: worksPending
} = useKunApi<WorkPage>(
  () =>
    `collection-works:${collectionId.value}:${sort.value}:${page.value}:${limit}:${stanceKey.value}`,
  (client, { signal }) =>
    client.GET('/collections/{collection_id}/works', {
      params: {
        path: { collection_id: collectionId.value },
        query: {
          page: page.value,
          limit,
          include_nsfw: allowsNsfw.value,
          sort: sort.value
        }
      },
      signal
    })
)

const { data: detail, problem, pending, refresh } = await collectionRead
useKunNotFound(() => isMissingProblem(problem.value), '未找到这个合集')

const galgames = useWorkCards(() => works.value?.items)

const displayName = computed(() =>
  detail.value
    ? collectionDisplayName(
        detail.value,
        detail.value.owner.name ?? deletedUserName
      )
    : '收藏夹'
)
const displayDescription = computed(() =>
  detail.value
    ? collectionDisplayDescription(
        detail.value,
        detail.value.owner.name ?? deletedUserName
      )
    : ''
)

useKunSeoMeta(() => {
  if (detail.value) {
    return {
      title: displayName.value,
      description: displayDescription.value || displayName.value
    }
  }
  return problem.value ? { noindex: true, title: '未找到该收藏夹' } : {}
})

const hiddenCount = computed(() =>
  detail.value && works.value
    ? Math.max(detail.value.item_count - works.value.total, 0)
    : 0
)
const hiddenNote = computed(() =>
  allowsNsfw.value
    ? `另有 ${hiddenCount.value} 部作品已下架，未显示`
    : `另有 ${hiddenCount.value} 部作品未显示：受你的内容显示设置影响，或已下架`
)

const canEdit = computed(() => !!detail.value?.viewer?.can_edit)
const canDelete = computed(() => !!detail.value?.viewer?.can_delete)

const visibilityMeta = (v?: CollectionVisibility) =>
  v === 'private'
    ? { icon: 'lucide:lock', label: '私密' }
    : { icon: 'lucide:globe', label: '公开' }

const editOpen = ref(false)
const editInitial = computed(() => ({
  title: detail.value?.title ?? '',
  description: detail.value?.description ?? '',
  visibility: detail.value?.visibility ?? ('public' as CollectionVisibility)
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
</script>

<template>
  <div>
    <div v-if="detail" class="space-y-6">
      <KunHeader :name="displayName" :description="displayDescription">
        <template #endContent>
          <div class="space-y-3">
            <div
              class="text-default-500 flex flex-wrap items-center gap-3 text-sm"
            >
              <KunLink
                :to="`/user/${detail.owner.id}`"
                underline="none"
                color="default"
                class-name="flex items-center gap-1.5"
              >
                <img
                  :src="detail.owner.avatar?.url ?? ''"
                  :alt="detail.owner.name ?? deletedUserName"
                  class="size-6 rounded-full object-cover"
                />
                {{ detail.owner.name ?? deletedUserName }}
              </KunLink>
              <span class="flex items-center gap-1">
                <KunIcon :name="visibilityMeta(detail.visibility).icon" />
                {{ visibilityMeta(detail.visibility).label }}
              </span>
              <span>{{ detail.item_count }} 个游戏</span>
            </div>

            <div v-if="canEdit || canDelete" class="flex gap-2">
              <KunButton
                v-if="canEdit"
                variant="light"
                size="sm"
                @click="editOpen = true"
              >
                <KunIcon name="lucide:pencil" />
                编辑
              </KunButton>
              <KunButton
                v-if="canDelete"
                variant="light"
                color="danger"
                size="sm"
                @click="remove"
              >
                <KunIcon name="lucide:trash-2" />
                删除
              </KunButton>
            </div>
          </div>
        </template>
      </KunHeader>

      <p
        v-if="hiddenCount > 0 && works?.items.length"
        class="text-default-500 text-sm"
      >
        {{ hiddenNote }}
      </p>

      <div v-if="detail.item_count > 1" class="flex justify-end">
        <KunSelect
          v-model="sort"
          :options="[...SORT_OPTIONS]"
          class-name="w-48"
          aria-label="排序"
        />
      </div>

      <GalgameCardGridSkeleton v-if="worksPending" />

      <div
        v-else-if="works && works.items.length"
        class="flex flex-col space-y-3"
      >
        <GalgameCard :is-transparent="false" :galgames="galgames" />
        <KunPagination
          v-if="works.total > limit"
          v-model:current-page="page"
          :total-page="Math.ceil(works.total / limit)"
          :is-loading="worksStatus === 'pending'"
        />
      </div>
      <KunNull
        v-else-if="works"
        :description="
          hiddenCount > 0 ? hiddenNote : '这个收藏夹还没有收藏任何 Galgame'
        "
      />

      <GalgameCollectionEditModal
        v-if="canEdit"
        v-model="editOpen"
        mode="edit"
        :collection-id="detail.id"
        :initial="editInitial"
        @saved="onEdited"
      />
    </div>

    <div v-else-if="pending" class="space-y-6" aria-hidden="true">
      <div class="space-y-2">
        <KunSkeleton variant="text" width="14rem" class-name="text-2xl" />
        <KunSkeleton variant="text" width="50%" />
        <KunSkeleton variant="text" width="16rem" class-name="text-sm" />
      </div>
      <GalgameCardGridSkeleton />
    </div>

    <KunNull v-else description="收藏夹不存在或你没有权限查看" />
  </div>
</template>
