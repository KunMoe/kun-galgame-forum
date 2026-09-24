<script setup lang="ts">
import { galgameEditLabel } from '~/constants/galgameEdit'
import { problemMessage } from '#shared/utils/api/message'
import {
  proposalUsers,
  toKitProposal,
  type EditProposalSummary
} from '~/utils/galgame/editAdapt'
import { patchEditProposal } from '~/utils/galgame/editProposal'

useKunDisableSeo('我的资料编辑提案')

const api = useApiClient()
const nameOf = useWorkName()

const { data, problem, status, refresh } = await useApi<{
  items: EditProposalSummary[]
  next_cursor?: string
}>('me-edit-proposals', (client, { signal }) =>
  client.GET('/me/edit-proposals', { params: { query: { limit: 50 } }, signal })
)
const { items, hasMore, loadingMore, loadMore } = useEditProposalPages(
  data,
  (cursor) =>
    api.GET('/me/edit-proposals', { params: { query: { limit: 50, cursor } } })
)
const users = computed(() => proposalUsers(items.value))

const withdrawing = ref(false)
const handleWithdraw = async (id: string) => {
  if (withdrawing.value) {
    return
  }
  withdrawing.value = true
  const result = await patchEditProposal(api, id, { state: 'withdrawn' })
  withdrawing.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('提案已撤回', 'success')
  await refresh()
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

    <KunNull v-else-if="problem" :description="problemMessage(problem)" />

    <KunNull
      v-else-if="!items.length"
      description="您还没有提交过资料编辑提案"
    />

    <div v-else class="space-y-3">
      <EditkitProposalCard
        v-for="item in items"
        :key="item.id"
        :proposal="toKitProposal(item)"
        :label-for="galgameEditLabel"
        :decider="item.decider ? users[Number(item.decider.id)] : undefined"
      >
        <template #title>
          <KunLink
            :to="`/galgame/${item.work_id}`"
            size="sm"
            class-name="font-medium"
          >
            {{ nameOf(item.work_summary) || `Galgame #${item.work_id}` }}
          </KunLink>
        </template>
        <template #proposer><span /></template>
        <template #actions>
          <KunButton
            v-if="item.viewer?.can_withdraw"
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
      <div v-if="hasMore" class="flex justify-center">
        <KunButton variant="light" :loading="loadingMore" @click="loadMore">
          加载更多
        </KunButton>
      </div>
    </div>
  </div>
</template>
