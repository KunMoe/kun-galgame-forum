<script setup lang="ts">
import {
  GALGAME_EDIT_LEGACY_ACTION_LABELS,
  galgameEditFieldConfig,
  galgameEditLabel
} from '~/constants/galgameEdit'
import { settle } from '#shared/utils/api/problem'
import {
  kitUsers,
  toKitRevision,
  type EditRevision,
  type EditRevisionDiff
} from '~/utils/galgame/editAdapt'

const route = useRoute()
const workId = computed(() => (route.params as { id: string }).id)

useKunDisableSeo('Galgame 修订历史')

const api = useApiClient()
const revertKey = useIdempotencyKey()
const userStore = usePersistUserStore()

const { data, status, refresh } = await useApi<{ items: EditRevision[] }>(
  () => `work-edit-revisions:${workId.value}`,
  (client, { signal }) =>
    client.GET('/works/{work_id}/edit-revisions', {
      params: { path: { work_id: workId.value }, query: { limit: 100 } },
      signal
    })
)
const revisions = computed(() => (data.value?.items ?? []).map(toKitRevision))
const users = computed(() =>
  kitUsers(
    (data.value?.items ?? []).flatMap((r) => [r.actor, r.last_amender])
  )
)

const diffOpen = ref(false)
const diffLoading = ref(false)
const diff = ref<EditRevisionDiff | null>(null)

const handleDiff = async (fromSeq: number, toSeq: number) => {
  diffLoading.value = true
  diffOpen.value = true
  const result = await settle(
    api.GET('/works/{work_id}/edit-revisions/diff', {
      params: {
        path: { work_id: workId.value },
        query: { from_seq: fromSeq, to_seq: toSeq }
      }
    })
  )
  diffLoading.value = false
  if (!result.ok) {
    diffOpen.value = false
    reportProblem(result.problem)
    return
  }
  diff.value = result.data
}

const latestSeq = computed(() => data.value?.items?.[0]?.seq ?? 0)
const revertTarget = ref<number | null>(null)
const revertOpen = computed({
  get: () => revertTarget.value !== null,
  set: (open: boolean) => {
    if (!open) {
      revertTarget.value = null
    }
  }
})
const reverting = ref(false)

const handleRevert = async () => {
  if (revertTarget.value === null || reverting.value) {
    return
  }
  reverting.value = true
  const body = { to_seq: revertTarget.value }
  const result = await settle(
    api.POST('/works/{work_id}/edit-reverts', {
      params: {
        path: { work_id: workId.value },
        header: {
          'Idempotency-Key': revertKey.take(
            `/works/${workId.value}/edit-reverts`,
            body
          )
        }
      },
      body
    })
  )
  reverting.value = false
  revertTarget.value = null
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  revertKey.clear()
  useMessage(
    result.data.revision
      ? '回滚成功（已生成一条新的修订记录）'
      : '回滚提案已提交，等待审核',
    'success'
  )
  await refresh()
}
</script>

<template>
  <div class="mx-auto flex max-w-3xl flex-col gap-3">
    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-2"
    >
      <KunHeader
        name="修订历史"
        description="该 Galgame 的完整编辑记录（含旧版百科迁移的历史修订），可任选两个版本对比差异"
        scale="h2"
      />
      <div class="flex gap-2">
        <KunButton
          variant="light"
          color="default"
          size="sm"
          @click="navigateTo(`/galgame/${workId}`)"
        >
          <KunIcon name="lucide:arrow-left" />
          返回游戏页
        </KunButton>
        <KunButton
          variant="light"
          color="default"
          size="sm"
          @click="navigateTo(`/galgame/${workId}/edit`)"
        >
          <KunIcon name="lucide:pencil" />
          编辑资料
        </KunButton>
      </div>
    </KunCard>

    <KunCard
      v-if="data"
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <EditkitRevisionTimeline
        :items="revisions"
        :users="users"
        :label-for="galgameEditLabel"
        :legacy-action-labels="GALGAME_EDIT_LEGACY_ACTION_LABELS"
        @diff="handleDiff"
      >
        <template #actions="{ revision }">
          <KunButton
            v-if="userStore.id && revision.seq < latestSeq"
            variant="light"
            color="warning"
            size="sm"
            @click="revertTarget = revision.seq"
          >
            <KunIcon name="lucide:undo-2" />
            回滚到此版本
          </KunButton>
        </template>
      </EditkitRevisionTimeline>
    </KunCard>

    <KunNull
      v-else-if="status !== 'pending'"
      description="无法加载修订历史（条目不存在或编辑服务暂不可用）"
    />

    <KunModal v-model="revertOpen">
      <div class="space-y-4">
        <KunHeader
          :name="`回滚到版本 #${revertTarget}`"
          description="回滚会把条目恢复到该版本的字段值，并生成一条新的修订记录（历史不会被删除）"
          scale="h3"
        />
        <div class="flex justify-end gap-2">
          <KunButton
            variant="light"
            color="default"
            @click="revertTarget = null"
          >
            取消
          </KunButton>
          <KunButton color="warning" :loading="reverting" @click="handleRevert">
            确认回滚
          </KunButton>
        </div>
      </div>
    </KunModal>

    <KunModal v-model="diffOpen" size="lg">
      <div class="space-y-4">
        <KunHeader
          :name="
            diff ? `版本对比 #${diff.from_seq} → #${diff.to_seq}` : '版本对比'
          "
          scale="h3"
        />
        <KunLoading v-if="diffLoading" />
        <template v-else-if="diff">
          <KunNull
            v-if="!diff.field_changes.length"
            description="两个版本没有差异"
          />
          <div v-else class="space-y-4">
            <EditkitFieldDiff
              v-for="row in diff.field_changes"
              :key="row.key"
              :label="galgameEditLabel(row.key)"
              :from="row.from"
              :to="row.to"
              :config="galgameEditFieldConfig(row.key)"
            />
          </div>
        </template>
      </div>
    </KunModal>
  </div>
</template>
