<script setup lang="ts">
import {
  KUN_PERMISSION_GROUPS,
  KUN_PERMISSION_META,
  KUN_PERM_ROLE_COLUMNS,
  type KunPermRoleColumn
} from '~/constants/permission'

const props = defineProps<{
  working: Record<string, Set<string>>
  baseline: Record<string, Set<string>>
  effective: Record<string, Set<string>>
  editable: Record<string, boolean>
  disabled?: boolean
}>()

const emit = defineEmits<{
  toggle: [role: string, permission: string, value: boolean]
  reset: [role: string]
}>()

const columns = KUN_PERM_ROLE_COLUMNS

const groups = KUN_PERMISSION_GROUPS

const myPerms = useMyPermissions()

const columnLocked = (col: KunPermRoleColumn) =>
  col.locked || !props.editable[col.role]

const possessionLocked = (col: KunPermRoleColumn, perm: string) =>
  col.editable && !columnLocked(col) && !myPerms.value(perm)

const cellDisabled = (col: KunPermRoleColumn, perm: string) =>
  columnLocked(col) || props.disabled || possessionLocked(col, perm)

const isChecked = (col: KunPermRoleColumn, perm: string) =>
  (columnLocked(col)
    ? props.effective[col.role]?.has(perm)
    : props.working[col.role]?.has(perm)) ?? false

const deviation = (
  col: KunPermRoleColumn,
  perm: string
): 'grant' | 'revoke' | null => {
  const shown = isChecked(col, perm)
  const inBase = props.baseline[col.role]?.has(perm) ?? false
  if (shown === inBase) return null
  return shown ? 'grant' : 'revoke'
}

const gridStyle =
  'grid-template-columns: minmax(11rem, 1fr) repeat(4, minmax(4.5rem, 7rem));'
</script>

<template>
  <div class="overflow-x-auto">
    <div class="grid min-w-[44rem]" :style="gridStyle">
      <div
        class="border-default-200 text-default-500 border-b px-3 py-2 text-sm font-medium"
      >
        权限
      </div>
      <div
        v-for="col in columns"
        :key="col.role"
        class="border-default-200 flex flex-col items-center gap-1 border-b px-2 py-2 text-center"
      >
        <div class="flex items-center gap-1">
          <span class="text-sm font-semibold">{{ col.label }}</span>
          <KunTooltip
            v-if="columnLocked(col)"
            :text="
              col.locked
                ? '莲角色恒持全部权限，不可调整'
                : '仅可编辑低于自身角色的权限'
            "
            position="top"
          >
            <KunIcon name="lucide:lock" class="text-default-400 text-xs" />
          </KunTooltip>
        </div>
        <KunButton
          v-if="col.editable && !columnLocked(col)"
          size="xs"
          variant="light"
          color="default"
          :disabled="disabled"
          @click="emit('reset', col.role)"
        >
          重置默认
        </KunButton>
      </div>

      <template v-for="entry in groups" :key="entry.group">
        <div
          class="bg-default-100/60 text-default-600 col-span-full px-3 py-1.5 text-xs font-semibold"
        >
          {{ entry.group }}
        </div>
        <template v-for="perm in entry.perms" :key="perm">
          <div
            class="border-default-100 flex items-center border-b px-3 py-2 text-sm"
          >
            {{ KUN_PERMISSION_META[perm].label }}
          </div>
          <div
            v-for="col in columns"
            :key="col.role"
            class="border-default-100 flex items-center justify-center border-b px-2 py-2"
          >
            <div class="relative">
              <KunTooltip
                v-if="possessionLocked(col, perm)"
                text="不可增删自己未持有的权限"
                position="top"
              >
                <KunCheckBox
                  :model-value="isChecked(col, perm)"
                  :disabled="true"
                  color="primary"
                />
              </KunTooltip>
              <KunCheckBox
                v-else
                :model-value="isChecked(col, perm)"
                :disabled="cellDisabled(col, perm)"
                color="primary"
                @change="(value) => emit('toggle', col.role, perm, value)"
              />
              <KunTooltip
                v-if="deviation(col, perm)"
                :text="
                  deviation(col, perm) === 'grant'
                    ? '已授予（高于基线）'
                    : '已撤销（低于基线）'
                "
                position="top"
              >
                <span
                  :class="
                    cn(
                      'absolute -top-1.5 -right-1.5 block size-2 rounded-full',
                      deviation(col, perm) === 'grant'
                        ? 'bg-success'
                        : 'bg-warning'
                    )
                  "
                />
              </KunTooltip>
            </div>
          </div>
        </template>
      </template>
    </div>
  </div>
</template>
