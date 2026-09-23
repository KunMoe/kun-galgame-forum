<script setup lang="ts">
const { id } = usePersistUserStore()
const canEditAnyResource = useCan('resource.edit_any')
const canDeleteAnyResource = useCan('resource.delete_any')

const props = defineProps<{
  resource: GalgameResource
  resourceTypeLabel: string
  refresh: () => void
}>()

const isEditOpen = ref(false)

const providerName = computed(() => {
  const names = props.resource.provider_names
  if (names && names.length > 0) {
    return names.join(' / ')
  }
  return props.resource.link_domain
})

const isFetching = ref(false)
const detail = ref<null | GalgameResourceDetailLink>(null)
const isResourceExpired = computed(() => props.resource.status === 1)

const handleDeleteResource = async () => {
  const res = await useComponentMessageStore().alert(
    '您确定删除 Galgame 资源链接吗？',
    '这将会扣除您发布 Galgame 资源获得的 5 萌萌点，并且扣除其它人对资源链接的点赞影响（萌萌点和点赞数减一），此操作不可撤销。'
  )
  if (!res) return

  const result = await kunFetch(
    `/galgame/${props.resource.galgame_id}/resource`,
    {
      method: 'DELETE',
      query: { galgame_resource_id: props.resource.id }
    }
  )

  if (result) {
    useMessage('删除资源成功', 'success')
    await navigateTo(`/galgame/${props.resource.galgame_id}`)
  }
}

const { status: reportStatus, report: reportExpire } =
  useReportResourceExpired()
const handleReportExpire = () =>
  reportExpire(props.resource.galgame_id, props.resource.id, () =>
    props.refresh()
  )

const handleGetResourceLink = async () => {
  if (detail.value) return

  isFetching.value = true
  const result = await kunFetch<GalgameResourceDetailLink>(
    `/galgame-resource/${props.resource.id}/detail`,
    {
      method: 'GET',
      query: { galgame_resource_id: props.resource.id }
    }
  )
  isFetching.value = false

  if (result) {
    detail.value = result
    props.refresh()
    return result
  }
}

const handleRewriteResource = async () => {
  if (!detail.value) {
    const res = await handleGetResourceLink()
    if (!res) return
  }
  isEditOpen.value = true
}

const handleEditDone = () => {
  detail.value = null
  props.refresh()
  isEditOpen.value = false
}
</script>

<template>
  <div class="flex h-full flex-col gap-3" v-if="resource">
    <div class="flex items-center gap-2">
      <KunAvatar :user="resource.user" />
      <span>{{ resource.user.name }}</span>
      <span class="text-default-500 text-sm">
        发布于 <KunTime :time="resource.created" />
      </span>
    </div>

    <KunInfo
      variant="bordered"
      v-if="resource.note"
      color="info"
      title="下载备注信息"
    >
      <KunContent compact :content="renderKatex(resource.note_html)" />
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
          <span class="text-default-500 text-sm">{{ providerName }}</span>
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

      <template #default v-if="detail">
        <div class="space-y-3">
          <p class="text-default-500 text-sm">点击下面的链接以下载</p>
          <KunLink
            v-for="(kun, index) in detail.link"
            :key="index"
            :to="kun"
            target="_blank"
            rel="noopener noreferrer"
            :is-show-anchor-icon="true"
          >
            {{ kun }}
          </KunLink>

          <div class="flex items-center gap-2">
            <KunCopy
              variant="solid"
              :color="isResourceExpired ? 'warning' : 'success'"
              v-if="detail.code"
              :name="`提取码 ${detail.code}`"
              :text="detail.code"
            />
            <KunCopy
              variant="solid"
              :color="isResourceExpired ? 'warning' : 'success'"
              v-if="detail.password"
              :name="`解压码 ${detail.password}`"
              :text="detail.password"
            />
          </div>

          <GalgameResourceBuyLegitNotice
            :work-id="resource.galgame_id"
            :purchase-url="resource.dlsite_purchase_url"
            :coupon-url="resource.dlsite_coupon_url"
            :campaign-name="resource.dlsite_campaign_name"
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
        <KunLink size="sm" :to="`/galgame/${resource.galgame_id}`">
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
        v-if="resource.user.id === id || canEditAnyResource"
      >
        编辑资源
        <KunIcon name="lucide:pencil" />
      </KunButton>
      <KunButton
        color="danger"
        variant="flat"
        @click="handleDeleteResource"
        :loading="isFetching"
        v-if="resource.user.id === id || canDeleteAnyResource"
      >
        删除资源
        <KunIcon name="lucide:trash-2" />
      </KunButton>

      <div v-if="id !== resource.user.id && !resource.status">
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

      <KunButton variant="flat" :href="`/galgame/${resource.galgame_id}`">
        反馈资源问题
      </KunButton>
    </div>

    <GalgameResourceExpireStatus :status="reportStatus" class="mt-3" />

    <GalgameResourceLinkEditModal
      v-if="detail"
      v-model="isEditOpen"
      :work-id="resource.galgame_id"
      :resource="detail"
      :refresh="handleEditDone"
    />
  </div>
</template>
