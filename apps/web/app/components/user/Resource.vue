<script setup lang="ts">
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP
} from '~/constants/galgameResource'
import {
  kunUserGalgameResourceNavItem,
  type KUN_USER_PAGE_GALGAME_RESOURCE_TYPE
} from '~/constants/user'
import type {
  GalgameResource,
  PageListGalgameResource
} from '#shared/utils/api/schemas'
import {
  resourceLanguageLabel,
  resourcePlatformLabel,
  resourceTypeLabel
} from '#shared/utils/galgameResourceVocab'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  userId: number
  type: (typeof KUN_USER_PAGE_GALGAME_RESOURCE_TYPE)[number]
}>()

const { allowsNsfw: includeNsfw } = useContentStance()
const workName = useWorkName()

const isCurrentUser = computed(() => usePersistUserStore().id === props.userId)
const canEdit = computed(
  () => isCurrentUser.value && props.type !== 'galgame_resource_like'
)
const activeTab = computed(() => props.type)
const pageData = reactive({
  page: usePageQuery(),
  limit: 50
})

const relation = computed(() =>
  props.type === 'galgame_resource_like'
    ? ('liked' as const)
    : ('published' as const)
)
const state = computed(() => {
  if (props.type === 'valid') {
    return 'valid' as const
  }
  if (props.type === 'expire') {
    return 'expired' as const
  }
  return undefined
})

const { data, status, refresh } = await useApi<PageListGalgameResource>(
  () =>
    `user-galgame-resources:${props.userId}:${relation.value}:${state.value ?? 'all'}:${pageData.page}:${pageData.limit}:${includeNsfw.value ? 'nsfw' : 'sfw'}`,
  (client, { signal }) =>
    client.GET('/users/{user_id}/galgame-resources', {
      params: {
        path: { user_id: String(props.userId) },
        query: {
          relation: relation.value,
          page: pageData.page,
          limit: pageData.limit,
          include_nsfw: includeNsfw.value,
          ...(state.value ? { state: state.value } : {})
        }
      },
      signal
    })
)

const editOpen = ref(false)
const editingWorkId = ref('')
const editingResourceId = ref<string | null>(null)

const workHref = (res: GalgameResource) =>
  res.work ? `/galgame/${res.work.id}?tab=resource` : ''

const workTitle = (res: GalgameResource) =>
  res.work ? workName(res.work) : res.title

const api = useApiClient()

const markValid = async (resourceId: string) => {
  const result = await settle(
    api.PATCH('/galgame-resources/{resource_id}', {
      params: { path: { resource_id: resourceId } },
      body: { state: 'valid' }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  useMessage(10548, 'success')
  return true
}

const afterLinkSave = async () => {
  if (props.type === 'expire' && editingResourceId.value) {
    await markValid(editingResourceId.value)
  }
  await refresh()
}

const handleMarkValid = async (res: GalgameResource) => {
  const ok = await useComponentMessageStore().alert(
    '您确定重新标记资源链接有效吗？',
    '资源链接修复后, 或核实过链接依然可用时, 可以重新标记为有效。'
  )
  if (ok && (await markValid(res.id))) {
    await refresh()
  }
}

const openEdit = (res: GalgameResource) => {
  if (!res.work) {
    return
  }
  editingWorkId.value = res.work.id
  editingResourceId.value = res.id
  editOpen.value = true
}
</script>

<template>
  <div class="space-y-3">
    <KunTab
      :items="kunUserGalgameResourceNavItem(userId)"
      :model-value="activeTab"
      variant="light"
      size="sm"
      scrollable
    />

    <div class="flex flex-col space-y-3" v-if="data && data.items.length">
      <KunCard
        v-for="res in data.items"
        :key="res.id"
        :is-hoverable="!canEdit"
        :href="canEdit ? undefined : workHref(res)"
      >
        <KunLink
          v-if="canEdit && res.work"
          :to="workHref(res)"
          underline="hover"
          color="default"
          class-name="mb-2 inline-flex items-center gap-1 text-lg font-medium hover:text-primary"
        >
          {{ workTitle(res) }}
          <KunIcon name="lucide:arrow-up-right" class="text-base opacity-60" />
        </KunLink>
        <div v-else class="mb-2 text-lg font-medium">
          {{ workTitle(res) }}
        </div>

        <div class="mb-2 flex flex-wrap items-center gap-2">
          <KunChip color="primary">
            <KunIcon :name="GALGAME_RESOURCE_TYPE_ICON_MAP[res.resource_type]" />
            {{ resourceTypeLabel(res.resource_type) }}
          </KunChip>
          <KunChip v-if="res.size" color="warning">
            <KunIcon name="lucide:database" />
            {{ res.size }}
          </KunChip>
          <KunChip
            v-for="platform in res.resource_platforms"
            :key="platform"
            color="success"
          >
            <KunIcon
              v-if="GALGAME_RESOURCE_PLATFORM_ICON_MAP[platform]"
              :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[platform]"
            />
            {{ resourcePlatformLabel(platform) }}
          </KunChip>
          <KunChip
            v-for="language in res.resource_languages"
            :key="language"
            color="secondary"
          >
            {{ resourceLanguageLabel(language) }}
          </KunChip>
          <KunChip :color="res.state === 'expired' ? 'danger' : 'success'">
            {{ res.state === 'expired' ? '链接过期' : '链接有效' }}
          </KunChip>
          <div class="text-default-500 text-sm">
            创建于 <KunTime :time="res.created_at" type="date" show-year />
          </div>
        </div>

        <div v-if="canEdit" class="flex justify-end gap-2">
          <KunButton
            v-if="res.state === 'expired'"
            variant="flat"
            color="success"
            @click="handleMarkValid(res)"
          >
            标记为有效
          </KunButton>
          <KunButton
            :color="props.type === 'expire' ? 'success' : 'primary'"
            :disabled="!res.work"
            @click="openEdit(res)"
          >
            {{ props.type === 'expire' ? '更改链接并标记为有效' : '更改链接' }}
          </KunButton>
        </div>
      </KunCard>

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull
      v-if="data && !data.items.length"
      description="这里暂无相关 Galgame 资源"
    />

    <GalgameResourceLinkEditModal
      v-model="editOpen"
      :work-id="editingWorkId"
      :resource-id="editingResourceId"
      :refresh="afterLinkSave"
    />
  </div>
</template>
