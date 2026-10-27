<script setup lang="ts">
import {
  GALGAME_RESOURCE_PROVIDER_BUCKETS,
  bucketizeResourceProvider,
  type GalgameResourceProviderBucketKey
} from '~/constants/galgameResource'
import type {
  GalgameResource,
  ListLetmoeResource,
  Work
} from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { kunFeedUrl } from '#shared/utils/feedUrl'

const route = useRoute()
const workId = computed(() => String((route.params as { id: string }).id))
const config = useRuntimeConfig()
const rssHref = computed(() =>
  kunFeedUrl(
    config.public.KUN_GALGAME_URL || '',
    `/rss/galgame/${workId.value}`
  )
)

const resourcePublishBanned = inject<Ref<boolean>>(
  'galgameResourcePublishBanned',
  ref(false)
)

const galgame = inject<Work>('galgame')
const isHostedElsewhere = computed(
  () => galgame?.is_resource_hosted_elsewhere ?? false
)
const letmoeHref = computed(() => `https://www.letmoe.com/game/${workId.value}`)

const isShowPublish = ref(false)
const emptyDescription = computed(() => {
  if (isHostedElsewhere.value) {
    return '本站不收录这部作品的资源, 请前往 LetMoe·一起萌 查看和发布'
  }
  if (resourcePublishBanned.value) {
    return '本游戏已禁止发布下载资源'
  }
  return '这个 Galgame 还没有资源链接, 快添加一个吧!'
})
const { id } = usePersistUserStore()

const emit = defineEmits<{
  'update:loading': [boolean]
}>()

const resourceRank = useState<Record<string, number>>(
  'galgame-resource-rank',
  () => ({})
)

const shuffle = (items: GalgameResource[]) => {
  for (const item of items) {
    resourceRank.value[item.id] ??= Math.random()
  }
  return items.sort(
    (a, b) =>
      Number(a.state === 'expired') - Number(b.state === 'expired') ||
      resourceRank.value[a.id]! - resourceRank.value[b.id]!
  )
}

const GUIDE_AUTHOR_ID = 20668
const { entry: guideAuthorCard, load: loadGuideAuthor } =
  useUserCard(GUIDE_AUTHOR_ID)
const guideAuthor = computed<KunUser>(() => {
  const card = guideAuthorCard.value
  return {
    id: GUIDE_AUTHOR_ID,
    name: '大伊兜子',
    avatar: card?.state === 'ready' ? (card.profile.avatar?.url ?? '') : ''
  }
})
onMounted(loadGuideAuthor)

const { data, status, refresh, problem } = await useApi<GalgameResource[]>(
  () => `work-resources:${workId.value}`,
  async (api, { signal }) => {
    const limit = 100
    const first = await api.GET('/works/{work_id}/resources', {
      params: {
        path: { work_id: workId.value },
        query: { page: 1, limit }
      },
      signal
    })
    if (first.error || !first.data) {
      return { error: first.error, response: first.response }
    }
    const items = [...first.data.items]
    let page = 2
    while (items.length < first.data.total) {
      const next = await api.GET('/works/{work_id}/resources', {
        params: {
          path: { work_id: workId.value },
          query: { page, limit }
        },
        signal
      })
      if (next.error || !next.data) {
        return { error: next.error, response: next.response }
      }
      if (!next.data.items.length) {
        break
      }
      items.push(...next.data.items)
      page += 1
    }
    return { data: shuffle(items), response: first.response }
  }
)
watchEffect(() => emit('update:loading', status.value === 'pending'))

useHead({
  link: computed(() =>
    data.value?.length
      ? [
          {
            rel: 'alternate',
            type: 'application/rss+xml',
            title: 'Galgame 资源订阅',
            href: rssHref.value
          }
        ]
      : []
  )
})

const { data: letmoe, status: letmoeStatus } = useApi<ListLetmoeResource>(
  () => `work-letmoe-resources:${workId.value}`,
  (api, { signal }) =>
    api.GET('/works/{work_id}/letmoe-resources', {
      params: { path: { work_id: workId.value } },
      signal
    }),
  { lazy: true }
)
const letmoeDirect = computed(
  () => letmoe.value?.items.filter((r) => r.has_direct_download) ?? []
)
const letmoeNetdisk = computed(
  () =>
    letmoe.value?.items.filter((r) =>
      r.providers.some((p) => p !== 'direct')
    ) ?? []
)
const isLoading = computed(
  () => status.value === 'pending' || letmoeStatus.value === 'pending'
)
const hasAnyResource = computed(
  () =>
    !!data.value?.length ||
    !!letmoeDirect.value.length ||
    !!letmoeNetdisk.value.length
)
const netdiskCount = computed(
  () => letmoeNetdisk.value.length + (data.value?.length ?? 0)
)

type DownloadKind = 'direct' | 'netdisk'
const pickedKind = ref<DownloadKind | null>(null)
const hasDirect = computed(() => letmoeDirect.value.length > 0)
const activeKind = computed<DownloadKind>({
  get: () => (hasDirect.value ? (pickedKind.value ?? 'direct') : 'netdisk'),
  set: (kind) => {
    pickedKind.value = kind
  }
})
const kindTabs = computed(() => [
  {
    value: 'direct',
    textValue: `直链下载 (${letmoeDirect.value.length})`,
    icon: 'lucide:zap'
  },
  {
    value: 'netdisk',
    textValue: `网盘下载 (${netdiskCount.value})`,
    icon: 'lucide:cloud',
    disabled: !netdiskCount.value
  }
])

const groupedResources = computed(() => {
  const grouped: Record<GalgameResourceProviderBucketKey, GalgameResource[]> = {
    baidu: [],
    quark: [],
    caiyun: [],
    pan123: [],
    xunlei: [],
    lanzou: [],
    other: []
  }
  for (const r of data.value ?? []) {
    grouped[bucketizeResourceProvider(r.provider_names)].push(r)
  }
  return GALGAME_RESOURCE_PROVIDER_BUCKETS.flatMap((bucket) => {
    const items = grouped[bucket.key]
    return items.length ? [{ ...bucket, items }] : []
  })
})

const providerTabs = computed(() => [
  ...(letmoeNetdisk.value.length
    ? [
        {
          value: 'letmoe',
          textValue: `letmoe (${letmoeNetdisk.value.length})`,
          icon: 'lucide:heart-handshake'
        }
      ]
    : []),
  ...groupedResources.value.map((g) => ({
    value: g.key,
    textValue: `${g.label} (${g.items.length})`,
    icon: g.icon
  }))
])

type ProviderTab = GalgameResourceProviderBucketKey | 'letmoe'
const pickedProvider = ref<ProviderTab | null>(null)
const activeProvider = computed<ProviderTab | ''>({
  get: () => {
    const tabs = providerTabs.value
    if (tabs.some((t) => t.value === pickedProvider.value)) {
      return pickedProvider.value!
    }
    return (tabs[0]?.value ?? '') as ProviderTab | ''
  },
  set: (key) => {
    pickedProvider.value = key || null
  }
})

const activeBucket = computed(() =>
  groupedResources.value.find((g) => g.key === activeProvider.value)
)
</script>

<template>
  <div class="space-y-3">
    <KunInfo
      v-if="isHostedElsewhere"
      color="warning"
      variant="bordered"
      title="这部作品的资源请到 LetMoe·一起萌 发布"
    >
      <div class="space-y-3">
        <p class="text-sm">
          鲲 Galgame 论坛只收录原语言为日语或中文的 Galgame 的下载资源,
          这部作品不能在本站发布资源。它的资源, 以及 SLG / RPG 等同人游戏、3D
          游戏, 请前往
          <GalgameResourceLetmoeLink :to="letmoeHref" />
          发布和下载, 那里与本站是同一个账号、同一个作品编号。
        </p>
        <KunButton
          :href="letmoeHref"
          target="_blank"
          rel="noopener noreferrer"
          color="warning"
          variant="solid"
        >
          <KunIcon name="lucide:upload" />
          前往 LetMoe 发布资源
        </KunButton>
      </div>
    </KunInfo>
    <KunInfo v-else color="warning" title="鲲 Galgame 论坛只提供 Galgame">
      <p class="text-sm">
        如果要获取或发布 SLG / RPG 等同人游戏、3D 游戏, 请前往
        <GalgameResourceLetmoeLink />
      </p>
    </KunInfo>
    <KunInfo
      v-if="resourcePublishBanned"
      color="danger"
      title="本游戏已禁止发布下载资源"
      description="部分游戏可能因为版权方通知，或者其余第三方原因导致不可用，已禁止在本游戏下发布任何下载资源。"
    />
    <KunHeader name="Galgame 资源链接" scale="h2">
      <template #headerEndContent>
        <div class="ml-auto flex items-center gap-1">
          <KunButton
            v-if="id"
            :href="`/user/${id}/resource/expire`"
            color="success"
            variant="flat"
          >
            批量更改已失效资源链接
          </KunButton>
          <KunButton
            v-if="isHostedElsewhere"
            :href="letmoeHref"
            target="_blank"
            rel="noopener noreferrer"
            color="warning"
          >
            <KunIcon name="lucide:external-link" />
            前往 LetMoe 发布资源
          </KunButton>
          <KunButton
            v-else-if="!resourcePublishBanned"
            @click="isShowPublish = !isShowPublish"
          >
            添加资源
          </KunButton>
        </div>
      </template>

      <template #endContent>
        <KunInfo
          color="info"
          title="一些小提示以及帮助文档"
          description="部分资源链接可能需要网络代理"
        >
          <div class="mb-1 flex items-center gap-1">
            <KunLink class-name="inline" size="sm" to="/topic/2431">
              Galgame萌新入门(待补充)
            </KunLink>
            - by
            <UserHoverCard :user-id="GUIDE_AUTHOR_ID">
              <KunUserChip size="sm" :user="guideAuthor" />
            </UserHoverCard>
          </div>

          <div class="flex items-center gap-1">
            <KunLink class-name="inline" size="sm" to="/topic/2522">
              如何安装镜像文件(教程)
            </KunLink>
            - by
            <UserHoverCard :user-id="GUIDE_AUTHOR_ID">
              <KunUserChip size="sm" :user="guideAuthor" />
            </UserHoverCard>
          </div>
        </KunInfo>
      </template>
    </KunHeader>

    <KunAdAIFYBanner />

    <KunNull v-if="problem" :description="problemMessage(problem)" />

    <KunNull
      v-else-if="!isLoading && !hasAnyResource"
      :description="emptyDescription"
    />

    <GalgameResourceLinkEditModal
      v-model="isShowPublish"
      :work-id="workId"
      :refresh="refresh"
    />

    <template v-if="!isLoading && hasAnyResource">
      <KunTab
        v-if="hasDirect"
        v-model="activeKind"
        :items="kindTabs"
        variant="solid"
        color="primary"
        size="md"
      />

      <div v-if="activeKind === 'direct'" class="space-y-3">
        <GalgameResourceLetmoeSourceNotice kind="direct" />
        <GalgameResourceLetmoeCard
          v-for="resource in letmoeDirect"
          :key="resource.id"
          :resource="resource"
          mode="direct"
        />
      </div>

      <template v-else>
        <KunTab
          v-if="providerTabs.length > 1"
          v-model="activeProvider"
          :items="providerTabs"
          variant="light"
          color="primary"
          size="md"
          scrollable
        />
        <div v-if="activeProvider === 'letmoe'" class="space-y-3">
          <GalgameResourceLetmoeSourceNotice kind="netdisk" />
          <GalgameResourceLetmoeCard
            v-for="resource in letmoeNetdisk"
            :key="resource.id"
            :resource="resource"
            mode="netdisk"
          />
        </div>
        <div v-else-if="activeBucket" class="space-y-3">
          <GalgameResourceLink
            v-for="resource in activeBucket.items"
            :key="resource.id"
            :resource="resource"
            :refresh="refresh"
          />
        </div>
      </template>
    </template>

    <KunLoading v-if="isLoading" />
  </div>
</template>
