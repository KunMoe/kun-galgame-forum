<script setup lang="ts">
import {
  GALGAME_RESOURCE_TYPE_ICON_MAP,
  GALGAME_RESOURCE_PLATFORM_ICON_MAP
} from '~/constants/galgameResource'
import {
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP
} from '~/constants/galgame'
import {
  kunUserGalgameResourceNavItem,
  type KUN_USER_PAGE_GALGAME_RESOURCE_TYPE
} from '~/constants/user'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  userId: number
  type: (typeof KUN_USER_PAGE_GALGAME_RESOURCE_TYPE)[number]
}>()

const api = useApiClient()

const isCurrentUser = computed(() => usePersistUserStore().id === props.userId)
const activeTab = ref(props.type)
const pageData = reactive({
  page: usePageQuery(),
  limit: 50,
  type: props.type,
  userId: props.userId
})

const { data, status, refresh } = await useKunFetch<{
  resources: UserGalgameResource[]
  total: number
}>(`/user/${props.userId}/resources`, { query: pageData })

const draftLinks = ref<string[]>([])

watch(
  () => data.value?.resources,
  (list) => {
    if (!list) {
      return
    }
    draftLinks.value = list.map((res) => (res.link ? res.link.join('\n') : ''))
  },
  { immediate: true }
)

const submitFix = async (index: number) => {
  if (!data.value) {
    return
  }
  const res = data.value.resources[index]
  if (!res?.id || !res?.galgame_id) {
    return
  }

  const linkArray = draftLinks.value[index]!.split('\n')
    .map((s) => s.trim())
    .filter(Boolean)

  const result = await settle(
    api.PATCH('/galgame-resources/{resource_id}', {
      params: { path: { resource_id: String(res.id) } },
      body: {
        download_urls: linkArray,
        state: 'valid'
      }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }

  useMessage('更新资源链接成功', 'success')
  refresh()
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

    <div class="flex flex-col space-y-3" v-if="data && data.resources.length">
      <template v-if="props.type !== 'galgame_resource_like' && isCurrentUser">
        <KunCard
          :is-hoverable="false"
          v-for="(res, index) in data.resources"
          :key="index"
        >
          <KunLink
            :to="`/galgame/${res.galgame_id}?tab=resource`"
            underline="hover"
            color="default"
            class-name="mb-2 inline-flex items-center gap-1 text-lg font-medium hover:text-primary"
          >
            {{ res.galgame_name }}
            <KunIcon
              name="lucide:arrow-up-right"
              class="text-base opacity-60"
            />
          </KunLink>

          <div class="mb-2 flex flex-wrap items-center gap-2">
            <KunChip color="primary">
              <KunIcon :name="GALGAME_RESOURCE_TYPE_ICON_MAP[res.type]" />
              {{ KUN_GALGAME_RESOURCE_TYPE_MAP[res.type] }}
            </KunChip>
            <KunChip color="warning">
              <KunIcon name="lucide:database" />
              {{ res.size }}
            </KunChip>
            <KunChip color="success">
              <KunIcon
                :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[res.platform]"
              />
              {{ KUN_GALGAME_RESOURCE_PLATFORM_MAP[res.platform] }}
            </KunChip>
            <KunChip color="secondary">
              {{ KUN_GALGAME_RESOURCE_LANGUAGE_MAP[res.language] }}
            </KunChip>
            <KunChip color="danger">链接过期</KunChip>
            <div class="text-default-500 text-sm">
              创建于 <KunTime :time="res.created" type="date" show-year />
            </div>
          </div>

          <div class="space-y-2">
            <div class="text-default-500 text-sm">
              资源链接 (如果有多个资源链接, 请使用英语逗号分隔每一个链接)
            </div>
            <KunTextarea v-model="draftLinks[index]" :rows="2" auto-grow />
            <div class="flex justify-end">
              <KunButton
                :color="props.type === 'expire' ? 'success' : 'primary'"
                @click="submitFix(index)"
              >
                {{
                  props.type === 'expire'
                    ? '确定更改并将资源标记为有效'
                    : '更改链接'
                }}
              </KunButton>
            </div>
          </div>
        </KunCard>
      </template>

      <template v-else>
        <KunCard
          v-for="(res, index) in data.resources"
          :key="index"
          :href="`/galgame/${res.galgame_id}?tab=resource`"
        >
          <div>
            {{ res.galgame_name }}
          </div>

          <div class="flex items-center justify-between">
            <div class="space-x-2">
              <KunChip color="primary">
                <KunIcon
                  :name="GALGAME_RESOURCE_PLATFORM_ICON_MAP[res.platform]"
                />
                {{ KUN_GALGAME_RESOURCE_PLATFORM_MAP[res.platform] }}
              </KunChip>

              <KunChip :color="res.status ? 'danger' : 'success'">
                {{ res.status ? '链接过期' : '链接有效' }}
              </KunChip>
            </div>

            <div class="text-default-500 text-sm">
              <KunTime :time="res.created" type="date" show-year />
            </div>
          </div>
        </KunCard>
      </template>

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull
      v-if="data && !data.resources.length"
      description="这里暂无相关 Galgame 资源"
    />
  </div>
</template>
