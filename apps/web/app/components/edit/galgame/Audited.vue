<script setup lang="ts">
const { data, items, hasMore, isLoadingMore, loadMore } =
  await useGalgameClaimList('/galgame/audited')

const previewWorkId = ref(0)
const previewState = ref('')
const isPreviewOpen = ref(false)

const openPreview = (item: UserClaimItem) => {
  previewWorkId.value = item.work_id
  previewState.value = item.claim_state
  isPreviewOpen.value = true
}
</script>

<template>
  <div class="space-y-4">
    <KunHeader
      name="我的 Galgame 审核"
      description="您审核过的 Galgame 申请, 包括已通过 / 已拒绝 / 已下架的作品。"
    >
      <template #endContent>
        <div class="flex gap-2">
          <KunLink to="/admin/submissions">
            <KunButton size="sm">审核队列</KunButton>
          </KunLink>
          <KunLink to="/edit/galgame/mine">
            <KunButton size="sm" variant="flat">我的提交</KunButton>
          </KunLink>
        </div>
      </template>
    </KunHeader>

    <KunDivider />

    <KunInfo
      v-if="!data"
      color="danger"
      title="加载失败"
      description="无法获取您的审核列表, 可能是后端 / Galgame 资料库暂时不可用, 请稍后重试。"
    />

    <div v-else-if="items.length" class="flex flex-col gap-3">
      <EditGalgameClaimRow
        v-for="item in items"
        :key="item.work_id"
        :item="item"
        time-label="首次审核"
      >
        <template #note>
          <div
            v-if="item.last_reason"
            class="text-default-500 bg-default-500/10 mt-1 rounded-md px-2 py-1 text-sm"
          >
            审核理由: {{ item.last_reason }}
          </div>
        </template>

        <template #actions>
          <KunLink
            v-if="item.work_id && isPublicState(item.claim_state)"
            :to="`/galgame/${item.work_id}`"
          >
            <KunButton size="sm" variant="flat">查看</KunButton>
          </KunLink>
          <KunButton
            v-else-if="item.work_id"
            size="sm"
            variant="flat"
            @click="openPreview(item)"
          >
            预览
          </KunButton>
        </template>
      </EditGalgameClaimRow>
    </div>

    <KunNull v-else />

    <KunButton
      v-if="hasMore"
      variant="flat"
      :loading="isLoadingMore"
      @click="loadMore"
    >
      加载更多
    </KunButton>

    <GalgamePreviewModal
      v-if="previewWorkId"
      v-model="isPreviewOpen"
      :work-id="previewWorkId"
      :claim-state="previewState"
    />
  </div>
</template>
