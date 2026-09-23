<script setup lang="ts">
import { editValueEqual } from '@nextmoe/edit-ui-core'
import {
  createGalgameEditConfig,
  galgameEditLabel,
  type GalgameEditNames
} from '~/constants/galgameEdit'

const route = useRoute()
const proposalId = computed(() => parseInt((route.params as { id: string }).id))

useKunDisableSeo('审阅提案')

const { canModerate } = useRole()

const { data, status, refresh } = await useKunFetch<GalgameEditProposalDetail>(
  `/galgame-edit/proposals/${proposalId.value}`,
  { method: 'GET', watch: false }
)

const canDecide = computed(() => data.value?.can_decide ?? false)

const proposal = computed(() => data.value?.proposal)
const isOpen = computed(() => proposal.value?.status === 'open')
const exitTo = computed(() =>
  canModerate.value
    ? '/galgame-edit/review'
    : `/galgame/${proposal.value?.gid ?? ''}/edit`
)
const effective = computed(
  () => proposal.value?.effective_patch ?? proposal.value?.patch ?? {}
)
const fieldOf = (key: string) => data.value?.fields.find((f) => f.key === key)

const names = ref<GalgameEditNames>({})
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
  const pools = [data.value?.values?.[key], effective.value[key]]
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
  const workId = proposal.value?.gid
  if (!workId) {
    return
  }
  const detail = await kunFetch<GalgameDetail>(`/galgame/${workId}`, {
    method: 'GET'
  })
  const toMap = (arr?: { id: number; name: string }[]) =>
    new Map((arr ?? []).map((x) => [x.id, x.name]))
  const staff = new Map<number, string>()
  for (const group of detail?.staff ?? []) {
    for (const person of group.people) {
      if (!staff.has(person.id)) {
        staff.set(person.id, person.name)
      }
    }
  }
  const maps: Record<
    'tag' | 'official' | 'engine' | 'series' | 'character' | 'staff',
    Map<number, string>
  > = {
    tag: toMap(detail?.tag),
    official: toMap(detail?.official),
    engine: toMap(detail?.engine),
    series: toMap(detail?.series),
    character: toMap(detail?.characters),
    staff
  }
  const creditCharacterIds = (): number[] => {
    const out = new Set<number>()
    for (const pool of [
      data.value?.values?.['catalog.work.credits'],
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
      map: maps.tag,
      ids: relationIds('catalog.work.tag_ids'),
      path: 'galgame-tag'
    },
    {
      map: maps.official,
      ids: relationIds('catalog.work.labels'),
      path: 'galgame-official'
    },
    {
      map: maps.engine,
      ids: relationIds('catalog.work.engine_ids'),
      path: 'galgame-engine'
    },
    {
      map: maps.series,
      ids: relationIds('catalog.work.series_ids'),
      path: 'galgame-series'
    },
    {
      map: maps.character,
      ids: [...relationIds('catalog.work.roster'), ...creditCharacterIds()],
      path: 'galgame-character'
    },
    {
      map: maps.staff,
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

const handleMerge = async () => {
  if (acting.value || !proposal.value) {
    return
  }
  if (!remainingKeys.value.length) {
    useMessage('所有字段都被拒绝了——请直接拒绝这个提案', 'warn')
    return
  }
  acting.value = true
  if (hasAmendment.value) {
    const amended = await kunFetch<unknown>(
      `/galgame-edit/proposals/${proposalId.value}/amend`,
      {
        method: 'POST',
        body: {
          set: amendSet.value,
          unset: amendUnset.value,
          note: note.value
        }
      }
    )
    if (!amended) {
      acting.value = false
      return
    }
  }
  const merged = await kunFetch<unknown>(
    `/galgame-edit/proposals/${proposalId.value}/merge`,
    { method: 'POST', body: { note: note.value } }
  )
  acting.value = false
  if (merged) {
    useMessage(
      hasAmendment.value ? '已修正并合并（双方署名）' : '提案已合并',
      'success'
    )
    await navigateTo(exitTo.value)
  }
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
  const declined = await kunFetch<unknown>(
    `/galgame-edit/proposals/${proposalId.value}/decline`,
    { method: 'POST', body: { note: note.value } }
  )
  acting.value = false
  declineOpen.value = false
  if (declined) {
    useMessage('提案已拒绝', 'success')
    await navigateTo(exitTo.value)
  }
}

const userName = (uid?: number) => {
  if (uid === undefined) {
    return ''
  }
  return data.value?.users?.[uid]?.name ?? `用户 #${uid}`
}
</script>

<template>
  <div class="mx-auto flex max-w-3xl flex-col gap-3">
    <template v-if="data && proposal">
      <KunCard
        :is-hoverable="false"
        :is-transparent="false"
        content-class="space-y-2"
      >
        <KunHeader :name="`审阅提案 #${proposal.id}`" scale="h2" />
        <div class="flex flex-wrap items-center gap-2 text-sm">
          <KunLink :to="`/galgame/${proposal.gid}`" size="sm">
            前往条目 #{{ proposal.gid }}
          </KunLink>
          <span class="text-default-400">
            提案人：{{ userName(proposal.proposer_uid) }} ·
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
            {{ canModerate ? '返回队列' : '返回编辑页' }}
          </KunButton>
        </div>
        <KunInfo
          v-if="proposal.note"
          color="info"
          title="提案说明"
          :description="proposal.note"
        />
        <KunInfo
          v-if="proposal.status === 'declined' && proposal.decision_note"
          color="danger"
          title="拒绝理由"
          :description="proposal.decision_note"
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
            #{{ a.seq }} · {{ userName(a.amender_uid) }}
            <KunTime :time="a.created_at" type="date" show-year />
          </p>
          <div class="mt-1 flex flex-wrap gap-1">
            <KunChip
              v-for="key in Object.keys(a.set ?? {})"
              :key="`set-${key}`"
              size="sm"
              variant="flat"
              color="secondary"
            >
              修正 {{ galgameEditLabel(key) }}
            </KunChip>
            <KunChip
              v-for="key in a.unset ?? []"
              :key="`unset-${key}`"
              size="sm"
              variant="flat"
              color="danger"
            >
              拒绝 {{ galgameEditLabel(key) }}
            </KunChip>
          </div>
          <p v-if="a.note" class="text-default-400 mt-1 text-xs">
            {{ a.note }}
          </p>
        </div>
      </KunCard>

      <KunCard
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
            :from="data.values[String(key)]"
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
        v-if="isOpen && !canDecide"
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

    <KunNull
      v-else-if="status !== 'pending'"
      description="提案不存在或编辑服务暂不可用"
    />
  </div>
</template>
