<script setup lang="ts">
import {
  KUN_PERMISSION_GROUPS,
  KUN_PERMISSION_KEYS,
  KUN_PERMISSION_META
} from '~/constants/permission'
import { KUN_USER_ROLE_MAP } from '~/constants/user'
import { settle } from '#shared/utils/api/problem'
import type {
  PermissionOverride,
  UserPermissions
} from '#shared/utils/api/schemas'

const props = defineProps<{ user: KunUser }>()
const open = defineModel<boolean>({ required: true })

const api = useApiClient()

const state = ref<UserPermissions | null>(null)
const loading = ref(false)
const saving = ref(false)

const working = ref<Set<string>>(new Set())
const initial = ref<Set<string>>(new Set())

const isRen = computed(() => state.value?.is_locked ?? false)

const myPerms = useMyPermissions()
const rankLocked = computed(
  () => !isRen.value && !(state.value?.viewer.can_edit ?? false)
)
const readOnly = computed(() => isRen.value || rankLocked.value)

const possessionLocked = (permission: string) =>
  !readOnly.value && !myPerms.value(permission)
const cellDisabled = (permission: string) =>
  readOnly.value || saving.value || possessionLocked(permission)

const roleEffective = computed(() => new Set(state.value?.baseline ?? []))

const groups = KUN_PERMISSION_GROUPS

const applyState = (s: UserPermissions) => {
  state.value = s
  working.value = new Set(s.effective)
  initial.value = new Set(s.effective)
}

const userPath = () => ({
  params: { path: { user_id: String(props.user.id) } }
})

const load = async () => {
  loading.value = true
  const result = await settle(
    api.GET('/admin/user-permissions/{user_id}', userPath())
  )
  loading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  applyState(result.data)
}

watch(open, (isOpen) => {
  if (isOpen) {
    state.value = null
    working.value = new Set()
    initial.value = new Set()
    load()
  }
})

const toggle = (permission: string, value: boolean) => {
  const next = new Set(working.value)
  if (value) {
    next.add(permission)
  } else {
    next.delete(permission)
  }
  working.value = next
}

const isChecked = (permission: string) => working.value.has(permission)

const deviation = (permission: string): 'grant' | 'revoke' | null => {
  const inWork = working.value.has(permission)
  const inRole = roleEffective.value.has(permission)
  if (inWork === inRole) return null
  return inWork ? 'grant' : 'revoke'
}

const deltas = computed(() => {
  const role = roleEffective.value
  const out: PermissionOverride[] = []
  for (const key of KUN_PERMISSION_KEYS) {
    const inWork = working.value.has(key)
    const inRole = role.has(key)
    if (inWork && !inRole) {
      out.push({ permission: key, effect: 'grant' })
    } else if (!inWork && inRole) {
      out.push({ permission: key, effect: 'revoke' })
    }
  }
  return out
})

const isDirty = computed(() => {
  if (working.value.size !== initial.value.size) return true
  for (const key of working.value) {
    if (!initial.value.has(key)) return true
  }
  return false
})

const hasOverrides = computed(() => (state.value?.overrides.length ?? 0) > 0)

const replaceOverrides = async (overrides: PermissionOverride[]) => {
  saving.value = true
  const result = await settle(
    api.PUT('/admin/user-permissions/{user_id}', {
      ...userPath(),
      body: { overrides }
    })
  )
  saving.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  applyState(result.data)
  return true
}

const roleChips = computed(() =>
  (state.value?.roles ?? []).map((role) => KUN_USER_ROLE_MAP[role] ?? role)
)

const handleDiscard = () => {
  working.value = new Set(initial.value)
}

const handleSave = async () => {
  if (!isDirty.value || saving.value || readOnly.value) {
    return
  }
  const ok = await useComponentMessageStore().alert(
    `调整「${props.user.name}」的权限`,
    '此操作会立即改变该用户的实际权限，请确认无误后再保存。'
  )
  if (ok && (await replaceOverrides(deltas.value))) {
    useMessage('已保存', 'success')
  }
}

const handleReset = async () => {
  if (!hasOverrides.value || saving.value || readOnly.value) {
    return
  }
  const ok = await useComponentMessageStore().alert(
    `重置「${props.user.name}」的权限为默认`,
    '将清除该用户的全部个人权限覆盖，恢复到其角色本身的权限。此操作会立即生效，请确认。'
  )
  if (ok && (await replaceOverrides([]))) {
    useMessage('已重置为默认', 'success')
  }
}
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-2xl w-[94vw]">
    <div class="flex max-h-[82dvh] flex-col gap-3">
      <header class="flex flex-wrap items-center gap-2">
        <h2 class="text-xl font-bold">权限调整</h2>
        <KunUserChip :user="user" size="sm" />
      </header>

      <div class="flex flex-wrap items-center gap-2 text-sm">
        <span class="text-default-500">角色:</span>
        <template v-if="roleChips.length">
          <KunChip
            v-for="label in roleChips"
            :key="label"
            size="sm"
            variant="flat"
            color="secondary"
          >
            {{ label }}
          </KunChip>
        </template>
        <span v-else class="text-default-400">无角色</span>
      </div>

      <KunLoading v-if="loading" />

      <template v-else-if="state">
        <KunInfo
          v-if="isRen"
          color="warning"
          title="莲权限不可调整"
          description="莲（ren）恒持全部权限，不可调整。"
        />
        <KunInfo
          v-else-if="rankLocked"
          color="warning"
          title="无法调整该用户"
          description="仅可调整低于自身角色的用户。"
        />
        <KunInfo
          v-else
          color="info"
          description="勾选 = 在角色之外额外授予该用户一项权限；取消勾选 = 撤销一项其角色本有的权限。带有小圆点的行即为相对角色权限的个人覆盖（绿色=个人授予，黄色=个人撤销）。"
        />

        <div class="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
          <div v-for="entry in groups" :key="entry.group">
            <div
              class="bg-default-100/60 text-default-600 rounded-md px-2 py-1 text-xs font-semibold"
            >
              {{ entry.group }}
            </div>
            <div
              v-for="perm in entry.perms"
              :key="perm"
              class="border-default-100 flex items-center gap-2 border-b px-2 py-2"
            >
              <div class="relative shrink-0">
                <KunTooltip
                  v-if="possessionLocked(perm)"
                  text="不可增删自己未持有的权限"
                  position="top"
                >
                  <KunCheckBox
                    :model-value="isChecked(perm)"
                    :disabled="true"
                    color="primary"
                  />
                </KunTooltip>
                <KunCheckBox
                  v-else
                  :model-value="isChecked(perm)"
                  :disabled="cellDisabled(perm)"
                  color="primary"
                  @change="(value: boolean) => toggle(perm, value)"
                />
                <KunTooltip
                  v-if="deviation(perm)"
                  :text="
                    deviation(perm) === 'grant'
                      ? '个人授予（高于角色）'
                      : '个人撤销（低于角色）'
                  "
                  position="top"
                >
                  <span
                    :class="
                      cn(
                        'absolute -top-1.5 -right-1.5 block size-2 rounded-full',
                        deviation(perm) === 'grant'
                          ? 'bg-success'
                          : 'bg-warning'
                      )
                    "
                  />
                </KunTooltip>
              </div>
              <span class="text-sm">{{ KUN_PERMISSION_META[perm].label }}</span>
            </div>
          </div>
        </div>

        <div
          v-if="!readOnly"
          class="border-default-200 flex flex-wrap items-center justify-between gap-3 border-t pt-3"
        >
          <span class="text-default-500 text-sm">
            <template v-if="isDirty">
              待保存
              <span class="text-warning font-medium">{{ deltas.length }}</span>
              项个人覆盖
            </template>
            <template v-else-if="hasOverrides">
              当前有
              <span class="text-primary font-medium">{{
                state.overrides.length
              }}</span>
              项个人覆盖
            </template>
            <template v-else>该用户暂无个人权限覆盖</template>
          </span>
          <div class="flex flex-wrap items-center gap-2">
            <KunButton
              variant="light"
              color="default"
              :disabled="!isDirty || saving"
              @click="handleDiscard"
            >
              放弃更改
            </KunButton>
            <KunButton
              variant="light"
              color="danger"
              :disabled="!hasOverrides || saving"
              @click="handleReset"
            >
              重置为默认
            </KunButton>
            <KunButton
              color="primary"
              :disabled="!isDirty"
              :loading="saving"
              @click="handleSave"
            >
              <KunIcon name="lucide:check" />
              保存
            </KunButton>
          </div>
        </div>
      </template>

      <KunNull v-else description="无法加载该用户的权限，请稍后重试" />
    </div>
  </KunModal>
</template>
