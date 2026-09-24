<script setup lang="ts">
import { parseEditProblem } from '@nextmoe/edit-ui-core'
import {
  createGalgameEditConfig,
  GALGAME_EDIT_GROUP_ORDER,
  GALGAME_EDIT_TABBED_GROUPS,
  galgameEditLabel,
  type GalgameEditNames
} from '~/constants/galgameEdit'
import { problemMessage } from '#shared/utils/api/message'
import { settle } from '#shared/utils/api/problem'
import type { Work } from '#shared/utils/api/schemas'
import {
  proposalUsers,
  toKitField,
  toKitProposal,
  toKitVocabularies,
  type EditForm,
  type EditProposalSummary
} from '~/utils/galgame/editAdapt'
import { patchEditProposal } from '~/utils/galgame/editProposal'
import { mapsFromWork } from '~/utils/galgame/workEditNames'

const route = useRoute()
const workId = computed(() => (route.params as { id: string }).id)

useKunDisableSeo('编辑 Galgame 资料')

const api = useApiClient()
const submitKey = useIdempotencyKey()

const { data: form, problem, status } = await useApi<EditForm>(
  () => `work-edit-form:${workId.value}`,
  (client, { signal }) =>
    client.GET('/works/{work_id}/edit-form', {
      params: { path: { work_id: workId.value } },
      signal
    })
)
const fields = computed(() => (form.value?.fields ?? []).map(toKitField))
const vocabularies = computed(() =>
  form.value ? toKitVocabularies(form.value) : {}
)

const nameOf = useCatalogName()
const { data: detail } = await useApi<Work>(
  () => `work-edit:${workId.value}`,
  (client, { signal }) =>
    client.GET('/works/{work_id}', {
      params: {
        path: { work_id: workId.value },
        query: { include_nsfw: true }
      },
      signal
    })
)
const editNames = computed<GalgameEditNames>(() =>
  mapsFromWork(detail.value, nameOf)
)
const editConfig = computed(() => createGalgameEditConfig(editNames.value))

const { data: openList, refresh: refreshOpen } = await useApi<{
  items: EditProposalSummary[]
}>(
  () => `work-edit-open:${workId.value}`,
  (client, { signal }) =>
    client.GET('/works/{work_id}/edit-proposals', {
      params: {
        path: { work_id: workId.value },
        query: { state: 'open', limit: 100 }
      },
      signal
    })
)
const openItems = computed(() => openList.value?.items ?? [])
const reviewable = computed(() =>
  openItems.value.filter((p) => !p.viewer?.is_proposer && p.viewer?.can_decide)
)
const mine = computed(() => openItems.value.filter((p) => p.viewer?.is_proposer))
const users = computed(() => proposalUsers(openItems.value))

const canReviewQueue = useCan('galgame.edit_proposal.review')

const gameName = computed(() =>
  String(form.value?.field_values['catalog.work.display_name'] ?? '')
)

const patch = ref<Record<string, unknown>>({})
const note = ref('')
const showNote = ref(true)
const submitting = ref(false)
const formValid = ref(true)
const fieldErrors = ref<Record<string, string[]>>({})
const formErrors = ref<string[]>([])
// edit-ui 0.3.0 holds a beforeunload listener while the patch is non-empty. A
// merged submit leaves that patch in place, so 返回游戏页 asked "leave site?"
// about changes that were already saved — reset() is what clears it.
const formRef = ref<{ reset: () => void } | null>(null)

const dirtyCount = computed(() => Object.keys(patch.value).length)
const dirtyLabels = computed(() =>
  Object.keys(patch.value).map((key) => galgameEditLabel(key))
)

const handleSubmit = async () => {
  if (!dirtyCount.value || submitting.value || !formValid.value) {
    return
  }
  submitting.value = true
  fieldErrors.value = {}
  formErrors.value = []
  const body = { patch: patch.value, note: note.value || null }
  const result = await settle(
    api.POST('/works/{work_id}/edit-proposals', {
      params: {
        path: { work_id: workId.value },
        header: {
          'Idempotency-Key': submitKey.take(
            `/works/${workId.value}/edit-proposals`,
            body
          )
        }
      },
      body
    })
  )
  submitting.value = false
  if (!result.ok) {
    fieldErrors.value = parseEditProblem({ errors: result.problem.errors }).fields
    reportProblem(result.problem)
    return
  }
  submitKey.clear()
  useMessage(
    result.data.state === 'merged' ? '修改已生效' : '提案已提交，等待审核',
    'success'
  )
  formRef.value?.reset()
  patch.value = {}
  note.value = ''
  await refreshOpen()
}

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
  await refreshOpen()
}
</script>

<template>
  <div class="flex w-full flex-col gap-3">
    <template v-if="form">
      <KunCard
        :is-hoverable="false"
        :is-transparent="false"
        content-class="space-y-2"
      >
        <KunHeader
          :name="gameName ? `编辑资料 · ${gameName}` : '编辑资料'"
          description="修改字段后保存：拥有直接编辑权限（管理员 / 游戏创建者）时立即生效，否则进入审核队列，由审核人处理（审核人可在合并前修正您的提案，双方都会署名）"
          scale="h2"
        />
        <div class="flex flex-wrap gap-2">
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
            @click="navigateTo(`/galgame/${workId}/history`)"
          >
            <KunIcon name="lucide:history" />
            修订历史
          </KunButton>
          <KunButton
            v-if="canReviewQueue"
            variant="light"
            color="primary"
            size="sm"
            @click="navigateTo('/galgame-edit/review')"
          >
            <KunIcon name="lucide:list-checks" />
            审核队列
          </KunButton>
        </div>
      </KunCard>

      <div v-if="reviewable.length" class="space-y-2">
        <EditkitProposalCard
          v-for="item in reviewable"
          :key="item.id"
          :proposal="toKitProposal(item)"
          :label-for="galgameEditLabel"
          :proposer="users[Number(item.proposer.id)]"
        >
          <template #title>
            <span class="text-default-700 text-sm font-medium">
              待审提案 #{{ item.id }}
            </span>
          </template>
          <template #actions>
            <KunButton
              variant="flat"
              color="primary"
              size="sm"
              @click="navigateTo(`/galgame-edit/review/${item.id}`)"
            >
              审阅
            </KunButton>
          </template>
        </EditkitProposalCard>
      </div>

      <div v-if="mine.length" class="space-y-2">
        <EditkitProposalCard
          v-for="item in mine"
          :key="item.id"
          :proposal="toKitProposal(item)"
          :label-for="galgameEditLabel"
        >
          <template #title>
            <span class="text-default-700 text-sm font-medium">
              我的待审提案 #{{ item.id }}
            </span>
          </template>
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
      </div>

      <KunCard :is-hoverable="false" :is-transparent="false">
        <EditkitSchemaForm
          ref="formRef"
          :fields="fields"
          :values="form.field_values"
          :config="editConfig"
          :vocabularies="vocabularies"
          :group-order="GALGAME_EDIT_GROUP_ORDER"
          :tabbed-groups="GALGAME_EDIT_TABBED_GROUPS"
          :disabled="submitting"
          :errors="fieldErrors"
          :form-errors="formErrors"
          layout="tabs"
          @update:patch="patch = $event"
          @update:valid="formValid = $event"
        />
      </KunCard>

      <div class="sticky bottom-0 z-20 pb-3">
        <KunCard
          :is-hoverable="false"
          :is-transparent="false"
          class-name="bg-[oklch(var(--content1))]!"
          content-class="space-y-2"
        >
          <KunTextarea
            v-if="showNote"
            v-model="note"
            label="编辑说明（可选）"
            placeholder="说明这次修改的内容与依据，帮助审核人更快处理"
            :rows="2"
            :maxlength="2000"
          />
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="text-default-500 min-w-0 space-y-1 text-sm">
              <template v-if="dirtyCount">
                <p>已修改 {{ dirtyCount }} 个字段</p>
                <div class="flex flex-wrap gap-1">
                  <KunChip
                    v-for="name in dirtyLabels"
                    :key="name"
                    size="sm"
                    variant="flat"
                    color="warning"
                  >
                    {{ name }}
                  </KunChip>
                </div>
              </template>
              <p v-else>尚未修改</p>
              <p v-if="!formValid" class="text-danger">
                有字段填写不完整，修好后才能提交
              </p>
            </div>
            <div class="flex items-center gap-2">
              <KunButton
                variant="light"
                color="default"
                size="sm"
                @click="showNote = !showNote"
              >
                <KunIcon name="lucide:pencil-line" />
                编辑说明
              </KunButton>
              <KunButton
                color="primary"
                :disabled="!dirtyCount || !formValid"
                :loading="submitting"
                @click="handleSubmit"
              >
                <KunIcon name="lucide:send" />
                提交修改
              </KunButton>
            </div>
          </div>
        </KunCard>
      </div>
    </template>

    <KunNull v-else-if="problem" :description="problemMessage(problem)" />
  </div>
</template>
