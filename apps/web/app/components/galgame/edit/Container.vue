<script setup lang="ts">
import { parseEditProblem } from '@nextmoe/edit-ui-core'
import {
  createGalgameEditConfig,
  GALGAME_EDIT_GROUP_ORDER,
  GALGAME_EDIT_TABBED_GROUPS,
  galgameEditLabel,
  type GalgameEditNames
} from '~/constants/galgameEdit'

const route = useRoute()
const gid = computed(() => parseInt((route.params as { gid: string }).gid))

useKunDisableSeo('编辑 Galgame 资料')

const { data: bootstrap, status } = await useKunFetch<GalgameEditBootstrap>(
  `/galgame/${gid.value}/edit/bootstrap`,
  { method: 'GET', watch: false }
)

const { data: detail } = await useKunFetch<GalgameDetail>(
  `/galgame/${gid.value}`,
  { method: 'GET', watch: false }
)
const editNames = computed<GalgameEditNames>(() => {
  const d = detail.value
  const toMap = (arr?: { id: number; name: string }[]) =>
    new Map((arr ?? []).map((x) => [x.id, x.name]))
  const staff = new Map<number, string>()
  for (const group of d?.staff ?? []) {
    for (const person of group.people) {
      if (!staff.has(person.id)) {
        staff.set(person.id, person.name)
      }
    }
  }
  return {
    tag: toMap(d?.tag),
    official: toMap(d?.official),
    engine: toMap(d?.engine),
    series: toMap(d?.series),
    character: toMap(d?.characters),
    staff,
    covers: d?.covers,
    screenshots: d?.screenshots
  }
})
const editConfig = computed(() => createGalgameEditConfig(editNames.value))

const { data: mine, refresh: refreshMine } =
  await useKunFetch<GalgameEditProposalList>('/galgame-edit/mine', {
    method: 'GET',
    watch: false,
    query: { gid: gid.value }
  })

const { data: pending, refresh: refreshPending } =
  await useKunFetch<GalgameEditProposalList>(
    `/galgame/${gid.value}/edit/proposals`,
    { method: 'GET', watch: false }
  )

const userStore = usePersistUserStore()
const reviewable = computed(() => {
  if (!bootstrap.value?.can_review) {
    return []
  }
  return (pending.value?.items ?? []).filter(
    (p) => p.status === 'open' && p.proposer_uid !== userStore.id
  )
})

const { canModerate } = useRole()

const gameName = computed(() =>
  String(bootstrap.value?.values['catalog.work.display_name'] ?? '')
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

const automergeByKey = computed(() => {
  const map: Record<string, boolean> = {}
  for (const field of bootstrap.value?.fields ?? []) {
    map[field.key] = field.would_automerge
  }
  return map
})
const willAutomerge = computed(
  () =>
    dirtyCount.value > 0 &&
    Object.keys(patch.value).every((key) => automergeByKey.value[key])
)

const handleSubmit = async () => {
  if (!dirtyCount.value || submitting.value || !formValid.value) {
    return
  }
  submitting.value = true
  fieldErrors.value = {}
  formErrors.value = []
  const result = await kunFetch<GalgameEditSubmitResult>(
    `/galgame/${gid.value}/edit/proposals`,
    {
      method: 'POST',
      body: { patch: patch.value, note: note.value },
      onApiError: (envelope: {
        code: number
        message: string
        errors?: unknown[]
      }) => {
        const parsed = parseEditProblem({
          detail: envelope.message,
          errors: envelope.errors
        })
        fieldErrors.value = parsed.fields
        formErrors.value = parsed.form
        return Object.keys(parsed.fields).length > 0
      }
    }
  )
  submitting.value = false
  if (!result) {
    return
  }
  if (result.merged) {
    useMessage('修改已生效', 'success')
  } else {
    useMessage('提案已提交，等待审核', 'success')
  }
  formRef.value?.reset()
  patch.value = {}
  note.value = ''
  await Promise.all([refreshMine(), refreshPending()])
}

const withdrawing = ref(false)
const handleWithdraw = async (id: number) => {
  if (withdrawing.value) {
    return
  }
  withdrawing.value = true
  const result = await kunFetch<GalgameEditProposal>(
    `/galgame-edit/proposals/${id}/withdraw`,
    { method: 'POST' }
  )
  withdrawing.value = false
  if (result) {
    useMessage('提案已撤回', 'success')
    await refreshMine()
  }
}
</script>

<template>
  <div class="flex w-full flex-col gap-3">
    <template v-if="bootstrap">
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
            @click="navigateTo(`/galgame/${gid}`)"
          >
            <KunIcon name="lucide:arrow-left" />
            返回游戏页
          </KunButton>
          <KunButton
            variant="light"
            color="default"
            size="sm"
            @click="navigateTo(`/galgame/${gid}/history`)"
          >
            <KunIcon name="lucide:history" />
            修订历史
          </KunButton>
          <KunButton
            v-if="canModerate"
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
          :proposal="item"
          :label-for="galgameEditLabel"
          :proposer="pending?.users?.[item.proposer_uid]"
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

      <div v-if="mine?.items.length" class="space-y-2">
        <EditkitProposalCard
          v-for="item in mine.items.filter((p) => p.status === 'open')"
          :key="item.id"
          :proposal="item"
          :label-for="galgameEditLabel"
        >
          <template #title>
            <span class="text-default-700 text-sm font-medium">
              我的待审提案 #{{ item.id }}
            </span>
          </template>
          <template #actions>
            <KunButton
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
          :fields="bootstrap.fields"
          :values="bootstrap.values"
          :config="editConfig"
          :vocabularies="bootstrap.vocabularies"
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
                <p>
                  已修改 {{ dirtyCount }} 个字段 ·
                  <span
                    :class="willAutomerge ? 'text-success' : 'text-warning'"
                  >
                    {{ willAutomerge ? '保存后立即生效' : '提交后需审核' }}
                  </span>
                </p>
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
                <KunIcon
                  :name="willAutomerge ? 'lucide:check' : 'lucide:send'"
                />
                {{ willAutomerge ? '保存修改' : '提交提案' }}
              </KunButton>
            </div>
          </div>
        </KunCard>
      </div>
    </template>

    <KunNull
      v-else-if="status !== 'pending'"
      description="无法加载编辑数据（条目不存在或编辑服务暂不可用）"
    />
  </div>
</template>
