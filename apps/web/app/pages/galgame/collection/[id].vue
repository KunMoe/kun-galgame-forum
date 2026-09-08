<script setup lang="ts">
const route = useRoute()
const collectionId = computed(() => Number((route.params as { id: string }).id))

const page = ref(1)
const limit = 24

const {
  data: detail,
  status,
  refresh
} = await useKunFetch<CollectionDetail>(
  () => `/galgame/collection/${collectionId.value}`,
  {
    query: computed(() => ({ page: page.value, limit })),
    watch: [page]
  }
)

const displayName = computed(() =>
  detail.value
    ? collectionDisplayName(detail.value, detail.value.owner.name)
    : '收藏夹'
)
const displayDescription = computed(() =>
  detail.value
    ? collectionDisplayDescription(detail.value, detail.value.owner.name)
    : ''
)

if (detail.value) {
  useKunSeoMeta({
    title: displayName.value,
    description: displayDescription.value || `${displayName.value}`
  })
} else {
  useKunDisableSeo('未找到该收藏夹')
}

const canEditAnyCollection = useCan('collection.edit_any')
const canDeleteAnyCollection = useCan('collection.delete_any')
const canEdit = computed(
  () => !!detail.value?.is_owner || canEditAnyCollection.value
)
const canDelete = computed(
  () => !!detail.value?.is_owner || canDeleteAnyCollection.value
)

const visibilityMeta = (v?: CollectionVisibility) =>
  v === 'private'
    ? { icon: 'lucide:lock', label: '私密' }
    : { icon: 'lucide:globe', label: '公开' }

const editOpen = ref(false)
const editInitial = computed(() => ({
  name: detail.value?.name ?? '',
  description: detail.value?.description ?? '',
  visibility: detail.value?.visibility ?? ('public' as CollectionVisibility),
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
  const res = await kunFetch(`/galgame/collection/${collectionId.value}`, {
    method: 'DELETE'
  })
  if (!res) {
    return
  }
  useMessage(10568, 'success')
  navigateTo(`/user/${detail.value?.owner.id}/collection/galgame`)
}
</script>

<template>
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
                :src="detail.owner.avatar"
                :alt="detail.owner.name"
                class="size-6 rounded-full object-cover"
              />
              {{ detail.owner.name }}
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
              v-if="canDelete && !detail.is_default"
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

    <div v-if="detail.galgames.length" class="flex flex-col space-y-3">
      <GalgameCard :is-transparent="false" :galgames="detail.galgames" />
      <KunPagination
        v-if="detail.total > limit"
        v-model:current-page="page"
        :total-page="Math.ceil(detail.total / limit)"
        :is-loading="status === 'pending'"
      />
    </div>
    <KunNull v-else description="这个收藏夹还没有收藏任何 Galgame" />

    <GalgameCollectionEditModal
      v-if="canEdit"
      v-model="editOpen"
      mode="edit"
      :collection-id="detail.id"
      :initial="editInitial"
      @saved="onEdited"
    />
  </div>

  <KunNull v-else description="收藏夹不存在或你没有权限查看" />
</template>
