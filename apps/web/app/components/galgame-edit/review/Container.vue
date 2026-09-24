<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import { galgameEditLabel } from '~/constants/galgameEdit'
import { problemMessage } from '#shared/utils/api/message'
import {
  proposalUsers,
  toKitProposal,
  type EditProposalSummary
} from '~/utils/galgame/editAdapt'

type QueueState = 'open' | 'merged' | 'declined' | 'withdrawn'

const canReview = useCan('galgame.edit_proposal.review')
const api = useApiClient()
const nameOf = useWorkName()

useKunDisableSeo('Galgame 提案审核队列')

const state = ref<QueueState>('open')

const { data, problem, status: fetchStatus } = await useApi<{
  items: EditProposalSummary[]
  next_cursor?: string
}>(
  () => `edit-proposal-queue:${state.value}`,
  (client, { signal }) =>
    client.GET('/edit-proposals', {
      params: { query: { state: state.value, limit: 50 } },
      signal
    }),
  { immediate: canReview.value }
)
const { items, hasMore, loadingMore, loadMore } = useEditProposalPages(
  data,
  (cursor) =>
    api.GET('/edit-proposals', {
      params: { query: { state: state.value, limit: 50, cursor } }
    })
)
const kitItems = computed(() => items.value.map(toKitProposal))
const byId = computed(
  () => new Map(items.value.map((item) => [Number(item.id), item]))
)
const users = computed(() => proposalUsers(items.value))

const titleOf = (id: number): string => {
  const item = byId.value.get(id)
  if (!item) {
    return `Galgame #${id}`
  }
  return nameOf(item.work_summary) || `Galgame #${item.work_id}`
}

const stateTabs: KunTabItem[] = [
  { value: 'open', textValue: '待审核' },
  { value: 'merged', textValue: '已合并' },
  { value: 'declined', textValue: '已拒绝' },
  { value: 'withdrawn', textValue: '已撤回' }
]
const onTab = (next: string) => {
  state.value = next as QueueState
}
</script>

<template>
  <div class="mx-auto flex max-w-3xl flex-col gap-3">
    <KunCard :is-hoverable="false" :is-transparent="false">
      <KunHeader
        name="Galgame 提案审核队列"
        description="社区提交的资料修改提案。审核时可直接修正提案内容再合并（Google Docs 建议模式的心智）——提案人与审核人都会署名"
        scale="h2"
      />
    </KunCard>

    <KunNull v-if="!canReview" description="需要审核资料编辑的权限" />

    <KunCard
      v-else
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <KunTab
        :model-value="state"
        :items="stateTabs"
        variant="light"
        size="sm"
        @update:model-value="onTab"
      />
      <div v-if="fetchStatus === 'pending'" class="flex justify-center py-8">
        <KunLoading />
      </div>
      <KunNull v-else-if="problem" :description="problemMessage(problem)" />
      <KunNull v-else-if="!kitItems.length" description="这里还没有提案" />
      <div v-else class="space-y-3">
        <EditkitProposalCard
          v-for="proposal in kitItems"
          :key="proposal.id"
          :proposal="proposal"
          :label-for="galgameEditLabel"
          :proposer="users[proposal.proposer_uid]"
          :decider="
            proposal.decided_by_uid !== undefined
              ? users[proposal.decided_by_uid]
              : undefined
          "
        >
          <template #title>
            <KunLink
              :to="`/galgame/${proposal.entity_id}`"
              size="sm"
              class-name="font-medium"
            >
              {{ titleOf(proposal.id) }}
            </KunLink>
          </template>
          <template #actions>
            <KunButton
              variant="flat"
              color="primary"
              size="sm"
              @click="navigateTo(`/galgame-edit/review/${proposal.id}`)"
            >
              {{ proposal.status === 'open' ? '审阅' : '查看' }}
            </KunButton>
          </template>
        </EditkitProposalCard>
      </div>
      <div v-if="hasMore" class="flex justify-center">
        <KunButton variant="light" :loading="loadingMore" @click="loadMore">
          加载更多
        </KunButton>
      </div>
    </KunCard>
  </div>
</template>
