<script setup lang="ts">
import { galgameEditLabel } from '~/constants/galgameEdit'

useKunDisableSeo('我的资料编辑提案')

const { data, status, refresh } = await useKunFetch<GalgameEditProposalList>(
  '/galgame-edit/mine',
  { method: 'GET' }
)

const briefName = (item: GalgameEditProposalItem): string => {
  if (!item.galgame) {
    return `Galgame #${item.gid}`
  }
  return item.galgame.name || `Galgame #${item.gid}`
}

const withdrawing = ref(false)
const handleWithdraw = async (id: number) => {
  if (withdrawing.value) {
    return
  }
  withdrawing.value = true
  const result = await kunFetch<unknown>(
    `/galgame-edit/proposals/${id}/withdraw`,
    { method: 'POST' }
  )
  withdrawing.value = false
  if (result) {
    useMessage('提案已撤回', 'success')
    await refresh()
  }
}
</script>

<template>
  <div class="mx-auto flex max-w-3xl flex-col gap-3">
    <KunCard :is-hoverable="false" :is-transparent="false">
      <KunHeader
        name="我的资料编辑提案"
        description="您提交的 Galgame 资料修改提案与它们的审核结果"
        scale="h2"
      />
    </KunCard>

    <div v-if="status === 'pending'" class="flex justify-center py-8">
      <KunLoading />
    </div>

    <KunNull
      v-else-if="!data?.items.length"
      description="您还没有提交过资料编辑提案"
    />

    <div v-else class="space-y-3">
      <EditkitProposalCard
        v-for="item in data.items"
        :key="item.id"
        :proposal="item"
        :label-for="galgameEditLabel"
        :decider="
          item.decided_by_uid !== undefined
            ? data.users?.[item.decided_by_uid]
            : undefined
        "
      >
        <template #title>
          <KunLink
            :to="`/galgame/${item.gid}`"
            size="sm"
            class-name="font-medium"
          >
            {{ briefName(item) }}
          </KunLink>
        </template>
        <template #proposer><span /></template>
        <template #actions>
          <KunButton
            v-if="item.status === 'open'"
            variant="flat"
            color="danger"
            size="sm"
            :loading="withdrawing"
            @click="handleWithdraw(item.id)"
          >
            撤回
          </KunButton>
        </template>
      </EditkitProposalCard>
    </div>
  </div>
</template>
