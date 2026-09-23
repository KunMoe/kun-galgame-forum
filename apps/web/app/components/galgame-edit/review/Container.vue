<script setup lang="ts">
import { galgameEditLabel } from '~/constants/galgameEdit'

const { canModerate } = useRole()

useKunDisableSeo('Galgame 提案审核队列')

const status = ref('open')

const { data, status: fetchStatus } =
  await useKunFetch<GalgameEditProposalList>('/galgame-edit/queue', {
    method: 'GET',
    query: { status }
  })

const briefName = (item: GalgameEditProposalItem): string => {
  if (!item.galgame) {
    return `Galgame #${item.gid}`
  }
  return item.galgame.name || `Galgame #${item.gid}`
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

    <KunNull v-if="!canModerate" description="需要管理权限" />

    <KunCard
      v-else
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <EditkitReviewQueue
        v-model:status="status"
        :items="data?.items ?? []"
        :users="data?.users"
        :label-for="galgameEditLabel"
        :loading="fetchStatus === 'pending'"
      >
        <template #item="{ proposal }">
          <EditkitProposalCard
            :proposal="proposal"
            :label-for="galgameEditLabel"
            :proposer="data?.users?.[proposal.proposer_uid]"
            :decider="
              proposal.decided_by_uid !== undefined
                ? data?.users?.[proposal.decided_by_uid]
                : undefined
            "
          >
            <template #title>
              <KunLink
                :to="`/galgame/${(proposal as GalgameEditProposalItem).gid}`"
                size="sm"
                class-name="font-medium"
              >
                {{ briefName(proposal as GalgameEditProposalItem) }}
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
        </template>
      </EditkitReviewQueue>
    </KunCard>
  </div>
</template>
