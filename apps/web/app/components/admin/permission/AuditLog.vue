<script setup lang="ts">
import { KUN_PERMISSION_META } from '~/constants/permission'
import { KUN_USER_ROLE_MAP } from '~/constants/user'
import type { ForumPermission } from '~/composables/useCan'
import { settle } from '#shared/utils/api/problem'
import type { PermissionChange } from '#shared/utils/api/schemas'
import { deletedUserName, toKunUser } from '~/utils/userRef'

type ChangePage = { items: PermissionChange[]; total: number }

const api = useApiClient()
const page = ref(1)
const limit = 20
const data = ref<ChangePage | null>(null)
const loading = ref(false)

const load = async () => {
  loading.value = true
  const result = await settle(
    api.GET('/admin/permission-changes', {
      params: { query: { page: page.value, limit } }
    })
  )
  loading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  data.value = result.data
}

onMounted(load)
watch(page, load)

const totalPage = computed(() =>
  data.value ? Math.max(1, Math.ceil(data.value.total / limit)) : 1
)

const permLabel = (permission: string) =>
  KUN_PERMISSION_META[permission as ForumPermission]?.label ?? permission

const subjectLabel = (entry: PermissionChange): string => {
  if (entry.target_role) {
    return `角色 · ${KUN_USER_ROLE_MAP[entry.target_role] ?? entry.target_role}`
  }
  return `用户 · ${entry.target_user?.name ?? deletedUserName}`
}

const isReset = (entry: PermissionChange) => entry.after.length === 0

type RowDelta = { permission: string; kind: 'grant' | 'revoke' | 'removed' }

const rowDeltas = (entry: PermissionChange): RowDelta[] => {
  const before = new Map(entry.before.map((o) => [o.permission, o.effect]))
  const after = new Map(entry.after.map((o) => [o.permission, o.effect]))
  const keys = new Set([...before.keys(), ...after.keys()])
  const out: RowDelta[] = []
  for (const key of keys) {
    const from = before.get(key)
    const to = after.get(key)
    if (from === to) continue
    out.push({ permission: key, kind: to ?? 'removed' })
  }
  return out
}

const DELTA_CHIP: Record<
  RowDelta['kind'],
  { color: 'success' | 'warning' | 'default'; prefix: string }
> = {
  grant: { color: 'success', prefix: '+授予' },
  revoke: { color: 'warning', prefix: '+撤销' },
  removed: { color: 'default', prefix: '移除覆盖' }
}
</script>

<template>
  <div class="space-y-3">
    <KunLoading v-if="loading && !data" />
    <KunNull v-else-if="!data?.items.length" description="暂无权限变更记录" />

    <template v-else>
      <div
        v-for="entry in data.items"
        :key="entry.id"
        class="border-default-200 space-y-2 rounded-lg border p-3"
      >
        <div class="flex flex-wrap items-center gap-2">
          <KunChip size="sm" :color="isReset(entry) ? 'default' : 'primary'">
            {{ isReset(entry) ? '重置' : '调整' }}
          </KunChip>
          <span class="text-sm font-medium">{{ subjectLabel(entry) }}</span>
          <span class="text-default-400 text-xs">操作人</span>
          <KunUserChip :user="toKunUser(entry.actor)" size="sm" />
          <KunTime
            :time="entry.created_at"
            type="datetime"
            show-year
            class="ml-auto text-xs"
          />
        </div>

        <div v-if="rowDeltas(entry).length" class="flex flex-wrap gap-1.5">
          <KunChip
            v-for="delta in rowDeltas(entry)"
            :key="delta.permission"
            size="sm"
            variant="flat"
            :color="DELTA_CHIP[delta.kind].color"
          >
            {{ DELTA_CHIP[delta.kind].prefix }}
            {{ permLabel(delta.permission) }}
          </KunChip>
        </div>
        <span v-else class="text-default-400 text-xs">无覆盖变化</span>
      </div>

      <KunPagination
        v-if="data && data.total > limit"
        v-model:current-page="page"
        :total-page="totalPage"
        :is-loading="loading"
      />
    </template>
  </div>
</template>
