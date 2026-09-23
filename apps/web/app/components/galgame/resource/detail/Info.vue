<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type {
  GalgameResource,
  GalgameResourceDownload
} from '#shared/utils/api/schemas'
import { contentPlainText } from '~/utils/contentPlainText'
import { toKunUser } from '~/utils/userRef'

const { id } = usePersistUserStore()
const api = useApiClient()

const props = defineProps<{
  resource: GalgameResource
  resourceTypeLabel: string
  refresh: () => void
}>()

const emits = defineEmits<{ downloaded: [] }>()

const isEditOpen = ref(false)
const author = computed(() => toKunUser(props.resource.author))
const workId = computed(() => props.resource.work?.id ?? '')
const hasNote = computed(
  () => contentPlainText(props.resource.content).trim().length > 0
)
const canEdit = computed(() => props.resource.viewer?.can_edit ?? false)
const canDelete = computed(() => props.resource.viewer?.can_delete ?? false)

const providerName = computed(() => {
  const names = props.resource.provider_names
  if (names.length > 0) {
    return names.join(' / ')
  }
  return ''
})

const isFetching = ref(false)
const download = ref<GalgameResourceDownload | null>(null)
const isResourceExpired = computed(() => props.resource.state === 'expired')
const isOwner = computed(() => Number(props.resource.author.id) === id)

const handleDeleteResource = async () => {
  const res = await useComponentMessageStore().alert(
    '您确定删除 Galgame 资源链接吗？',
    '这将会扣除您发布 Galgame 资源获得的 3 萌萌点，此操作不可撤销。'
  )
  if (!res) return

  isFetching.value = true
  const result = await settle(
    api.DELETE('/galgame-resources/{resource_id}', {
      params: { path: { resource_id: props.resource.id } }
    })
  )
  isFetching.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('删除资源成功', 'success')
  if (workId.value) {
    await navigateTo(`/galgame/${workId.value}`)
  }
}

const { status: reportStatus, report: reportExpire } =
  useReportResourceExpired()
const handleReportExpire = () =>
  reportExpire(props.resource.id, () => props.refresh())

const handleGetResourceLink = async () => {
  if (download.value) return

  isFetching.value = true
  const result = await settle(
    api.POST('/galgame-resources/{resource_id}/downloads', {
      params: { path: { resource_id: props.resource.id } }
    })
  )
  isFetching.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  download.value = result.data
  emits('downloaded')
}

const handleRewriteResource = () => {
  isEditOpen.value = true
}

const handleEditDone = () => {
  download.value = null
  props.refresh()
  isEditOpen.value = false
}
</script>

<template>
  <div class="flex h-full flex-col gap-3" v-if="resource">
    <div class="flex items-center gap-2">
      <KunAvatar :user="author" />
      <span>{{ author.name }}</span>
      <span class="text-default-500 text-sm">
        发布于 <KunTime :time="resource.created_at" />
      </span>
    </div>

    <KunInfo v-if="hasNote" variant="bordered" color="info" title="下载备注信息">
      <ContentDocument compact :document="resource.content" />
    </KunInfo>

    <KunAdAIFYBanner class-name="block lg:hidden" />

    <KunInfo
      :color="isResourceExpired ? 'warning' : 'success'"
      variant="bordered"
      class-name="relative"
    >
      <template #title>
        <div class="flex w-full flex-wrap items-center gap-2">
          <span>
            {{ `${props.resourceTypeLabel}下载链接` }}
          </span>
          <span v-if="providerName" class="text-default-500 text-sm">{{
            providerName
          }}</span>
          <KunButton
            class-name="ml-auto whitespace-nowrap"
            :color="isResourceExpired ? 'warning' : 'success'"
            :loading="isFetching"
            @click="handleGetResourceLink"
          >
            获取链接
          </KunButton>
        </div>
      </template>

      <template #default v-if="download">
        <div class="space-y-3">
          <p
            v-if="!download.download_urls.length"
            class="text-default-500 text-sm"
          >
            这个资源没有登记下载链接，链接可能写在提取码或备注里
          </p>
          <template v-else>
            <p class="text-default-500 text-sm">点击下面的链接以下载</p>
            <KunLink
              v-for="(kun, index) in download.download_urls"
              :key="index"
              :to="kun"
              target="_blank"
              rel="noopener noreferrer"
              :is-show-anchor-icon="true"
            >
              {{ kun }}
            </KunLink>
          </template>

          <div class="flex items-center gap-2">
            <KunCopy
              variant="solid"
              :color="isResourceExpired ? 'warning' : 'success'"
              v-if="download.extraction_code"
              :name="`提取码 ${download.extraction_code}`"
              :text="download.extraction_code"
            />
            <KunCopy
              variant="solid"
              :color="isResourceExpired ? 'warning' : 'success'"
              v-if="download.archive_password"
              :name="`解压码 ${download.archive_password}`"
              :text="download.archive_password"
            />
          </div>

          <GalgameResourceBuyLegitNotice
            :work-id="workId"
            :purchase-url="resource.dlsite?.purchase_url"
            :coupon-url="resource.dlsite?.coupon_url ?? undefined"
          />

          <div class="flex justify-end">
            <KunChip
              :color="isResourceExpired ? 'danger' : 'success'"
              variant="solid"
            >
              {{
                isResourceExpired
                  ? '该资源链接被其它用户标记为失效'
                  : '该资源链接可用'
              }}
            </KunChip>
          </div>
        </div>
      </template>
    </KunInfo>

    <KunInfo title="鲲的小请求">
      <p>
        在您下载这部 Galgame 并游玩之后, 可否请您在本网站的
        <KunLink size="sm" :to="`/galgame/${workId}`">
          Galgame 评分页面
        </KunLink>
        为这部 Galgame 提交一个评分, 这将有助于我们把优秀的 Galgame
        推荐给更多人, 谢谢您的支持
      </p>
    </KunInfo>

    <div class="mt-auto flex flex-wrap items-center justify-end gap-1">
      <KunButton
        variant="flat"
        @click="handleRewriteResource"
        :loading="isFetching"
        v-if="canEdit"
      >
        编辑资源
        <KunIcon name="lucide:pencil" />
      </KunButton>
      <KunButton
        color="danger"
        variant="flat"
        @click="handleDeleteResource"
        :loading="isFetching"
        v-if="canDelete"
      >
        删除资源
        <KunIcon name="lucide:trash-2" />
      </KunButton>

      <div v-if="!isOwner && resource.state !== 'expired'">
        <KunButton
          variant="flat"
          color="danger"
          @click="handleReportExpire"
          :loading="reportStatus === 'checking'"
          :disabled="reportStatus === 'checking'"
        >
          报告链接过期
        </KunButton>
      </div>

      <KunButton variant="flat" :href="`/galgame/${workId}`">
        反馈资源问题
      </KunButton>
    </div>

    <GalgameResourceExpireStatus :status="reportStatus" class="mt-3" />

    <GalgameResourceLinkEditModal
      v-model="isEditOpen"
      :work-id="workId"
      :resource-id="resource.id"
      :refresh="handleEditDone"
    />
  </div>
</template>
