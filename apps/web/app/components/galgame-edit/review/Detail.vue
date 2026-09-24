<script setup lang="ts">
import { editValueEqual } from '@nextmoe/edit-ui-core'
import {
  createGalgameEditConfig,
  galgameEditLabel,
  type GalgameEditNames
} from '~/constants/galgameEdit'
import { problemMessage } from '#shared/utils/api/message'
import { settle } from '#shared/utils/api/problem'
import type { UserRef, Work } from '#shared/utils/api/schemas'
import {
  toKitField,
  type EditForm,
  type EditProposal
} from '~/utils/galgame/editAdapt'
import {
  amendEditProposal,
  patchEditProposal
} from '~/utils/galgame/editProposal'
import { mapsFromWork } from '~/utils/galgame/workEditNames'
import { toKunUser } from '~/utils/userRef'

const route = useRoute()
const proposalId = computed(() => (route.params as { id: string }).id)

useKunDisableSeo('审阅提案')

const canReviewQueue = useCan('galgame.edit_proposal.review')
const amendKey = useIdempotencyKey()

const { data, problem, refresh } = await useApi<{
  proposal: EditProposal
  etag: string | null
}>(
  () => `edit-proposal:${proposalId.value}`,
  async (client, { signal }) => {
    const res = await client.GET('/edit-proposals/{proposal_id}', {
      params: { path: { proposal_id: proposalId.value } },
      signal
    })
    return {
      ...res,
      data: res.data
        ? { proposal: res.data, etag: res.response.headers.get('ETag') }
        : undefined
    }
  }
)

const proposal = computed(() => data.value?.proposal)
const workId = computed(() => proposal.value?.work_id)
const { data: form, problem: formProblem } = await useApi<EditForm>(
  () => `work-edit-form:${workId.value ?? ''}`,
  (client, { signal }) =>
    client.GET('/works/{work_id}/edit-form', {
      params: { path: { work_id: workId.value ?? '' } },
      signal
    }),
  { immediate: !!workId.value }
)
const values = computed(() => form.value?.field_values ?? {})
const fields = computed(() => (form.value?.fields ?? []).map(toKitField))
const userName = (ref: UserRef) => toKunUser(ref).name

const canDecide = computed(
  () => !formProblem.value && (proposal.value?.viewer?.can_decide ?? false)
)
const isOpen = computed(() => proposal.value?.state === 'open')
const exitTo = computed(() =>
  canReviewQueue.value
    ? '/galgame-edit/review'
    : `/galgame/${proposal.value?.work_id ?? ''}/edit`
)
const effective = computed(
  () => proposal.value?.effective_patch ?? proposal.value?.patch ?? {}
)
const fieldOf = (key: string) => fields.value.find((f) => f.key === key)

const names = ref<GalgameEditNames>({})
const nameOf = useCatalogName()
const api = useApiClient()
const editConfig = computed(() => createGalgameEditConfig(names.value))
const configOf = (key: string) => editConfig.value[key]

const idsFrom = (key: string, el: unknown): number[] => {
  if (key === 'catalog.work.labels') {
    return [Number((el as { label_id?: unknown })?.label_id)]
  }
  if (key === 'catalog.work.roster') {
    return [Number((el as { character_id?: unknown })?.character_id)]
  }
  if (key === 'catalog.work.credits') {
    return [Number((el as { credit_name_id?: unknown })?.credit_name_id)]
  }
  return [Number(el)]
}

const relationIds = (key: string): number[] => {
  const pools = [values.value[key], effective.value[key]]
  const out = new Set<number>()
  for (const pool of pools) {
    if (!Array.isArray(pool)) {
      continue
    }
    for (const el of pool) {
      for (const id of idsFrom(key, el)) {
        if (Number.isFinite(id) && id > 0) {
          out.add(id)
        }
      }
    }
  }
  return [...out]
}

onMounted(async () => {
  if (!workId.value) {
    return
  }
  const result = await settle(
    api.GET('/works/{work_id}', {
      params: {
        path: { work_id: workId.value },
        query: { include_nsfw: true }
      }
    })
  )
  const detail: Work | undefined = result.ok ? result.data : undefined
  const maps = mapsFromWork(detail, nameOf)
  const creditCharacterIds = (): number[] => {
    const out = new Set<number>()
    for (const pool of [
      values.value['catalog.work.credits'],
      effective.value['catalog.work.credits']
    ]) {
      if (!Array.isArray(pool)) {
        continue
      }
      for (const el of pool) {
        const id = Number((el as { character_id?: unknown })?.character_id)
        if (Number.isFinite(id) && id > 0) {
          out.add(id)
        }
      }
    }
    return [...out]
  }

  const families = [
    {
      map: maps.tag ?? new Map(),
      ids: relationIds('catalog.work.tag_ids'),
      path: 'galgame-tag'
    },
    {
      map: maps.official ?? new Map(),
      ids: relationIds('catalog.work.labels'),
      path: 'galgame-official'
    },
    {
      map: maps.engine ?? new Map(),
      ids: relationIds('catalog.work.engine_ids'),
      path: 'galgame-engine'
    },
    {
      map: maps.series ?? new Map(),
      ids: relationIds('catalog.work.series_ids'),
      path: 'galgame-series'
    },
    {
      map: maps.character ?? new Map(),
      ids: [...relationIds('catalog.work.roster'), ...creditCharacterIds()],
      path: 'galgame-character'
    },
    {
      map: maps.staff ?? new Map(),
      ids: relationIds('catalog.work.credits'),
      path: 'galgame-staff'
    }
  ]
  await Promise.all(
    families.flatMap(({ map, ids, path }) =>
      ids
        .filter((id) => !map.has(id))
        .map(async (id) => {
          const hit = await kunFetch<{ id: number; name: string }>(
            `/${path}/${id}`,
            { method: 'GET' }
          )
          if (hit?.name) {
            map.set(id, hit.name)
          }
        })
    )
  )
  names.value = maps
})

const overrides = reactive<Record<string, unknown>>({})
const editing = reactive<Record<string, boolean>>({})
const rejected = reactive<Record<string, boolean>>({})

const startEdit = (key: string) => {
  if (!(key in overrides)) {
    overrides[key] = structuredClone(toRaw(effective.value[key]) ?? null)
  }
  editing[key] = true
  rejected[key] = false
}

const cancelEdit = (key: string) => {
  Reflect.deleteProperty(overrides, key)
  editing[key] = false
}

const toggleReject = (key: string) => {
  rejected[key] = !rejected[key]
  if (rejected[key]) {
    editing[key] = false
    Reflect.deleteProperty(overrides, key)
  }
}

const amendSet = computed<Record<string, unknown>>(() => {
  const out: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(overrides)) {
    if (editing[key] && !editValueEqual(value, effective.value[key])) {
      out[key] = value
    }
  }
  return out
})
const amendUnset = computed(() =>
  Object.keys(rejected).filter((key) => rejected[key])
)
const hasAmendment = computed(
  () => Object.keys(amendSet.value).length > 0 || amendUnset.value.length > 0
)
const remainingKeys = computed(() =>
  Object.keys(effective.value).filter((key) => !rejected[key])
)

const note = ref('')
const acting = ref(false)
const amendedEtag = ref<string | null>(null)

const resetAmendment = () => {
  for (const record of [overrides, editing, rejected]) {
    for (const key of Object.keys(record)) {
      Reflect.deleteProperty(record, key)
    }
  }
}

const handleMerge = async () => {
  if (acting.value || !proposal.value) {
    return
  }
  if (!remainingKeys.value.length) {
    useMessage('所有字段都被拒绝了——请直接拒绝这个提案', 'warn')
    return
  }
  acting.value = true
  const amending = hasAmendment.value
  if (amending) {
    const body = {
      set: amendSet.value,
      unset: amendUnset.value,
      note: note.value || null
    }
    const amended = await amendEditProposal(
      api,
      proposalId.value,
      body,
      amendedEtag.value ?? data.value?.etag ?? null,
      amendKey.take(`/edit-proposals/${proposalId.value}/amendments`, body)
    )
    if (!amended.result.ok) {
      acting.value = false
      reportProblem(amended.result.problem)
      return
    }
    amendKey.clear()
    resetAmendment()
    amendedEtag.value = amended.etag
    await refresh()
  }
  const merged = await patchEditProposal(
    api,
    proposalId.value,
    { state: 'merged', note: note.value || null },
    amendedEtag.value ?? data.value?.etag ?? null
  )
  acting.value = false
  if (!merged.ok) {
    reportProblem(merged.problem)
    return
  }
  useMessage(
    amending ? '已修正并合并（双方署名）' : '提案已合并',
    'success'
  )
  await navigateTo(exitTo.value)
}

const declineOpen = ref(false)
const handleDecline = async () => {
  if (acting.value || !note.value.trim()) {
    if (!note.value.trim()) {
      useMessage('请先在下方填写拒绝理由', 'warn')
    }
    return
  }
  acting.value = true
  const declined = await patchEditProposal(
    api,
    proposalId.value,
    { state: 'declined', note: note.value },
    amendedEtag.value ?? data.value?.etag ?? null
  )
  acting.value = false
  declineOpen.value = false
  if (!declined.ok) {
    reportProblem(declined.problem)
    return
  }
  useMessage('提案已拒绝', 'success')
  await navigateTo(exitTo.value)
}
</script>

<template>
  <div class="mx-auto flex max-w-3xl flex-col gap-3">
    <template v-if="proposal">
      <KunCard
        :is-hoverable="false"
        :is-transparent="false"
        content-class="space-y-2"
      >
        <KunHeader :name="`审阅提案 #${proposal.id}`" scale="h2" />
        <div class="flex flex-wrap items-center gap-2 text-sm">
          <KunLink :to="`/galgame/${proposal.work_id}`" size="sm">
            前往条目 #{{ proposal.work_id }}
          </KunLink>
          <span class="text-default-400">
            提案人：{{ userName(proposal.proposer) }} ·
            <KunTime :time="proposal.created_at" type="date" show-year />
          </span>
          <KunButton
            variant="light"
            color="default"
            size="sm"
            class-name="ml-auto"
            @click="navigateTo(exitTo)"
          >
            <KunIcon name="lucide:arrow-left" />
            {{ canReviewQueue ? '返回队列' : '返回编辑页' }}
          </KunButton>
        </div>
        <KunInfo
          v-if="proposal.note"
          color="info"
          title="提案说明"
          :description="proposal.note"
        />
      </KunCard>

      <KunCard
        v-if="proposal.amendments?.length"
        :is-hoverable="false"
        :is-transparent="false"
        content-class="space-y-2"
      >
        <KunHeader name="审核修正记录" scale="h3" />
        <div
          v-for="a in proposal.amendments"
          :key="a.id"
          class="border-default-200 rounded border p-2 text-sm"
        >
          <p class="text-default-500">
            #{{ a.seq }} · {{ userName(a.amender) }}
            <KunTime :time="a.created_at" type="date" show-year />
          </p>
          <p v-if="a.note" class="text-default-400 mt-1 text-xs">
            {{ a.note }}
          </p>
        </div>
      </KunCard>

      <KunInfo
        v-if="formProblem"
        color="danger"
        title="无法读取条目的当前资料，暂时不能审阅或裁决"
        :description="problemMessage(formProblem)"
      />

      <KunCard
        v-else
        :is-hoverable="false"
        :is-transparent="false"
        content-class="space-y-5"
      >
        <KunHeader
          name="逐字段审阅"
          :description="
            isOpen
              ? '每个字段可原样接受、修正后接受、或单独拒绝——修正后合并将同时署名提案人与审核人'
              : '提案已关闭，以下为最终内容'
          "
          scale="h3"
        />

        <div
          v-for="(value, key) in effective"
          :key="key"
          class="space-y-2"
          :class="rejected[key] ? 'opacity-50' : ''"
        >
          <EditkitFieldDiff
            :label="galgameEditLabel(String(key))"
            :diff-hint="fieldOf(String(key))?.diff_hint"
            :from="values[String(key)]"
            :to="editing[String(key)] ? overrides[String(key)] : value"
            :config="configOf(String(key))"
          />

          <div
            v-if="isOpen && canDecide && fieldOf(String(key))?.can_review"
            class="flex flex-wrap items-center gap-2"
          >
            <template v-if="!editing[String(key)]">
              <KunButton
                variant="flat"
                color="secondary"
                size="sm"
                :disabled="rejected[String(key)]"
                @click="startEdit(String(key))"
              >
                <KunIcon name="lucide:pencil" />
                修正该值
              </KunButton>
            </template>
            <template v-else>
              <KunButton
                variant="flat"
                color="default"
                size="sm"
                @click="cancelEdit(String(key))"
              >
                取消修正
              </KunButton>
            </template>
            <KunButton
              variant="flat"
              :color="rejected[String(key)] ? 'default' : 'danger'"
              size="sm"
              @click="toggleReject(String(key))"
            >
              <KunIcon name="lucide:x" />
              {{ rejected[String(key)] ? '恢复该字段' : '拒绝该字段' }}
            </KunButton>
          </div>

          <div
            v-if="
              isOpen &&
              canDecide &&
              editing[String(key)] &&
              fieldOf(String(key))
            "
            class="border-secondary-200 rounded border p-3"
          >
            <EditkitSchemaField
              v-model="overrides[String(key)]"
              :field="fieldOf(String(key))!"
              :config="configOf(String(key))"
            />
          </div>
        </div>

        <KunNull
          v-if="!Object.keys(effective).length"
          description="提案的所有字段都已被移除"
        />
      </KunCard>

      <KunInfo
        v-if="isOpen && !canDecide && !formProblem"
        color="info"
        title="只读审阅"
        description="您可以查看此提案，但只有具备裁决权限的管理员（或该条目的创建者）可以合并、修正或拒绝。"
      />

      <KunCard
        v-if="isOpen && canDecide"
        :is-hoverable="false"
        :is-transparent="false"
        content-class="space-y-3"
      >
        <KunTextarea
          v-model="note"
          label="审核说明"
          placeholder="合并备注，或拒绝理由（拒绝时必填）"
          :maxlength="2000"
        />
        <div class="flex flex-wrap items-center justify-end gap-2">
          <span v-if="hasAmendment" class="text-secondary-600 text-sm">
            将先保存 {{ Object.keys(amendSet).length + amendUnset.length }}
            处修正，再合并
          </span>
          <KunButton
            variant="flat"
            color="danger"
            :loading="acting"
            @click="declineOpen = true"
          >
            拒绝提案
          </KunButton>
          <KunButton color="primary" :loading="acting" @click="handleMerge">
            {{ hasAmendment ? '修正并合并' : '合并提案' }}
          </KunButton>
        </div>
      </KunCard>

      <KunModal v-model="declineOpen">
        <div class="space-y-3">
          <KunHeader name="拒绝这个提案？" scale="h3" />
          <p class="text-default-500 text-sm">
            拒绝理由将展示给提案人：{{
              note || '（尚未填写，请返回填写审核说明）'
            }}
          </p>
          <div class="flex justify-end gap-2">
            <KunButton
              variant="flat"
              color="default"
              @click="declineOpen = false"
            >
              取消
            </KunButton>
            <KunButton color="danger" :loading="acting" @click="handleDecline">
              确认拒绝
            </KunButton>
          </div>
        </div>
      </KunModal>
    </template>

    <KunNull v-else-if="problem" :description="problemMessage(problem)" />
  </div>
</template>
