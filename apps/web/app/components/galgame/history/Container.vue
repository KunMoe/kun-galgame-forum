<script setup lang="ts">
import {
  GALGAME_EDIT_LEGACY_ACTION_LABELS,
  galgameEditFieldConfig,
  galgameEditLabel
} from '~/constants/galgameEdit'

const route = useRoute()
const workId = computed(() => parseInt((route.params as { id: string }).id))

useKunDisableSeo('Galgame 修订历史')

const { data, status, refresh } = await useKunFetch<GalgameEditRevisionList>(
  `/galgame/${workId.value}/edit/revisions`,
  { method: 'GET', watch: false, query: { limit: 200 } }
)

const diffOpen = ref(false)
const diffLoading = ref(false)
const diff = ref<GalgameEditDiff | null>(null)

const handleDiff = async (fromSeq: number, toSeq: number) => {
  diffLoading.value = true
  diffOpen.value = true
  diff.value = await kunFetch<GalgameEditDiff>(
    `/galgame/${workId.value}/edit/diff`,
    { method: 'GET', query: { from: fromSeq, to: toSeq } }
  )
  diffLoading.value = false
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
  const result = await kunFetch<GalgameEditRevertResult>(
    `/galgame/${workId.value}/edit/revert`,
    { method: 'POST', body: { to_seq: revertTarget.value } }
  )
  reverting.value = false
  revertTarget.value = null
  if (result) {
    useMessage('回滚成功（已生成一条新的修订记录）', 'success')
    await refresh()
  }
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
        :items="data.items"
        :users="data.users"
        :label-for="galgameEditLabel"
        :legacy-action-labels="GALGAME_EDIT_LEGACY_ACTION_LABELS"
        @diff="handleDiff"
      >
        <template #actions="{ revision }">
          <KunButton
            v-if="data.can_revert && revision.seq < latestSeq"
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
          <KunNull v-if="!diff.fields.length" description="两个版本没有差异" />
          <div v-else class="space-y-4">
            <EditkitFieldDiff
              v-for="row in diff.fields"
              :key="row.key"
              :label="galgameEditLabel(row.key)"
              :diff-hint="row.diff_hint"
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
